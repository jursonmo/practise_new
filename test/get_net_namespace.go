package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func listNetworkNamespaces() ([]string, error) {
	var namespaces []string

	// 检查默认的/var/run/netns目录
	files, err := os.ReadDir("/var/run/netns")
	if err == nil {
		for _, f := range files {
			namespaces = append(namespaces, f.Name())
		}
		return namespaces, nil
	}

	// 如果/var/run/netns不存在，检查/proc/[pid]/ns/net
	procDirs, err := filepath.Glob("/proc/[0-9]*/ns/net")
	if err != nil {
		return nil, err
	}

	for _, procDir := range procDirs {
		pid := strings.Split(procDir, "/")[2]
		namespaces = append(namespaces, pid)
	}

	return namespaces, nil
}

func main() {
	nsList, err := listNetworkNamespaces()
	if err != nil {
		fmt.Printf("Error listing network namespaces: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Network Namespaces:")
	for _, ns := range nsList {
		fmt.Println(ns)
	}
}
