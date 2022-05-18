// fork from https://github.com/grpc/grpc-go/blob/ede71d589c/encoding/gzip/gzip.go
package gzip

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"sync"

	"github.com/pkg/errors"
)

var defaultCompressor *compressor

func init() {
	defaultCompressor = &compressor{}
	defaultCompressor.poolCompressor.New = func() interface{} {
		return &writer{Writer: gzip.NewWriter(ioutil.Discard), pool: &defaultCompressor.poolCompressor}
	}
}

type writer struct {
	*gzip.Writer
	pool *sync.Pool
}

// SetLevel updates the registered gzip compressor to use the compression level specified (gzip.HuffmanOnly is not supported).
// NOTE: this function must only be called during initialization time (i.e. in an init() function),
// and is not thread-safe.
//
// The error returned will be nil if the specified level is valid.
func SetLevel(level int) error {
	if level < gzip.DefaultCompression || level > gzip.BestCompression {
		return fmt.Errorf("grpc: invalid gzip compression level: %d", level)
	}
	defaultCompressor.poolCompressor.New = func() interface{} {
		w, err := gzip.NewWriterLevel(ioutil.Discard, level)
		if err != nil {
			panic(err)
		}
		return &writer{Writer: w, pool: &defaultCompressor.poolCompressor}
	}
	return nil
}

func (c *compressor) Compress(w io.Writer) (io.WriteCloser, error) {
	z := c.poolCompressor.Get().(*writer)
	z.Writer.Reset(w)
	return z, nil
}

func (z *writer) Close() error {
	defer z.pool.Put(z)
	return z.Writer.Close()
}

type reader struct {
	*gzip.Reader
	pool *sync.Pool
}

func (c *compressor) Decompress(r io.Reader) (io.Reader, error) {
	z, inPool := c.poolDecompressor.Get().(*reader)
	if !inPool {
		newZ, err := gzip.NewReader(r)
		if err != nil {
			return nil, err
		}
		return &reader{Reader: newZ, pool: &c.poolDecompressor}, nil
	}
	if err := z.Reset(r); err != nil {
		c.poolDecompressor.Put(z)
		return nil, err
	}
	return z, nil
}

func (z *reader) Read(p []byte) (n int, err error) {
	n, err = z.Reader.Read(p)
	if err == io.EOF {
		z.pool.Put(z)
	}
	return n, err
}

// RFC1952 specifies that the last four bytes "contains the size of
// the original (uncompressed) input data modulo 2^32."
// gRPC has a max message size of 2GB so we don't need to worry about wraparound.
func (c *compressor) DecompressedSize(buf []byte) int {
	last := len(buf)
	if last < 4 {
		return -1
	}
	return int(binary.LittleEndian.Uint32(buf[last-4 : last]))
}

type compressor struct {
	poolCompressor   sync.Pool
	poolDecompressor sync.Pool
}

func Compress(in []byte) (out []byte, err error) {
	buf := &bytes.Buffer{}

	var z io.WriteCloser
	if z, err = defaultCompressor.Compress(buf); err != nil {
		err = errors.Wrap(err, "create gzip writer")
		return
	}

	if _, err = z.Write(in); err != nil {
		err = errors.Wrap(err, "compress data")
		return
	}

	if err = z.Close(); err != nil {
		err = errors.Wrap(err, "finish gzip compress")
		return
	}

	out = buf.Bytes()

	return
}

func Decompress(in []byte) (out []byte, n int, err error) {
	r := bytes.NewReader(in)

	var or io.Reader
	if or, err = defaultCompressor.Decompress(r); err != nil {
		err = errors.Wrap(err, "create gzip reader")
		return
	}

	dsize := defaultCompressor.DecompressedSize(in)

	buf := bytes.NewBuffer(make([]byte, 0, dsize+bytes.MinRead))

	var rn int64
	rn, err = buf.ReadFrom(io.LimitReader(or, int64(dsize+1)))

	n = int(rn)

	out = buf.Bytes()

	return
}

func DecompressSize(in []byte) int {
	return defaultCompressor.DecompressedSize(in)
}
