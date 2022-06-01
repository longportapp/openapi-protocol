package client

import (
	"context"
	"fmt"
	"net/url"

	protocol "github.com/longbridgeapp/openapi-protocol/go"
)

var connDialers = make(map[string]DialConnFunc)

// RegisterDialer use to register conn dialer
func RegisterDialer(scheme string, fn DialConnFunc) {
	if _, ok := connDialers[scheme]; ok {
		panic(fmt.Errorf("dialer for %s is already exists", scheme))
	}

	connDialers[scheme] = fn
}

// GetDialer use to find dialer for specific scheme
func GetDialer(scheme string) (DialConnFunc, bool) {
	f, ok := connDialers[scheme]
	return f, ok
}

type DialConnFunc func(ctx context.Context, cli *Client, uri *url.URL, handshake *protocol.Handshake, opts *DialOptions) (ClientConn, error)

// ClientConn is socket conn abstract
type ClientConn interface {
	Close(error)
	OnPacket(func(*protocol.Packet, error))
	Write(*protocol.Packet, ...protocol.PackOption) error
	Context() *protocol.Context
	NeedHandleControl() bool
}
