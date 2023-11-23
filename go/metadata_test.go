package protocol

import (
	"crypto/rand"
	"testing"

	"github.com/longbridgeapp/assert"
)

func randomString(n int) string {
	data := make([]byte, n)

	_, _ = rand.Read(data)

	return string(data)
}

func TestStringMarshal(t *testing.T) {
	str1 := randomString(128)
	str2 := randomString(max15BitLength + 1)
	str3 := randomString(max15BitLength)
	str4 := randomString(257)

	cases := []struct {
		label   string
		target  string
		tooLong bool
		result  []byte
	}{
		{
			label:   "marshal normal string should ok",
			target:  "hello",
			tooLong: false,
			result:  []byte{5, 'h', 'e', 'l', 'l', 'o'},
		}, {
			label:   "marshal string length large 127 should ok",
			target:  str1,
			tooLong: false,
			result:  append([]byte{128, 128}, str1...),
		}, {
			label:   "marshal too long string should return too long",
			target:  str2,
			tooLong: true,
		}, {
			label:   "marshal longest string should ok",
			target:  str3,
			tooLong: false,
			result:  append([]byte{255, 255}, str3...),
		}, {
			label:  "two byte length marshal should ok",
			target: str4,
			result: append([]byte{129, 1}, str4...),
		},
	}

	for _, item := range cases {
		t.Run(item.label, func(t *testing.T) {
			out, tooLong := marshalString(item.target)
			assert.Equal(t, item.tooLong, tooLong)
			assert.Equal(t, item.result, out)

			if !tooLong {
				l, bitSize, err := unmarshalStringLength(out)
				assert.NoError(t, err)
				assert.Equal(t, len(item.target), l)

				switch bitSize {
				case length7Bit:
					assert.Equal(t, []byte(item.target), out[1:])
				case length15Bit:
					assert.Equal(t, []byte(item.target), out[2:])
				}

			}

		})
	}
}

func TestUnmarshalStringLength(t *testing.T) {
	d1, _ := marshalString("hello")
	d2, _ := marshalString(randomString(1000))

	cases := []struct {
		label   string
		data    []byte
		length  int
		bitSize uint8
		err     bool
	}{
		{
			label:   "unmarshal string length should ok",
			data:    d1,
			length:  5,
			bitSize: length7Bit,
		}, {
			label:   "unmarshal string length greate 127 should ok",
			data:    d2,
			length:  1000,
			bitSize: length15Bit,
		}, {
			label:   "zero length should not raise error",
			data:    []byte{0, 0},
			length:  0,
			bitSize: length7Bit,
		}, {
			label:   "two bytes length can't less than 128",
			data:    []byte{128, 6},
			length:  6,
			bitSize: length15Bit,
			err:     true,
		},
	}

	for _, item := range cases {
		t.Run(item.label, func(t *testing.T) {
			l, bitSize, err := unmarshalStringLength(item.data)

			assert.Equal(t, item.length, l)
			assert.Equal(t, item.bitSize, bitSize)

			if item.err {
				assert.Error(t, err)
			}
		})
	}
}

func TestMarshalValues(t *testing.T) {
	// 	cases := []string{
	// 		label string
	// 		data []byte
	// 	}

	r1 := randomString(68)
	r2 := randomString(101)
	r3 := randomString(187)
	r4 := randomString(167)

	cases := []struct {
		label     string
		md        *Metadata
		values    map[string]string
		maxLength int
		err       bool
	}{
		{
			label: "marshal and unmarshal should ok",
			md: &Metadata{
				Values: map[string]string{
					"key1":      r1,
					"key2":      r2,
					"KEY3":      r3,
					"key4":      r4,
					"empty_key": "",
				},
			},
			values: map[string]string{
				"key1":      r1,
				"key2":      r2,
				"key3":      r3,
				"key4":      r4,
				"empty_key": "",
			},
			maxLength: 1<<16 - 1,
		}, {
			label: "max length can limit marshal result",
			md: &Metadata{
				Values: map[string]string{
					"key1": "hello",
					"key2": "world",
				},
			},
			values: map[string]string{
				"key1": "hello",
			},
			maxLength: 12,
		},
	}

	for _, item := range cases {
		t.Run(item.label, func(t *testing.T) {
			data := item.md.MarshalValues(item.maxLength)

			err := item.md.UnmarshalValues(data)

			if item.err {
				assert.Error(t, err)
			}

			assert.Equal(t, item.values, item.md.Values)
		})
	}
}
