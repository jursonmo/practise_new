package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
)

func main() {
	// 打开文件
	file, err := os.Open("example.txt")
	if err != nil {
		log.Fatalf("failed opening file: %s", err)
	}
	defer file.Close() // 确保在函数结束时关闭文件

	// 创建一个新的Scanner来读取文件
	scanner := bufio.NewScanner(file)

	// 逐行读取文件
	for scanner.Scan() {
		line := scanner.Text() // 获取当前行的文本
		fmt.Println(line)      // 打印当前行
	}

	// 检查是否有任何扫描错误发生
	if err := scanner.Err(); err != nil {
		log.Fatalf("error reading file: %s", err)
	}
}
