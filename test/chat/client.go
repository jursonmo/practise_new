package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

func main() {
	conn, err := net.Dial("tcp", "localhost:12345")
	if err != nil {
		fmt.Println("连接服务器失败：", err.Error())
		os.Exit(1)
	}
	defer conn.Close()

	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("请输入用户名：")
	username, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("读取用户名失败：", err.Error())
		os.Exit(1)
	}
	fmt.Printf("%s", username)
	fmt.Fprintf(conn, "%s", username)

	go receiveMessages(conn)

	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("读取消息失败：", err.Error())
			continue
		}
		msg = strings.TrimSpace(msg)
		fmt.Fprintf(conn, "%s\n", msg)
		fmt.Printf("%s", msg)
		if msg == "exit" {
			return
		}
	}
}

func receiveMessages(conn net.Conn) {
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		msg := scanner.Text()
		fmt.Println(msg)
	}
}
