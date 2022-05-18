package protocol

import (
	"fmt"
	"testing"

	"github.com/longbridgeapp/assert"
)

func TestCodec(t *testing.T) {
	cases := []string{"unknown", "protobuf", "json", "unknown", "unknown"}

	for i, c := range cases {
		assert.Equal(t, c, CodecType(i).String())
	}
}

func TestHandshake_Pack(t *testing.T) {
	cases := []struct {
		label     string
		handshake Handshake
		bits      string
	}{
		{
			label: "version:3 codec:pb platform:ios",
			handshake: Handshake{
				Version:  3,
				Codec:    CodecProtobuf,
				Platform: PlatformIOS,
				Reserve:  0,
			},
			bits: "0001001100000001",
		},
		{
			label: "version:1 codec:pb platform:android",
			handshake: Handshake{
				Version:  1,
				Codec:    CodecProtobuf,
				Platform: PlatformAndroid,
				Reserve:  0,
			},
			bits: "0001000100000010",
		},
		{
			label: "version:2 codec:pb platform:web",
			handshake: Handshake{
				Version:  2,
				Codec:    CodecProtobuf,
				Platform: PlatformWeb,
				Reserve:  0,
			},
			bits: "0001001000000011",
		},
	}

	for _, c := range cases {
		c := c
		t.Run(fmt.Sprintf("pack %s should ok", c.label), func(t *testing.T) {
			bs := c.handshake.Pack()

			assert.Equal(t, c.bits, fmt.Sprintf("%08b%08b", bs[0], bs[1]))
		})
	}
}

func TestHandshake_Unpack(t *testing.T) {
	cases := []struct {
		label     string
		bytes     []byte
		handshake Handshake
	}{
		{
			label: "version:3 codec:pb platform:ios",
			bytes: []byte{0b00010011, 0b00000001},
			handshake: Handshake{
				Version:  3,
				Codec:    CodecProtobuf,
				Platform: PlatformIOS,
				Reserve:  0,
			},
		},
		{
			label: "version:1 codec:pb platform:android",
			handshake: Handshake{
				Version:  1,
				Codec:    CodecProtobuf,
				Platform: PlatformAndroid,
				Reserve:  0,
			},
			bytes: []byte{0b00010001, 0b00000010},
		},
		{
			label: "version:2 codec:pb platform:web",
			handshake: Handshake{
				Version:  2,
				Codec:    CodecProtobuf,
				Platform: PlatformWeb,
				Reserve:  0,
			},
			bytes: []byte{0b00010010, 0b00000011},
		},
	}

	for _, c := range cases {
		c := c
		t.Run(fmt.Sprintf("unpack %s should ok", c.label), func(t *testing.T) {
			h := Handshake{}
			err := h.Unpack(c.bytes)
			assert.Nil(t, err)

			assert.Equal(t, c.handshake, h)
		})
	}
}
