package main

import (
	"bytes"
	"fmt"
	"math"
)

func main() {
	a := []byte("AAAA/BBBBB")
	index := bytes.IndexByte(a, '/')
	b := a[:index]
	c := a[index+1:]
	b = append(b, "CCC"...)
	fmt.Println(string(a))
	fmt.Println(string(b))
	fmt.Println(string(c))

	test()
}

func test() {
	var a uint = 1
	var b uint = 2
	fmt.Println(a - b)
	fmt.Println(uint64(math.MaxUint64))
	fmt.Println(uint64(math.MaxUint64 + 1)) //ide 提示溢出了
	max := uint64(math.MaxUint64)
	fmt.Println(uint64(max + 1)) //这样，ide 就不提示溢出了, 因为ide 不知道max 的值
}
