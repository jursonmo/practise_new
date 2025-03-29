package main

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

func main() {
	GetLteandWanlink(context.Background(), "8c34ff047c16709d6ba709df4f530c74")
}
func GetLteandWanlink(ctx context.Context, deviceID string) (string, error) {
	//keyFilePath := "/root/noc_server/id_rsa_2048_unencrypted"
	keyFilePath := "/Users/will/.ssh/id_rsa"
	//username := "chengjiajun"
	username := "mjw13"
	host := "frp.obc.center"
	port := 2222

	// 构建 SSH 命令
	cmd := exec.CommandContext(ctx, "ssh", "-tt", "-o", "StrictHostKeychecking=no", "-p", fmt.Sprintf("%d", port), "-i", keyFilePath, fmt.Sprintf("%s@%s", username, host))
	// out, err := cmd.CombinedOutput()
	// if err != nil {
	// 	fmt.Printf("cmd.Run() failed with %s\n", err)
	// 	return "", fmt.Errorf("cmd.Run() failed with %s\n", err)
	// }
	// fmt.Printf("combined out:\n%s\n", string(out))
	// time.Sleep(10 * time.Second)
	// //cmd := exec.CommandContext(ctx, "sh")
	// 获取标准输入和输出
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("获取标准输入失败: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("获取标准输出失败: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		panic(err)
	}
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Printf(" stderr输出: %s\n", line)
		}
	}()
	// 启动 SSH 会话
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("启动 SSH 命令失败: %w", err)
	}

	scanner := bufio.NewScanner(stdout)

	// 等待设备输出 `ID>` 提示符
	fmt.Printf("等待设备输出提示符 `ID>`...\n")
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Printf("%s", line)
		if strings.Contains(line, "\r") || strings.Contains(line, "\n") {
			fmt.Printf("需要换行?\n")
		}
		if strings.Contains(line, "ID>") {
			fmt.Printf("检测到提示符 `ID>`，准备输入 deviceID\n")
			// 输入 deviceID
			fmt.Printf("输入 deviceID: %s\n", deviceID)
			id := []byte(deviceID)
			_, err = stdin.Write(append(id, '\r'))
			if err != nil {
				panic(err)
			}
		}
		if strings.Contains(line, "Last login:") {
			_, err = stdin.Write([]byte("ip ro sh\n\n"))
			if err != nil {
				return "", fmt.Errorf("发送 `ip ro sh` 命令失败: %w", err)
			}
		}
	}
	if err := cmd.Wait(); err != nil {
		panic(err)
	}
	return "", nil
}
