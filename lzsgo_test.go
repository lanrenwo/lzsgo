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
		ret := int(Compress(&comprBuf[0], PkgSize, &pkgBuf[0], int32(i)))
		if ret <= 0 {
			t.Errorf("Compress failed: %d %d", ret, i)
		}
		unprBuf := make([]byte, i)
		ret = int(Uncompress(&unprBuf[0], PkgSize, &comprBuf[0], int32(ret)))
		if ret <= 0 {
			t.Errorf("Uncompress failed: %d %d", ret, i)
		}
		if !bytes.Equal(pkgBuf[:i], unprBuf[:ret]) {
			t.Errorf("Uncompress failed: %d %d", i, ret)
		}
	}
}

func BenchmarkCompress(b *testing.B) {
	buf := randBytes(1500)
	comprBuf := make([]byte, len(buf))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Compress(&comprBuf[0], int32(len(comprBuf)), &buf[0], int32(len(buf)))
	}
}

func BenchmarkUncompress(b *testing.B) {
	buf := randBytes(1500)
	comprBuf := make([]byte, len(buf))
	Compress(&comprBuf[0], int32(len(comprBuf)), &buf[0], int32(len(buf)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Uncompress(&buf[0], int32(len(buf)), &comprBuf[0], int32(len(comprBuf)))
	}
}

func randBytes(n int) []byte {
	b := make([]byte, n)
	for i := 0; i < n; i++ {
		b[i] = byte(rand.Intn(256))
	}
	return b
}
