package main

import (
	"fmt"

	//lru "github.com/hashicorp/golang-lru/v2"
	lru "github.com/hashicorp/golang-lru" //为了使用老的golang 版本.
)

func main() {
	// 创建一个容量为 128 的 LRU 缓存
	l, _ := lru.New(3)

	// 添加键值对
	l.Add("key1", "value1")
	l.Add("key2", "value2")
	l.Add("key3", "value3")
	// 获取值
	if value, ok := l.Get("key1"); ok {
		fmt.Println(value)
	}

	// 当缓存满时，最久未访问的条目会被自动淘汰
}
