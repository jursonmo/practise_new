package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/BurntSushi/toml"
)

// Config 结构体定义了所有可能的配置项
// 使用 toml 标签指定 TOML 文件中的字段名
type Config struct {
	Server struct {
		Host string `toml:"host"`
		Port int    `toml:"port"`
	} `toml:"server"`
	DebugMode bool   `toml:"debug_mode"`
	MaxConn   int    `toml:"max_connections"`
	LogLevel  string `toml:"log_level"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() Config {
	var config Config
	config.Server.Host = "localhost"
	config.Server.Port = 8080
	config.DebugMode = false
	config.MaxConn = 100
	config.LogLevel = "info"
	return config
}

// LoadConfig 加载配置，优先使用配置文件中的值
func LoadConfig(configPath string) (Config, error) {
	config := DefaultConfig()

	if configPath == "" {
		return config, nil
	}

	// 读取配置文件
	file, err := os.ReadFile(configPath)
	if err != nil {
		return config, fmt.Errorf("failed to read config file: %v", err)
	}

	// 解析 TOML 配置文件，覆盖默认值
	_, err = toml.Decode(string(file), &config)
	if err != nil {
		return config, fmt.Errorf("failed to parse config file: %v", err)
	}

	return config, nil
}

func main() {
	// 解析命令行参数
	var configPath string
	flag.StringVar(&configPath, "c", "./config.toml", "Path to configuration file (TOML format)")
	flag.Parse()

	// 加载配置
	config, err := LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// 打印最终配置
	fmt.Println("Final configuration:")
	fmt.Printf("Server.Host: %s\n", config.Server.Host)
	fmt.Printf("Server.Port: %d\n", config.Server.Port)
	fmt.Printf("DebugMode: %v\n", config.DebugMode)
	fmt.Printf("MaxConn: %d\n", config.MaxConn)
	fmt.Printf("LogLevel: %s\n", config.LogLevel)
}
