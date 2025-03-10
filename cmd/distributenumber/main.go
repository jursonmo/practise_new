package main

import (
	"fmt"
)

// 分发m个数字到n个桶中
// 假如有8个数字，尽量平均放到桶里，如果只有一个桶，那么这个桶存放的数字是[0,1,2,3,4,5,6,7]；
// 如果有两个桶，那么就每个桶放四个，第一个桶放[0,1,2,3]， 第二个桶放[4,5,6,7];
// 如果有个三个桶，那么第一个桶放[0,1,2], 第二桶放[3,4,5],第三个桶[6,7],以此类推。
func DistributeNumbers(m, n int) [][]int {
	numbers := make([]int, 0, m)
	for i := 0; i < m; i++ {
		numbers = append(numbers, i)
	}

	if n <= 0 {
		return nil // 或者返回空，根据需求处理无效输入
	}

	result := make([][]int, 0, n)

	base := m / n
	rem := m % n

	start := 0
	for i := 0; i < n; i++ {
		size := base
		if i < rem {
			size++
		}

		end := start + size
		if end > m {
			end = m
		}

		bucket := numbers[start:end]
		result = append(result, bucket)
		start = end
	}

	return result

}

func main() {
	// 测试用例
	numbersCount := 8
	testCases := []int{1, 2, 3, 5, 8, 9}
	for _, n := range testCases {
		fmt.Printf("n=%d:\n", n)
		buckets := DistributeNumbers(numbersCount, n)
		for i, b := range buckets {
			fmt.Printf("Bucket %d: %v\n", i, b)
		}
		fmt.Println()
	}
}
