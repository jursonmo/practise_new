package main

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
)

func main() {
	cmd := exec.CommandContext(context.Background(), "sh")
	// 获取标准输入和输出
	stdin, err := cmd.StdinPipe()
	if err != nil {
		panic(fmt.Errorf("获取标准输入失败: %w", err))
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(fmt.Errorf("获取标准输出失败: %w", err))
	}

	// 启动 SSH 会话
	if err := cmd.Start(); err != nil {
		panic(err)
	}

	scanner := bufio.NewScanner(stdout)
	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Printf(" 输出: %s\n", line)
		}
	}()

	stdin.Write([]byte("ls\n"))
	select {}
}
