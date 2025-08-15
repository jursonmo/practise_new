package main

import "fmt"

// 测试指针方法
type human struct {
	person *Person
}

type Person struct {
	Name string
	Age  int
}

func main() {
	p := &Person{
		Name: "will",
		Age:  18,
	}
	h := &human{
		person: p,
	}
	fmt.Printf("h:%+v\n", h)
}

func (p *Person) String() string {
	return fmt.Sprintf("mystring person:%v", *p)
}

// 为 human 实现 String() 方法，控制打印格式, 避免打印human结构体里的指针 h:&{person:0x1400000c018}
func (h *human) String() string {
	return fmt.Sprintf("&{person:%+v}", h.person)
}
