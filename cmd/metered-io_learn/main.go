package main

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/samber/go-metered-io"
)

func main() {
	r := metered.NewReader(strings.NewReader("Hello, world!"))
	buf := make([]byte, 5)
	_, err := r.Read(buf)
	if err != nil {
		panic(err)
	}
	totalBytes := r.Rx()
	fmt.Printf("buf:%s, totalBytes:%d\n", string(buf), totalBytes)

	var buffer bytes.Buffer
	w := metered.NewWriter(&buffer)
	_, err = w.Write([]byte("Hello, world!"))
	if err != nil {
		panic(err)
	}
	totalBytes = w.Tx()
	fmt.Printf("buffer:%s, totalBytes:%d\n", string(buffer.Bytes()), totalBytes)

}
