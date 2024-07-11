package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

// https://blog.csdn.net/Naisu_kun/article/details/130598129

/*
./gossh root@192.168.134.128:2200 123456

登录成功后服务器gosshd打印:
2024/06/24 04:04:35 New SSH connection from 192.168.134.1:51127 (SSH-2.0-Go)
2024/06/24 04:04:35 Creating pty...
*/
func main() {
	// 设置客户端请求参数
	// var hostKey ssh.PublicKey
	addr := "root@192.168.134.128:22"
	user := "root"
	passwd := "123456"
	if len(os.Args) > 1 {
		addr = os.Args[1]
		if strings.Contains(addr, "@") {
			arr := strings.Split(addr, "@")
			user = arr[0]
			addr = arr[1]
		}
	}
	if len(os.Args) > 2 {
		passwd = os.Args[2]
	}
	fmt.Printf("addr:%s, user:%s, passwd:%s\n", addr, user, passwd)
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(passwd),
		},
		// HostKeyCallback: ssh.FixedHostKey(hostKey),
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 忽略主机密钥
	}

	// 作为客户端连接SSH服务器
	conn, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		log.Fatal("unable to connect: ", err)
	}
	defer conn.Close()

	// 创建会话
	session, err := conn.NewSession()
	if err != nil {
		log.Fatal("unable to create session: ", err)
	}
	defer session.Close()

	// 设置会话的标准输出、错误输出、标准输入
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	session.Stdin = os.Stdin

	// 设置终端参数
	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	termWidth, termHeight, err := term.GetSize(int(os.Stdout.Fd())) // 获取当前标准输出终端窗口尺寸 // 该操作可能有的平台上不可用，那么下面手动指定终端尺寸即可
	if err != nil {
		log.Fatal("unable to terminal.GetSize: ", err)
	}

	// 设置虚拟终端与远程会话关联
	if err := session.RequestPty("xterm", termHeight, termWidth, modes); err != nil {
		log.Fatal("request for pseudo terminal failed: ", err)
	}

	// 启动远程Shell
	if err := session.Shell(); err != nil {
		log.Fatal("failed to start shell: ", err)
	}

	// 阻塞直至结束会话
	if err := session.Wait(); err != nil {
		log.Fatal("exit error: ", err)
	}
}
