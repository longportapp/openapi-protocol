package protocol

import (
	"context"
	"sync/atomic"
)

type ContextKey int

const (
	ContextKeyHeader ContextKey = iota
	ContextKeyAuthInfo
	ContextKeyEnd
)

type Uint32Gen func() uint32

func GetRequestIDGen() Uint32Gen {
	var id uint32

	return func() uint32 {
		return atomic.AddUint32(&id, 1)
	}
}

type PeerSide string

const (
	ServerSide PeerSide = "server"
	ClientSide PeerSide = "client"
)

type Context struct {
	context.Context
	buildin [ContextKeyEnd]interface{}

	NextReqId  Uint32Gen
	Side       PeerSide
	Handshaked bool
	Authed     bool
	Codec      CodecType
	Version    uint8
	Platform   PlatformType

	beginUnpack bool
	closed      bool
	proxyed     bool
}

func (c *Context) SetProxyed() {
	c.proxyed = true
}

func (c *Context) IsProxyed() bool {
	return c.proxyed
}

func (c *Context) SetClosed() {
	c.closed = true
}

func (c *Context) IsClosed() bool {
	return c.closed
}

func (c *Context) Value(k interface{}) interface{} {
	if key, ok := k.(ContextKey); ok {
		return c.buildin[key]
	}

	return c.Context.Value(k)
}

func (c *Context) SetAuth(info interface{}) {
	c.Authed = true
	c.buildin[ContextKeyAuthInfo] = info
}

func (c *Context) GetAuth() (bool, interface{}) {
	if c.Authed {
		return true, c.buildin[ContextKeyAuthInfo]
	}

	return false, nil
}

// TODO: unpack metrics
func (c *Context) BeginUnpack() {
	c.beginUnpack = true
}

func (c *Context) InUnpack() bool {
	return c.beginUnpack
}

func (c *Context) EndUnpack() {
	c.beginUnpack = false
	c.buildin[ContextKeyHeader] = nil
}

func (c *Context) SetHeader(h interface{}) {
	c.buildin[ContextKeyHeader] = h
}

func (c *Context) GetHeader() interface{} {
	return c.buildin[ContextKeyHeader]
}

func (c *Context) Handshake(h *Handshake) error {
	_, err := GetProtocol(h.Version)
	if err != nil {
		return err
	}

	c.Version = h.Version
	c.Codec = h.Codec
	c.Handshaked = true
	c.Platform = h.Platform

	return nil
}

// NewContext create *protocol.Context
func NewContext(parent context.Context, side PeerSide) *Context {
	return &Context{
		Side:      side,
		Context:   parent,
		NextReqId: GetRequestIDGen(),
	}
}
