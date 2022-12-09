# lzsgo
[![Go](https://github.com/lanrenwo/lzsgo/workflows/Go/badge.svg)](https://github.com/lanrenwo/lzsgo/actions)
[![codecov](https://codecov.io/gh/lanrenwo/lzsgo/branch/main/graph/badge.svg)](https://codecov.io/gh/lanrenwo/lzsgo)

lzsgo, converts OpenConnect's LZS library, pure golang code, good performance.

# Installation
```
go get github.com/lanrenwo/lzsgo
```
# How to use
```
package main

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/lanrenwo/lzsgo"
)

func main() {
	var (
		n   int
		err error
	)
	s := "hello world"
	src := []byte(strings.Repeat(s, 50))

	comprBuf := make([]byte, 2048)
	n, err = lzsgo.Compress(src, comprBuf)
	if err != nil {
		fmt.Printf("Compress failed: %s", err)
		return
	}

	unprBuf := make([]byte, 2048)
	n, err = lzsgo.Uncompress(comprBuf[:n], unprBuf)
	if err != nil {
		fmt.Printf("Uncompress failed: %s", err)
		return
	}

	if !bytes.Equal(src, unprBuf[:n]) {
		fmt.Printf("Compress and uncompress data not equal")
		return
	}

	fmt.Println("ok")
}
```

# Benchmarks
cpu: Intel(R) Xeon(R) CPU E5-2603 v4 @ 1.70GHz
1. [Lzsc](https://github.com/lanrenwo/lzsc): LZS CGO Version
2. [Lz4](https://github.com/pierrec/lz4): LZ4 compression and decompression in pure Go
3. [Lzsgo](https://github.com/lanrenwo/lzsgo)：LZS pure Go Version

| **No.** | **Lzsc (µs)** | **Lz4 (µs)** | **Lzsgo (µs)** |
|---------|---------------|--------------|----------------|
| 1       | 16.259        | 21.271       | 37.199         |
| 2       | 16.509        | 32.02        | 26.146         |
| 3       | 16.466        | 20.822       | 36.489         |
| 4       | 16.478        | 32.43        | 26.209         |
| 5       | 16.228        | 21.226       | 37.17          |
| 6       | 16.419        | 21.225       | 26.805         |
| 7       | 11.442        | 15.693       | 18.748         |
| 8       | 18.105        | 21.741       | 75.763         |
| 9       | 16.379        | 31.703       | 26.084         |
| 10      | 16.787        | 21.008       | 49.658         |
| 11      | 17.976        | 23.757       | 42.22          |
| 12      | 16.916        | 21.491       | 26.831         |
| 13      | 16.459        | 21.363       | 26.833         |
| 14      | 26.699        | 21.253       | 26.305         |
| 15      | 16.324        | 27.265       | 26.789         |
| 16      | 16.5          | 21.378       | 26.557         |
| 17      | 16.739        | 21.267       | 60.373         |
| 18      | 13.809        | 19.062       | 22.715         |
| 19      | 16.473        | 21.606       | 26.615         |
| 20      | 16.626        | 21.128       | 26.788         |
| 21      | 16.233        | 21.182       | 26.631         |
| 22      | 16.261        | 21.1         | 26.697         |
| 23      | 16.482        | 21.321       | 26.77          |
| 24      | 16.306        | 21.285       | 26.621         |
| 25      | 29.922        | 57.146       | 45.37          |
| 26      | 49.096        | 45.1         | 49.116         |
| 27      | 16.551        | 21.653       | 26.575         |
| 28      | 16.791        | 21.368       | 27.13          |
| 29      | 16.888        | 20.996       | 26.866         |
| 30      | 16.784        | 21.606       | 26.873         |
| 31      | 16.869        | 21.189       | 26.965         |
| 32      | 16.445        | 21.212       | 26.727         |
| 33      | 16.443        | 21.23        | 26.725         |
| 34      | 16.237        | 21.688       | 26.429         |
| 35      | 16.448        | 21.06        | 26.866         |
| 36      | 10.638        | 14.251       | 17.47          |
| 37      | 28.48         | 21.251       | 26.828         |
| 38      | 10.603        | 14.228       | 17.373         |
| 39      | 28.701        | 21.126       | 26.577         |
| 40      | 16.143        | 21.196       | 42.042         |
| 41      | 16.393        | 21.156       | 26.631         |
| 42      | 13.663        | 19.206       | 22.521         |
| 43      | 16.209        | 21.307       | 26.775         |
| 44      | 16.544        | 21.178       | 26.739         |
| 45      | 13.632        | 18.839       | 22.636         |
| 46      | 28.589        | 31.978       | 45.615         |
| 47      | 40.428        | 37.663       | 60.06          |
| 48      | 32.282        | 34.105       | 48.15          |
| **AVG:**   | **18.78**         | **23.65**        | **31.75**          |



# Thanks
[OpenConnect](https://gitlab.com/openconnect/)

[goplus c2go](https://github.com/goplus/c2go)
