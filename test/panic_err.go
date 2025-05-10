package main

import (
	"errors"
	"fmt"
)

// 希望recover panic时，返回err
func main() {
	err := test()
	fmt.Printf("main err:%v\n", err)
	if errors.Is(err, PanicError) {
		fmt.Printf("yes, it is my panic error\n")
	}
}

var PanicError = errors.New("PanicError")

func test() (err error) {
	defer func() {
		if r := recover(); r != nil {
			if v, ok := r.(error); ok {
				err = v
				return
			}
			fmt.Println("recover:", r)
			err = fmt.Errorf("recover:%v, err:%w", r, PanicError)
		}
	}()
	//panic("mypanic")
	//panic(errors.New("myPanicErr"))
	panicCall()
	return nil
}

func panicCall() {
	fmt.Println("now panic called")
	panic(errors.New("myPanicErr"))
}
