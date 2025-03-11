package main

import (
	"fmt"

	"github.com/zeromicro/go-zero/core/limit"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

func main() {
	store, err := redis.NewRedis(redis.RedisConf{Host: "127.0.0.1:6379", Type: redis.NodeType})
	if err != nil {
		panic(err)
	}
	limiter := limit.NewTokenLimiter(3, 1, store, "example-key")

	if limiter.Allow() {
		fmt.Println("Request allowed")
	} else {
		fmt.Println("Request not allowed")
	}

	if limiter.Allow() {
		fmt.Println("Request2 allowed")
	} else {
		fmt.Println("Request2 not allowed")
	}

	if limiter.Allow() {
		fmt.Println("Request3 allowed")
	} else {
		fmt.Println("Request3 not allowed")
	}
}
