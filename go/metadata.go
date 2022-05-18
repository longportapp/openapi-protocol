package protocol

import "github.com/pkg/errors"

var (
	ErrKeyLengthTooLong    = errors.New("key length should not greate 2^14 - 1")
	ErrValueLengthTooLong  = errors.New("value length should not greate 2^14 - 1")
	ErrInvalidMetadataData = errors.New("invalid metadata binary data")
)

const (
	length7Bit  uint8 = 0
	length15Bit uint8 = 0b10000000
	lengthMask  uint8 = 0b10000000

	max7BitLength   = 1<<7 - 1
	max15BitLength  = 1<<15 - 1
	maxStringLength = max15BitLength
)

type KVPair struct {
	Key string
	Val string
}

type Metadata struct {
	Nonce      uint64
	RequestId  uint32
	CmdCode    uint32
	Verify     bool
	Gzip       bool
	Timeout    uint16
	Codec      CodecType
	StatusCode uint8
	Type       PacketType
	Signature  []byte
	Values     map[string]string
}

func (md *Metadata) UnmarshalValues(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	if len(data) < 2 {
		return ErrInvalidMetadataData
	}

	values := make(map[string]string)

	getString := func() (string, error) {
		l := len(data)

		sl, bitSize, err := unmarshalStringLength(data)

		if err != nil {
			return "", err
		}

		switch bitSize {
		case length7Bit:
			if sl > l-1 {
				return "", ErrInvalidMetadataData
			}

			str := string(data[1 : sl+1])
			data = data[sl+1:]
			return str, nil
		case length15Bit:
			if sl > l-2 {
				return "", ErrInvalidMetadataData
			}
			str := string(data[2 : sl+2])
			data = data[sl+2:]
			return str, nil
		}
		return "", ErrInvalidMetadataData
	}

	for {
		if len(data) == 0 {
			break
		}

		key, err := getString()

		if err != nil {
			return err
		}

		val, err := getString()

		if err != nil {
			return err
		}
		values[key] = val
	}

	md.Values = values

	return nil
}

func unmarshalStringLength(data []byte) (l int, bitSize uint8, err error) {
	if len(data) == 0 {
		err = ErrInvalidMetadataData
		return
	}

	bitSize = uint8(data[0]) & lengthMask

	switch bitSize {
	case length7Bit:
		l = int(uint8(data[0]) & ^lengthMask)
	case length15Bit:
		first := int(uint8(data[0]) & ^lengthMask)
		if len(data) < 2 {
			err = ErrInvalidMetadataData
			return
		}
		second := int(data[1])
		l = first<<8 + second

		if l <= max7BitLength {
			err = ErrInvalidMetadataData
		}
	}

	if l == 0 {
		err = ErrInvalidMetadataData
	}

	return
}

func marshalString(str string) (data []byte, tooLong bool) {
	l := len(str)

	var lenBytes []byte

	if l <= max7BitLength {
		lenBytes = []byte{byte(l)}
	} else if l <= max15BitLength {
		first := byte(uint8(l>>8) | length15Bit)
		second := byte(l & 0b11111111)
		lenBytes = []byte{first, second}
	} else {
		tooLong = true
		return
	}

	data = append(lenBytes, str...)

	return

}

func (md *Metadata) MarshalValues(max int) []byte {
	if len(md.Values) == 0 {
		return nil
	}

	var data []byte

	for key, val := range md.Values {
		if key == "" || val == "" {
			continue
		}

		k, tooLong := marshalString(key)

		// filter too long key
		if tooLong {
			continue
		}

		v, tooLong := marshalString(val)

		// filter too long value
		if tooLong {
			continue
		}

		// metadata binary length is too long
		if len(k)+len(v)+len(data) > max {
			break
		}

		data = append(data, k...)
		data = append(data, v...)

	}

	return data
}

// Set metadata value
func (md *Metadata) Set(k, v string) error {
	if len(k) > maxStringLength {
		return ErrKeyLengthTooLong
	}

	if len(v) > maxStringLength {
		return ErrValueLengthTooLong
	}
	md.Values[k] = v

	return nil
}

// Get metadata value
func (md *Metadata) Get(k string) string {
	return md.Values[k]
}
