package protocol

import (
	"github.com/Allenxuxu/ringbuffer"
	"github.com/pkg/errors"
)

const (
	HandshakeLength = 2
)

var (
	ErrHandshakeLen           = errors.New("invalid handshake frame length")
	ErrInvalidProtocolVersion = errors.New("invalid protocol version")
)

var manager map[uint8]Protocol

func init() {
	manager = make(map[uint8]Protocol)
}

func Register(v uint8, p Protocol) {
	manager[v] = p
}

type CodecType uint8

const (
	CodecUnknown CodecType = iota
	CodecProtobuf
	CodecJSON
)

var codecStrings = []string{"unknown", "protobuf", "json"}

func (t CodecType) String() string {
	if int(t) >= len(codecStrings) {
		return "unknown"
	}

	return codecStrings[int(t)]
}

type PlatformType uint8

const (
	PlatformUnknow PlatformType = iota
	PlatformIOS
	PlatformAndroid
	PlatformWeb
	PlatformServer
	PlatformDesktopMac
	PlatformDesktopWin
	PlatformDesktopLinux
	PlatformTerminal
	PlatformOpenapi
)

var platformDisplays = []string{"unknown", "iOS", "Android", "web", "server", "desktop-mac", "desktop-windows", "desktop-linux", "terminal", "openapi"}

func (t PlatformType) String() string {
	v := int(t)

	if v >= 0 && v < len(platformDisplays) {
		return platformDisplays[v]
	}

	return "unknown"
}

const (
	VersionMask  = 0xf
	CodecMask    = 0xf0
	PlatformMask = 0xf
	ReserveMask  = 0xf0
)

// Handshake
type Handshake struct {
	Version  uint8
	Codec    CodecType
	Platform PlatformType
	Reserve  uint8
}

func (h Handshake) Pack() []byte {
	fb := h.Version | uint8(h.Codec<<4)
	sb := uint8(h.Platform) | (h.Reserve << 4)

	return []byte{fb, sb}
}

func (h *Handshake) Unpack(data []byte) error {
	if len(data) != 2 {
		return ErrHandshakeLen
	}

	h.Version = VersionMask & data[0]
	h.Codec = CodecType((CodecMask & data[0]) >> 4)

	h.Platform = PlatformType(PlatformMask & data[1])
	h.Reserve = (ReserveMask & data[1]) >> 4

	return nil
}

// Porotcol
type Protocol interface {
	// Unpack used to unpack data from *rinbuffer.Ringbuffer
	Unpack(ctx *Context, buffer *ringbuffer.RingBuffer) (p *Packet, done bool, err error)
	// Unpack used to unpack data from raw data bytes
	UnpackBytes(ctx *Context, bytes []byte) (p *Packet, err error)
	// Pack used to pack Packet to binary data
	Pack(*Context, *Packet, ...PackOption) ([]byte, error)
	// Version return version of protocol
	Version() uint8
}

func GetProtocol(v uint8) (Protocol, error) {
	p := manager[v]

	if p == nil {
		return nil, ErrInvalidProtocolVersion
	}

	return p, nil
}

type PackOptions struct {
	MinGzipSize int
}

type PackOption func(*PackOptions)

func GzipSize(n int) PackOption {
	return func(o *PackOptions) {
		o.MinGzipSize = n
	}
}

func NewPackOptions(opts ...PackOption) *PackOptions {
	o := &PackOptions{}

	for _, opt := range opts {
		opt(o)
	}

	return o
}
