package main

import (
	"fmt"
	"runtime"
	"strings"
)

// 它在作用是打印整个函数调用顺序，这样可以快速了解复杂代码的调用过程。比如想快速学习某个开源库或者复杂的代码
func main() {
	fmt.Println("Hello, 世界")
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("  %s\n", "123"))
	builder.WriteString(fmt.Sprintf("  %s\n", "456"))
	fmt.Println(builder.String())
	test1()
}
func test1() {
	fmt.Println("test1")
	test2()
}
func test2() {
	fmt.Println(GetCallStack(2))
}

// 获取当前调用栈的字符串表示
func GetCallStack(skip int) string {
	var pc [50]uintptr
	n := runtime.Callers(skip, pc[:])

	var builder strings.Builder
	builder.WriteString("调用栈:\n")

	for i := 0; i < n; i++ {
		f := runtime.FuncForPC(pc[i])
		file, line := f.FileLine(pc[i])

		// 格式化输出
		// 可以过滤掉runtime包的内容
		if !strings.Contains(f.Name(), "runtime.") || !strings.Contains(f.Name(), "runtime/internal") {
			builder.WriteString(fmt.Sprintf("  %s\n    %s:%d\n",
				f.Name(), file, line))
		}
	}
	return builder.String()
}
