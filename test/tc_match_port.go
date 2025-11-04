package main

// 端口范围转换为多个掩码规则, 主要是为了 tc qos 端口匹配规则, 生成 match dst port a 0xyyy
import (
	"fmt"
)

type PortRange struct {
	Start int
	End   int
}

type PortMask struct {
	Base int
	Mask int
}

// ConvertPortRangeToMinimalMasks 使用最少规则数量覆盖端口范围
func ConvertPortRangeToMinimalMasks(start, end int) []PortMask {
	if start > end {
		return nil
	}

	var rules []PortMask
	current := start

	for current <= end {
		// 找到从current开始的最大可能掩码块
		rule := findLargestMask(current, end)
		rules = append(rules, rule)

		// 计算这个规则覆盖的最后一个端口
		_, maxPort := getCoverage(rule.Base, rule.Mask)
		current = maxPort + 1
	}

	return rules
}

// findLargestMask 找到从start开始能覆盖到end的最大掩码块
func findLargestMask(start, end int) PortMask {
	if start == end {
		return PortMask{start, 0xFFFF}
	}

	// 计算start和end的异或，找到第一个不同的位
	diff := start ^ end

	// 找到最高不同的位
	mask := 0xFFFF
	for (mask & diff) != 0 {
		mask = mask << 1 & 0xFFFF
	}

	// 计算base
	base := start & mask

	// 验证这个掩码是否覆盖整个需要的范围
	_, maxPort := getCoverage(base, mask)
	if maxPort < end {
		// 如果不够，尝试更小的掩码
		mask = mask >> 1 & 0xFFFF
		base = start & mask
	}

	return PortMask{base, mask}
}

// getCoverage 计算掩码覆盖的端口范围
func getCoverage(base, mask int) (int, int) {
	minPort := base
	maxPort := base | (^mask & 0xFFFF)
	return minPort, maxPort
}

// 更高效的算法，使用位运算技巧
func ConvertPortRangeToMinimalMasksV2(start, end int) []PortMask {
	if start > end {
		return nil
	}

	var rules []PortMask

	for start <= end {
		// 计算start的最低有效位
		lsb := start & -start

		// 如果start是0，特殊处理
		if lsb == 0 {
			lsb = 1 << 16
		}

		// 找到最大的块大小，使得块不超过end
		blockSize := lsb
		for blockSize > 0 {
			mask := ^(blockSize - 1) & 0xFFFF
			base := start & mask
			blockEnd := base + blockSize - 1

			if blockEnd <= end {
				rules = append(rules, PortMask{base, mask})
				start = blockEnd + 1
				break
			}
			blockSize >>= 1
		}
	}

	return rules
}

// 最优算法 - 使用数学方法找到最小规则数
func ConvertPortRangeToMinimalMasksOptimal(start, end int) []PortMask {
	if start > end {
		return nil
	}

	var rules []PortMask

	for start <= end {
		// 方法1: 找到start中最低的1位
		lowBit := start & -start

		// 方法2: 找到能覆盖start到end的最大2的幂次方块
		maxBlock := end - start + 1
		blockSize := 1

		// 找到不超过maxBlock的最大2的幂
		for blockSize*2 <= maxBlock && (start&(blockSize*2-1)) == 0 {
			blockSize *= 2
		}

		// 选择更大的块
		finalBlockSize := blockSize
		if lowBit > blockSize && start+lowBit-1 <= end {
			finalBlockSize = lowBit
		}

		mask := ^(finalBlockSize - 1) & 0xFFFF
		base := start & mask

		rules = append(rules, PortMask{base, mask})
		start += finalBlockSize
	}

	return rules
}

// 验证规则是否完全覆盖且没有重叠
func ValidateRules(rules []PortMask, start, end int) (bool, []string) {
	coverage := make([]bool, end-start+1)
	var issues []string

	for i, rule := range rules {
		base := rule.Base
		mask := rule.Mask
		minPort, maxPort := getCoverage(base, mask)

		// 检查规则是否在目标范围内
		if minPort < start || maxPort > end {
			issues = append(issues, fmt.Sprintf("规则%d: 覆盖范围%d-%d超出目标范围", i, minPort, maxPort))
		}

		// 标记覆盖的端口
		for port := minPort; port <= maxPort; port++ {
			if port >= start && port <= end {
				index := port - start
				if coverage[index] {
					issues = append(issues, fmt.Sprintf("端口%d被多个规则覆盖", port))
				}
				coverage[index] = true
			}
		}
	}

	// 检查是否有未覆盖的端口
	for i, covered := range coverage {
		if !covered {
			issues = append(issues, fmt.Sprintf("端口%d未被覆盖", start+i))
		}
	}

	return len(issues) == 0, issues
}

