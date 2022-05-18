package gzip

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGzip_Compress(t *testing.T) {
	f, err := os.Open("./testdata/plain")
	assert.Nil(t, err)

	defer f.Close()

	in, err := ioutil.ReadAll(f)

	assert.Nil(t, err)

	assert.Nil(t, err)

	out, err := Compress(in)
	assert.Nil(t, err)

	dsize := DecompressSize(out)

	assert.Equal(t, len(in), dsize)
	fmt.Printf("plain size: %d\n", len(in))
	fmt.Printf("compressed size: %d\n", len(out))
}

func TestGzip_Decompresse(t *testing.T) {
	f, err := os.Open("./testdata/plain")
	assert.Nil(t, err)

	defer f.Close()

	in, err := ioutil.ReadAll(f)
	assert.Nil(t, err)

	cb, err := Compress(in)
	assert.Nil(t, err)

	out, n, err := Decompress(cb)

	assert.Nil(t, err)

	assert.Equal(t, n, len(in))
	assert.Equal(t, in, out)
}

func BenchmarkGzip_Comporess(b *testing.B) {
	f, err := os.Open("./testdata/plain")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	in, err := ioutil.ReadAll(f)

	if err != nil {
		panic(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = Compress(in)
	}
}

func BenchmarkGzip_Decompress(b *testing.B) {
	f, err := os.Open("./testdata/plain")
	if err != nil {
		panic(err)
	}

	defer f.Close()

	in, err := ioutil.ReadAll(f)

	if err != nil {
		panic(err)
	}

	data, err := Compress(in)

	if err != nil {
		panic(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _, _ = Decompress(data)
	}
}
