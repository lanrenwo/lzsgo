# lzsgo
lzsgo, using c2go, converts OpenConnect's LZS library, pure golang code, good performance.

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

# Thanks
[OpenConnect](https://gitlab.com/openconnect/)

[goplus c2go](https://github.com/goplus/c2go)