// 统计规则效率
func AnalyzeEfficiency(rules []PortMask, start, end int) {
	totalPorts := end - start + 1
	coveredPorts := 0

	for _, rule := range rules {
		minPort, maxPort := getCoverage(rule.Base, rule.Mask)
		rulePorts := maxPort - minPort + 1
		// 只计算在目标范围内的端口
		actualMin := max(minPort, start)
		actualMax := min(maxPort, end)
		actualPorts := actualMax - actualMin + 1
		coveredPorts += actualPorts

		efficiency := float64(actualPorts) / float64(rulePorts) * 100
		fmt.Printf("规则 %d/0x%04X: 覆盖%d-%d (%d端口), 效率: %.1f%%\n",
			rule.Base, rule.Mask, minPort, maxPort, rulePorts, efficiency)
	}

	fmt.Printf("\n总计: %d个规则覆盖%d/%d个端口 (%.1f%%)\n",
		len(rules), coveredPorts, totalPorts, float64(coveredPorts)/float64(totalPorts)*100)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func main() {
	testCases := []struct {
		start, end int
		desc       string
	}{
		{80, 80, "单个端口"},
		{80, 95, "16端口范围"},
		{1000, 1999, "1000端口范围"},
		{1024, 65535, "大范围"},
		{0, 65535, "所有端口"},
		{8080, 8088, "小范围"},
		{3000, 3010, "不规则范围"},
		{1, 7, "小范围测试"},
	}

	fmt.Println("=== 最小规则算法 V1 ===")
	for _, tc := range testCases {
		rules := ConvertPortRangeToMinimalMasks(tc.start, tc.end)
		valid, issues := ValidateRules(rules, tc.start, tc.end)

		fmt.Printf("\n%s: %d-%d -> %d个规则\n", tc.desc, tc.start, tc.end, len(rules))
		for i, rule := range rules {
			minPort, maxPort := getCoverage(rule.Base, rule.Mask)
			fmt.Printf("  规则%d: %d/0x%04X (覆盖%d-%d)\n", i+1, rule.Base, rule.Mask, minPort, maxPort)
		}

		if !valid {
			fmt.Printf("  ⚠ 验证问题: %v\n", issues)
		} else {
			fmt.Printf("  ✓ 验证通过\n")
		}
		AnalyzeEfficiency(rules, tc.start, tc.end)
	}

	fmt.Println("\n=== 最小规则算法 V2 ===")
	for _, tc := range testCases {
		rules := ConvertPortRangeToMinimalMasksV2(tc.start, tc.end)
		valid, issues := ValidateRules(rules, tc.start, tc.end)

		fmt.Printf("\n%s: %d-%d -> %d个规则\n", tc.desc, tc.start, tc.end, len(rules))
		for i, rule := range rules {
			minPort, maxPort := getCoverage(rule.Base, rule.Mask)
			fmt.Printf("  规则%d: %d/0x%04X (覆盖%d-%d)\n", i+1, rule.Base, rule.Mask, minPort, maxPort)
		}

		if !valid {
			fmt.Printf("  ⚠ 验证问题: %v\n", issues)
		} else {
			fmt.Printf("  ✓ 验证通过\n")
		}
	}

	fmt.Println("\n=== 最优算法 ===")
	for _, tc := range testCases {
		rules := ConvertPortRangeToMinimalMasksOptimal(tc.start, tc.end)
		valid, issues := ValidateRules(rules, tc.start, tc.end)

		fmt.Printf("\n%s: %d-%d -> %d个规则\n", tc.desc, tc.start, tc.end, len(rules))
		for i, rule := range rules {
			minPort, maxPort := getCoverage(rule.Base, rule.Mask)
			fmt.Printf("  规则%d: %d/0x%04X (覆盖%d-%d)\n", i+1, rule.Base, rule.Mask, minPort, maxPort)
		}

		if !valid {
			fmt.Printf("  ⚠ 验证问题: %v\n", issues)
		} else {
			fmt.Printf("  ✓ 验证通过\n")
		}

		// 比较不同算法的规则数量
		rulesV1 := ConvertPortRangeToMinimalMasks(tc.start, tc.end)
		rulesV2 := ConvertPortRangeToMinimalMasksV2(tc.start, tc.end)
		fmt.Printf("  规则数量比较: V1=%d, V2=%d, 最优=%d\n",
			len(rulesV1), len(rulesV2), len(rules))
	}
}
