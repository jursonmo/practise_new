package main

import (
	"fmt"
	"log"

	"dario.cat/mergo"
	"github.com/jinzhu/copier"
)

type redisConfig struct {
	Address string
	Port    int
	DB      int
}

var defaultConfig = redisConfig{
	Address: "127.0.0.1",
	Port:    6381,
	DB:      1,
}

type redisConfig2 struct {
	Address string
	Port    int
	DB2     int
}

func main() {
	var config redisConfig

	if err := mergo.Merge(&config, defaultConfig); err != nil {
		log.Fatal(err)
	}

	fmt.Println("redis address: ", config.Address)
	fmt.Println("redis port: ", config.Port)
	fmt.Println("redis db: ", config.DB)

	var m = make(map[string]interface{})
	if err := mergo.Map(&m, defaultConfig); err != nil {
		log.Fatal(err)
	}

	fmt.Println(m)

	var config2 redisConfig2
	//测试不同的类型, 这样会报错：src and dst must be of same type
	//合并数据（如配置默认值和用户输入），推荐使用 mergo。
	// if err := mergo.Merge(&config2, defaultConfig); err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Println("config2:", config2)

	//不同类型的数据, 使用 copier 合并
	err := copier.Copy(&config2, defaultConfig)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("copier config2:%+v\n", config2)
}
