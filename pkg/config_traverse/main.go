package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

//遍历配置目录下的所有toml文件，将它们的内容合并到AppConfig中，最后打印AppConfig的内容
//以config.toml为基础，遍历configDir目录下的所有toml文件，将它们的内容合并到AppConfig中，最后打印AppConfig的内容

type AppConfig struct {
	Server struct {
		Port int `toml:"port"`
	} `toml:"server"`
}

func main() {
	configPath := "config.toml"
	configDir := "./"
	TraverseConfig(configPath, configDir)
}

func TraverseConfig(configPath, configDir string) {
	config := &AppConfig{}
	_, err := toml.DecodeFile(configPath, config)
	if err != nil {
		log.Fatalf("failed to decode config file: %v", err)
	}

	err = filepath.Walk(configDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		fmt.Printf("load config file: %q\n", path)
		if filepath.Ext(path) == ".toml" {
			_, err := toml.DecodeFile(path, config)
			return err
		}
		return nil
	})
	if err != nil {
		fmt.Print(err)
		return
	}
	log.Printf("config: %v", config)
}
