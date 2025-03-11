package main

import (
	"fmt"

	"github.com/zeromicro/go-zero/core/limit"

	"github.com/zeromicro/go-zero/core/stores/redis"
)

func main() {
	store := redis.New("localhost:6379")
	limiter := limit.NewPeriodLimit(1, 2, store, "exampleKey")

	f := func() {
		result, err := limiter.Take("user1")
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		switch result {
		case limit.Allowed:
			fmt.Println("Request allowed")
		case limit.HitQuota:
			fmt.Println("Hit the quota")
		case limit.OverQuota:
			fmt.Println("Over the quota")
		default:
			fmt.Println("Unknown status")
		}
	}

	fmt.Printf("take 1\n")
	f()
	fmt.Printf("take 2\n")
	f()
	fmt.Printf("take 3\n")
	f()
}
