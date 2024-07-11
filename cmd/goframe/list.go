package main

import (
	"fmt"

	"github.com/gogf/gf/v2/container/glist"
)

func main() {
	// Not concurrent-safe in default.
	l := glist.New()

	// Push
	l.PushBack(1)       //从后面插入值
	l.PushBack(2)       //从后面插入值
	e := l.PushFront(0) //从前面插入值

	// Insert
	l.InsertBefore(e, -1) //从0的前面插入值
	l.InsertAfter(e, "a") //从0的后面插入值
	fmt.Println(l)

	// Pop Pop 出栈后，从list里移除
	fmt.Println(l.PopFront()) // 从前面出栈，返回出栈的值
	fmt.Println(l.PopBack())  //从后面出栈
	fmt.Println(l)

	// All
	fmt.Println(l.FrontAll()) //正序返回一个复本
	fmt.Println(l.BackAll())  //逆序返回一个复本

	// Output:
	// [-1,0,"a",1,2]
	// -1
	// 2
	// [0,"a",1]
	// [0 "a" 1]
	// [1 "a" 0]
}
