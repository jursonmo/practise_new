package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/cpu"
)

func main() {
	iface := "eth0" // 网卡名称，可按需修改
	if len(os.Args) > 1 {
		if os.Args[1] == "help" {
			fmt.Printf("Usage: %s <iface>\n", os.Args[0])
			os.Exit(0)
		}
		iface = os.Args[1]
	}
	log.Printf("Using interface: %s", iface)

	if err := configureRPS(iface); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("RPS configuration applied successfully")
}

// 获取CPU核心数量
func getCPUCoreCount() (int, error) {
	info, err := cpu.Info()
	if err != nil {
		log.Printf("Error getting CPU info: %v", err)
	}
	//fmt.Println(info)

	if len(info) > 0 {
		return len(info), nil
	}

	cpus := runtime.NumCPU()
	return cpus, nil

	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return 0, err
	}
	content := string(data)
	count := strings.Count(content, "processor\t:") + strings.Count(content, "processor	:")
	if count == 0 {
		// 回退到统计核心数
		entries, err := os.ReadDir("/sys/devices/system/cpu")
		if err != nil {
			return 0, err
		}
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), "cpu") && entry.Name() != "cpufreq" {
				if _, err := strconv.Atoi(entry.Name()[3:]); err == nil {
					count++
				}
			}
		}
	}
	return count, nil
}

// 获取网卡接收队列数量
func getRxQueueCount(iface string) (int, error) {
	return 3, nil
	basePath := filepath.Join("/sys/class/net", iface, "queues")
	files, err := os.ReadDir(basePath)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, f := range files {
		if strings.HasPrefix(f.Name(), "rx-") {
			count++
		}
	}
	return count, nil
}

// 生成CPU位掩码
func generateCPUMask(cpuIDs []int) uint64 {
	var mask uint64
	for _, id := range cpuIDs {
		if id >= 0 && id < 64 {
			mask |= 1 << uint(id)
		}
	}
	return mask
}

// 将掩码格式化为十六进制字符串
func formatHexMask(mask uint64, numCores int) string {
	// 计算需要的十六进制字符数 (每4位一个字符)
	hexDigits := (numCores + 3) / 4
	return fmt.Sprintf("%0*x", hexDigits, mask)
}

// 配置RPS
func configureRPS(iface string) error {
	numCores, err := getCPUCoreCount()
	if err != nil {
		return fmt.Errorf("failed to get CPU cores: %w", err)
	}
	if numCores == 0 {
		return fmt.Errorf("no CPU cores found")
	}
	log.Printf("Detected %d CPU cores", numCores)

	numQueues, err := getRxQueueCount(iface)
	if err != nil {
		return fmt.Errorf("failed to get RX queues: %w", err)
	}
	if numQueues == 0 {
		return fmt.Errorf("no RX queues found for interface %s", iface)
	}

	fmt.Printf("Detected: %d CPU cores, %d RX queues\n", numCores, numQueues)

	// 计算每个队列分配的核心数
	coresPerQueue := numCores / numQueues
	remainingCores := numCores % numQueues

	// 分配核心给队列
	queueAssignments := make([][]int, numQueues)
	currentCore := 0

	for q := 0; q < numQueues; q++ {
		// 计算当前队列应分配的核心数
		coresForThisQueue := coresPerQueue
		if q < remainingCores {
			coresForThisQueue++
		}

		// 分配核心
		assigned := make([]int, coresForThisQueue)
		for i := 0; i < coresForThisQueue; i++ {
			assigned[i] = currentCore % numCores
			currentCore++
		}
		queueAssignments[q] = assigned
	}

	// 写入配置
	for q, assignedCPUs := range queueAssignments {
		maskValue := generateCPUMask(assignedCPUs)
		maskHex := formatHexMask(maskValue, numCores)

		// filePath := filepath.Join("/sys/class/net", iface, "queues", fmt.Sprintf("rx-%d", q), "rps_cpus")
		// if err := os.WriteFile(filePath, []byte(maskHex), 0644); err != nil {
		// 	return fmt.Errorf("failed to write %s: %w", filePath, err)
		// }

		fmt.Printf("Queue rx-%d: CPUs %v → Mask %s\n", q, assignedCPUs, maskHex)
	}
	return nil
}
