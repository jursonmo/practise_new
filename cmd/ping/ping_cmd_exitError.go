package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// 查看 exec.Cmd 退出exec.ExitError的具体错误是什么,

func main() {
	test()
	fmt.Println("test end")
	return
	// 设置目标地址
	target := "1.1.1.1"
	//count := "4" // 发包次数

	// 构造 ping 命令
	//cmd := exec.Command("ping", target, "-c", count) // Linux/macOS
	args := []string{"-c", fmt.Sprintf("%d", 4), "-W", fmt.Sprintf("%d", 6), "-I", "eth0", target}
	cmd := exec.Command("ping", args...)
	// cmd := exec.Command("ping", target, "-n", count) // Windows
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Failed to execute ping: %v\n", err)
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ProcessState != nil {
			fmt.Printf("Ping command terminated. Exit code: %d, Signal: %v\n",
				exitErr.ProcessState.ExitCode(), exitErr.ProcessState.Sys())
		}
		fmt.Println(string(output)) // 打印错误输出
		return
	}
	// 打印原始输出
	fmt.Println("CombinedOutput output:", string(output))
	return
	// 捕获命令输出
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	// 执行命令
	err = cmd.Run()
	if err != nil {
		fmt.Printf("Failed to execute ping: %v\n", err)
		fmt.Println(out.String()) // 打印错误输出
		return
	}

	// 打印原始输出
	fmt.Println("Raw output:")
	fmt.Println(out.String())

	// 解析统计信息
	parsePingResult(out.String())
}

func parsePingResult(output string) {
	scanner := bufio.NewScanner(strings.NewReader(output))
	statsRegex := regexp.MustCompile(`(\d+) packets transmitted, (\d+) received, (\d+)% packet loss`)
	rttRegex := regexp.MustCompile(`rtt min/avg/max/mdev = ([\d.]+)/([\d.]+)/([\d.]+)/([\d.]+) ms`)

	for scanner.Scan() {
		line := scanner.Text()

		// 匹配统计信息
		if statsRegex.MatchString(line) {
			matches := statsRegex.FindStringSubmatch(line)
			fmt.Printf("Packets: Sent = %s, Received = %s, Loss = %s%%\n", matches[1], matches[2], matches[3])
		}

		// 匹配延迟信息
		if rttRegex.MatchString(line) {
			matches := rttRegex.FindStringSubmatch(line)
			fmt.Printf("Round-trip times: Min = %sms, Avg = %sms, Max = %sms\n", matches[1], matches[2], matches[3])
		}
	}
}

func test() {
	args := []string{
		"-c", fmt.Sprintf("%d", 4),
		"-W", fmt.Sprintf("%d", 6),
	}

	args = append(args, "-I", "eth0")

	args = append(args, "1.1.1.1")

	cmd := exec.Command("ping", args...)
	fmt.Printf("Ping command: %s\n", strings.Join(cmd.Args, " "))
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Failed to execute ping: %v\n", err)
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ProcessState != nil {
			fmt.Printf("Ping command terminated. Exit code: %d, Signal: %v\n",
				exitErr.ProcessState.ExitCode(), exitErr.ProcessState.Sys())
		}
		fmt.Println(string(output)) // 打印错误输出
		return
	}
	// 打印原始输出
	fmt.Println("CombinedOutput output:", string(output))
}

func test2() {
	args := []string{
		"-c", fmt.Sprintf("%d", 4),
		"-W", fmt.Sprintf("%d", 6),
	}

	args = append(args, "-I", "eth0")

	args = append(args, "1.1.1.1")

	cmd := exec.Command("sh", "-c", fmt.Sprintf("ping %s", strings.Join(args, " ")))
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Failed to execute ping: %v\n", err)
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ProcessState != nil {
			fmt.Printf("Ping command terminated. Exit code: %d, Signal: %v\n",
				exitErr.ProcessState.ExitCode(), exitErr.ProcessState.Sys())
		}
		fmt.Println(string(output)) // 打印错误输出
		return
	}
	// 打印原始输出
	fmt.Println("CombinedOutput output:", string(output))
}
