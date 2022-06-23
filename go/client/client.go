package client

import (
	"context"
	"fmt"
	"net/url"

	"sync"
	"time"

	control "github.com/longbridgeapp/openapi-protobufs/gen/go/control"
	protocol "github.com/longbridgeapp/openapi-protocol/go"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

var (
	ErrSessExpired     = errors.New("session expired")
	ErrHitMaxReconnect = errors.New("hit max reconnect count")

	errConnClosed = errors.New("client conn closed")
)

// Request represents an socket request to server
type Request struct {
	Cmd  uint32
	Body proto.Message
}

// New returns a new client instance
func New(opts ...ClientOption) *Client {
	c := &Client{
		closeCh: make(chan struct{}),
		subs:    make(map[uint32][]func(*protocol.Packet)),
		recvs:   make(map[uint32]chan *protocol.Packet),
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.Context == nil {
		c.Context = context.Background()
	}

	if c.Logger == nil {
		c.Logger = &protocol.DefaultLogger{}
	}
	return c
}

// Client is an socket client
type Client struct {
	sync.RWMutex

	Context context.Context
	Logger  protocol.Logger

	conn ClientConn

	closeCh chan struct{}

	// custom ping packet handler
	onPing func(*protocol.Packet)
	// custom pong packet handler
	onPong func(*protocol.Packet)

	// handle client close event
	onClose func(err error)

	afterReconnected func()

	authInfo  *control.AuthResponse
	handshake *protocol.Handshake

	subs map[uint32][]func(*protocol.Packet)

	recvsMu sync.RWMutex
	recvs   map[uint32]chan *protocol.Packet

	lastKeepaliveId uint32
	lastPongAt      time.Time
	reconnectCount  int
	doReconnectting bool

	addr        *url.URL
	dialOptions *DialOptions
}

// Dial using to dial with server
func (c *Client) Dial(ctx context.Context, u string, handshake *protocol.Handshake, opts ...DialOption) error {
	c.handshake = handshake

	uri, err := url.Parse(u)

	if err != nil {
		return errors.Wrap(err, "parse dial url")
	}

	c.addr = uri

	dialer, ok := GetDialer(uri.Scheme)

	if !ok {
		return errors.Errorf("dialer for scheme %s not exists", uri.Scheme)
	}

	dopts := newDialOptions(opts...)

	c.dialOptions = dopts

	c.Logger.Debug("get conn")
	if err = c.dial(ctx, dialer); err != nil {
		return err
	}

	c.Logger.Debug("got conn")

	if c.dialOptions.Keepalive != 0 {
		go c.keepalive()
	}

	return c.auth()
}

func (c *Client) AuthInfo() *control.AuthResponse {
	return c.authInfo
}

func (c *Client) dial(ctx context.Context, dialer DialConnFunc) (err error) {
	if c.conn, err = dialer(ctx, c, c.addr, c.handshake, c.dialOptions); err == nil {
		c.conn.OnPacket(c.onPacket)
	}

	return
}

func (c *Client) auth() error {
	if c.dialOptions.AuthTokenGetter == nil {
		return nil
	}
	token, err := c.dialOptions.AuthTokenGetter()
	if err != nil {
		return err
	}
	res, err := c.Do(context.Background(), &Request{
		Cmd:  uint32(control.Command_CMD_AUTH),
		Body: &control.AuthRequest{Token: token},
	}, RequestTimeout(c.dialOptions.AuthTimeout))

	if err != nil {
		return errors.Wrap(err, "do auth")
	}

	if e := res.Err(); e != nil {
		return fmt.Errorf("code: %d message: %s", e.Code, e.Msg)
	}

	var info control.AuthResponse

	if err = res.Unmarshal(&info); err != nil {
		return errors.Wrap(err, "auth unmarshal res")
	}

	c.authInfo = &info

	return nil
}

func (c *Client) reconnecting() {
	c.Lock()
	if c.doReconnectting {
		return
	}

	c.doReconnectting = true
	c.Unlock()

	waitCh := make(chan struct{})

	go func() {
		defer func() {
			waitCh <- struct{}{}
		}()

		for {

			c.Logger.Info("start reconnecting.")

			err := c.reconnect()

			if err == nil {
				c.Logger.Info("reconnect success")
				c.afterReconnected()
				return
			}

			if err == ErrHitMaxReconnect {
				c.Logger.Error("close client for hit max reconnect count")
				c.Close(err)
				return
			}

			c.Logger.Errorf("reconnect failed, err: %v", err)

			// TODO: dynamic value for sleepping
			time.Sleep(time.Second * 1)
		}
	}()

	<-waitCh

	c.Lock()
	c.doReconnectting = false
	c.Unlock()
}

func (c *Client) reconnect() error {
	if c.dialOptions.MaxReconnect > 0 {
		if c.reconnectCount >= c.dialOptions.MaxReconnect {
			return ErrHitMaxReconnect
		}
	}

	c.reconnectCount = c.reconnectCount + 1

	if c.conn != nil {
		c.conn.Close(errors.New("close old conn for reconnect"))
	}

	c.recvsMu.Lock()
	for _, ch := range c.recvs {
		close(ch)
	}
	c.recvs = make(map[uint32]chan *protocol.Packet)
	c.recvsMu.Unlock()

	dialer, _ := GetDialer(c.addr.Scheme)

	if err := c.dial(c.Context, dialer); err != nil {
		return err
	}

	// server needn't auth
	if c.authInfo == nil {
		return nil
	}

	// do auth
	if c.isAuthExpired() {
		return c.auth()
	}

	return c.reconnectDial()
}

func (c *Client) reconnectDial() error {
	res, err := c.Do(c.Context, &Request{Cmd: uint32(control.Command_CMD_RECONNECT), Body: &control.ReconnectRequest{
		SessionId: c.authInfo.SessionId,
	}}, RequestTimeout(c.dialOptions.AuthTimeout))

	if err != nil {
		return errors.Wrap(err, "reconnect request")
	}

	if res.CMD() == uint32(protocol.StatusUnauthenticated) {
		return c.auth()
	}

	var info control.AuthResponse

	if err = res.Unmarshal(&info); err != nil {
		return errors.Wrap(err, "reconnect unmarshal")
	}

	c.authInfo = &info
	return nil
}

func (c *Client) isAuthExpired() bool {
	if c.authInfo == nil {
		return true
	}

	expireAt := time.Unix(c.authInfo.GetExpires()/1000-10, c.authInfo.GetExpires()%1000*int64(time.Millisecond))
	return time.Since(expireAt) >= 0

}

// Do will do request to server
func (c *Client) Do(ctx context.Context, req *Request, opts ...RequestOption) (res *protocol.Packet, err error) {
	rp, e := protocol.NewRequest(c.conn.Context(), req.Cmd, req.Body)

	if e != nil {
		err = e
		return
	}

	ropts := newRequestOptions(opts...)

	rc, cancel := context.WithTimeout(ctx, ropts.timeout)
	defer cancel()

	if err = c.write(&rp); err != nil {
		return
	}

	res, err = c.recv(rc, rp.Metadata.RequestId)

	return
}

// Subscribe using to register handle of push data
// concurrency unsafe, please sub at first time
func (c *Client) Subscribe(cmd uint32, sub func(*protocol.Packet)) {
	var (
		subs []func(*protocol.Packet)
		ok   bool
	)

	if subs, ok = c.subs[cmd]; !ok {
		subs = make([]func(*protocol.Packet), 1)
		subs[0] = sub
		c.subs[cmd] = subs
	} else {
		c.subs[cmd] = append(subs, sub)
	}
}

// OnPing using to custom handle ping packet
func (c *Client) OnPing(fn func(*protocol.Packet)) {
	c.onPing = fn
}

// OnPong using to custom handle pong packet
func (c *Client) OnPong(fn func(*protocol.Packet)) {
	c.onPong = fn
}

// OnClose using to handle client close
func (c *Client) OnClose(fn func(err error)) {
	c.onClose = fn
}

// AfterReconnected using to handle client after reconnected
func (c *Client) AfterReconnected(fn func()) {
	c.afterReconnected = fn
}

// Close used to close conn between server
func (c *Client) Close(err error) error {
	c.Logger.Info("close client")

	c.conn.Close(errors.New("close by client"))

	close(c.closeCh)

	if c.onClose != nil {
		c.onClose(err)
	}

	return nil
}

func (c *Client) closeByServer(packet *protocol.Packet) {
	var reason control.Close

	if err := packet.Unmarshal(&reason); err != nil {
		c.Logger.Errorf("failed to unmarshal close reason, err: %v", err)
	} else {
		c.Logger.Errorf("close by server, code: %v, reason: %s", reason.Code, reason.Reason)
	}

	c.conn.Close(errors.New("close by server"))

	c.reconnecting()
}

func (c *Client) keepalive() {
	t := time.NewTicker(c.dialOptions.Keepalive)

	now := time.Now()
	c.lastPongAt = now

	check := func() error {
		if c.lastKeepaliveId == 0 {
			return nil
		}

		if d := time.Since(c.lastPongAt); d > c.dialOptions.KeepaliveTimeout {
			return errors.Errorf("keepalive timeout %s", d.String())
		}

		return nil
	}

	ping := func() error {
		id := c.conn.Context().NextReqId()
		hid := new(int32)
		*hid = int32(id)

		p, err := protocol.NewPacket(c.conn.Context(), protocol.RequestPacket, uint32(control.Command_CMD_HEARTBEAT), &control.Heartbeat{Timestamp: time.Now().UnixNano() / int64(time.Millisecond), HeartbeatId: hid}, protocol.WithRequestId(id))

		if err != nil {
			return err
		}

		if err = c.write(&p); err != nil {
			return err
		}

		c.lastKeepaliveId = id

		return nil
	}

	for {
		select {
		case <-c.closeCh:
			return
		case <-t.C:
			if err := check(); err != nil {
				c.Logger.Errorf("keepalive error: %v", err)
				c.reconnecting()
				return
			}

			if err := ping(); err != nil {
				c.Logger.Errorf("keepalive failed to ping, err: %v", err)
				c.reconnecting()
				return
			}
		}
	}
}

func (c *Client) onPacket(packet *protocol.Packet, err error) {
	if err != nil {
		c.Logger.Errorf("conn receive packet error: %v", err)
		c.reconnecting()
		return
	}

	c.Logger.Debugf("got packet, type: %s, cmd: %d, req_id: %d, status_code: %d", packet.Metadata.Type, packet.CMD(), packet.Metadata.RequestId, packet.Metadata.StatusCode)

	if packet.IsControl() {
		c.handleControl(packet)
		return
	}

	if packet.Metadata.Type == protocol.PushPacket {
		c.handlePush(packet)
		return
	}

	if packet.Metadata.Type == protocol.ResponsePacket {
		c.handleResponse(packet)
		return
	}

	c.Logger.Warnf("client did't support request now, cmd: %d", packet.Metadata.CmdCode)
}

func (c *Client) handlePush(packet *protocol.Packet) {
	subs, ok := c.subs[packet.CMD()]

	if !ok || len(subs) == 0 {
		return
	}

	for _, sub := range subs {
		sub(packet)
	}
}

func (c *Client) handleControl(packet *protocol.Packet) {
	if packet.IsPing() {
		c.handlePing(packet)
		return
	}

	if packet.IsPong() {
		c.handlePong(packet)
		return
	}

	if packet.IsClose() {
		c.closeByServer(packet)
		return
	}

	if packet.IsAuth() || packet.IsReconnect() {
		c.handleResponse(packet)
	}
}

func (c *Client) handleResponse(packet *protocol.Packet) {
	c.recvsMu.RLock()
	defer c.recvsMu.RUnlock()

	if ch, ok := c.recvs[packet.Metadata.RequestId]; ok {
		select {
		case ch <- packet:
		default:
			c.Logger.Warnf("duplicate response of req %d", packet.Metadata.RequestId)
		}
		return
	}
	c.Logger.Warnf("no receiver for req %d", packet.Metadata.RequestId)
}

func (c *Client) handlePing(packet *protocol.Packet) {
	if c.onPing != nil {
		c.onPing(packet)
	}

	if !c.conn.NeedHandleControl() {
		return
	}

	res, _ := protocol.NewResponse(c.conn.Context(), uint32(control.Command_CMD_HEARTBEAT), protocol.StatusSuccess, packet.Body, protocol.WithRequestId(packet.Metadata.RequestId))

	if err := c.write(&res); err != nil {
		c.Logger.Errorf("failed to send heartbeat ack, err: %v", err)
	}
}

func (c *Client) handlePong(packet *protocol.Packet) {
	if c.onPong != nil {
		c.onPong(packet)
	}

	if packet.Metadata.RequestId == c.lastKeepaliveId {
		c.lastPongAt = time.Now()
	}
}

func (c *Client) recv(ctx context.Context, rid uint32) (res *protocol.Packet, err error) {
	ch := make(chan *protocol.Packet, 1)

	defer func() {
		c.recvsMu.Lock()
		delete(c.recvs, rid)
		c.recvsMu.Unlock()

		close(ch)
	}()

	c.recvsMu.Lock()
	c.recvs[rid] = ch
	c.recvsMu.Unlock()

	select {
	case res = <-ch:
	case <-ctx.Done():
		err = errors.Errorf("wait for %d response timeout", rid)
	}

	return
}

func (c *Client) write(p *protocol.Packet) error {
	return c.conn.Write(p, protocol.GzipSize(c.dialOptions.MinGzipSize))
}
