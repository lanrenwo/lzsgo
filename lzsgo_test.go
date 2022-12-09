package lzsgo

import (
	"bytes"
	"math/rand"
	"testing"
)

const (
	PkgSize = 2048
	MaxMTU  = 1500
)

func TestLZSGo(t *testing.T) {
	for i := 1; i < MaxMTU; i++ {
		pkgBuf := randBytes(i)
		comprBuf := make([]byte, PkgSize)
		ret, err := Compress(pkgBuf, comprBuf)
		if err != nil {
			t.Errorf("Compress failed: %d %d %s", ret, i, err)
		}
		unprBuf := make([]byte, i)
		ret, err2 := Uncompress(comprBuf, unprBuf)
		if err2 != nil {
			t.Errorf("Uncompress failed: %d %d %s", ret, i, err2)
		}
		if !bytes.Equal(pkgBuf[:i], unprBuf[:ret]) {
			t.Errorf("Compress and uncompress data not equal")
		}
	}
}

func BenchmarkCompress(b *testing.B) {
	buf := randBytes(1500)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		comprBuf := make([]byte, PkgSize)
		Compress(buf, comprBuf)
	}
}

func BenchmarkParallelCompress(b *testing.B) {
	buf := randBytes(1500)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			comprBuf := make([]byte, PkgSize)
			Compress(buf, comprBuf)
		}
	})
}

func BenchmarkUncompress(b *testing.B) {
	buf := randBytes(1500)
	comprBuf := make([]byte, len(buf))
	Compress(buf, comprBuf)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Uncompress(comprBuf, buf)
	}
}

func randBytes(n int) []byte {
	b := make([]byte, n)
	for i := 0; i < n; i++ {
		b[i] = byte(rand.Intn(256))
	}
	return b
}
