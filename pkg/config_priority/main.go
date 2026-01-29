package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
)

// Config 结构体定义了所有可能的配置项
type Config struct {
	ServerHost string `json:"server_host"`
	ServerPort int    `json:"server_port"`
	DebugMode  bool   `json:"debug_mode"`
	MaxConn    int    `json:"max_connections"`
	LogLevel   string `json:"log_level"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() Config {
	return Config{
		ServerHost: "localhost",
		ServerPort: 8080,
		DebugMode:  false,
		MaxConn:    100,
		LogLevel:   "info",
	}
}

// LoadConfig 加载配置，优先使用配置文件中的值
func LoadConfig(configPath string) (Config, error) {
	config := DefaultConfig()
	log.Printf("default config:%#v\n", config)
	if configPath == "" {
		return config, nil
	}

	// 读取配置文件
	file, err := os.ReadFile(configPath)
	if err != nil {
		return config, fmt.Errorf("failed to read config file: %v", err)
	}

	// 解析配置文件，覆盖默认值
	err = json.Unmarshal(file, &config)
	if err != nil {
		return config, fmt.Errorf("failed to parse config file: %v", err)
	}

	log.Printf("config:%#v\n", config)
	return config, nil
}

func main() {
	// 解析命令行参数
	var configPath string
	flag.StringVar(&configPath, "c", "./config.json", "Path to configuration file")
	flag.Parse()

	log.Printf("configPath:%#v\n", configPath)
	// 加载配置
	config, err := LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// 打印最终配置
	fmt.Println("Final configuration:")
	fmt.Printf("ServerHost: %s\n", config.ServerHost)
	fmt.Printf("ServerPort: %d\n", config.ServerPort)
	fmt.Printf("DebugMode: %v\n", config.DebugMode)
	fmt.Printf("MaxConn: %d\n", config.MaxConn)
	fmt.Printf("LogLevel: %s\n", config.LogLevel)
}
