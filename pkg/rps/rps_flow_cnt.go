package main

import (
	"fmt"

	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// SetRPSFlowCnt 设置指定网卡所有RX队列的rps_flow_cnt值
func SetRPSFlowCnt(iface string, flowCnt int) error {
	// 检查输入值是否有效
	if flowCnt < 0 {
		return fmt.Errorf("invalid flow count value, must be greater than 0")
	}

	// 构造sysfs路径
	sysfsPath := filepath.Join("/sys/class/net", iface, "queues")

	// 查找所有rx-*目录
	rxDirs, err := filepath.Glob(filepath.Join(sysfsPath, "rx-*"))
	if err != nil {
		return fmt.Errorf("failed to find rx queues: %v", err)
	}

	if len(rxDirs) == 0 {
		return fmt.Errorf("no rx queues found for interface %s", iface)
	}

	// 遍历所有rx队列
	for _, rxDir := range rxDirs {
		flowCntPath := filepath.Join(rxDir, "rps_flow_cnt")

		// 检查文件是否存在
		if _, err := os.Stat(flowCntPath); os.IsNotExist(err) {
			continue // 如果该队列没有rps_flow_cnt文件，跳过
		}

		// 写入新值
		err := os.WriteFile(flowCntPath, []byte(strconv.Itoa(flowCnt)), 0644)
		if err != nil {
			return fmt.Errorf("failed to write to %s: %v", flowCntPath, err)
		}
	}

	return nil
}

// GetRPSFlowCnt 获取指定网卡所有RX队列的rps_flow_cnt值
func GetRPSFlowCnt(iface string) (map[string]int, error) {
	result := make(map[string]int)

	sysfsPath := filepath.Join("/sys/class/net", iface, "queues")
	rxDirs, err := filepath.Glob(filepath.Join(sysfsPath, "rx-*"))
	if err != nil {
		return nil, fmt.Errorf("failed to find rx queues: %v", err)
	}

	for _, rxDir := range rxDirs {
		flowCntPath := filepath.Join(rxDir, "rps_flow_cnt")

		if _, err := os.Stat(flowCntPath); os.IsNotExist(err) {
			continue
		}

		data, err := os.ReadFile(flowCntPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %v", flowCntPath, err)
		}

		value, err := strconv.Atoi(strings.TrimSpace(string(data)))
		if err != nil {
			return nil, fmt.Errorf("failed to parse value from %s: %v", flowCntPath, err)
		}

		queueName := filepath.Base(rxDir)
		result[queueName] = value
	}

	return result, nil
}

func main() {
	// 示例用法
	iface := "eth0"
	flowCnt := 4096

	// 获取当前值
	currentValues, err := GetRPSFlowCnt(iface)
	if err != nil {
		fmt.Printf("Error getting current values: %v\n", err)
		return
	}
	fmt.Printf("Current rps_flow_cnt values for %s: %v\n", iface, currentValues)

	// 设置新值
	err = SetRPSFlowCnt(iface, flowCnt)
	if err != nil {
		fmt.Printf("Error setting rps_flow_cnt: %v\n", err)
		return
	}
	fmt.Printf("Successfully set rps_flow_cnt to %d for all rx queues of %s\n", flowCnt, iface)

	// 验证新值
	newValues, err := GetRPSFlowCnt(iface)
	if err != nil {
		fmt.Printf("Error verifying new values: %v\n", err)
		return
	}
	fmt.Printf("New rps_flow_cnt values for %s: %v\n", iface, newValues)
}
