package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

type Config struct {
	AppName  string
	LogLevel string

	MySQL MySQLConfig
	Redis RedisConfig
}

type MySQLConfig struct {
	IP       string
	Port     int
	User     string
	Password string
	Database string
}

type RedisConfig struct {
	IP   string
	Port int
}

var file string = "config.toml"

func main() {
	log.Printf("config file: %s", filepath.Base(file))
	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		log.Printf("read config failed: %v", err)
	}

	fmt.Println("mysql ip: ", viper.GetString("mysql.ip")) //viper默认是case-insensitive的，所以这里可以用mysql.ip 或 MySQL.IP 或 mysql.IP 都可以
	fmt.Println("mysql port: ", viper.GetInt("mysql.port"))

	if viper.IsSet("redis.port") {
		fmt.Println("redis.port is set")
	} else {
		fmt.Println("redis.port is not set")
	}

	viper.WatchConfig()

	viper.OnConfigChange(func(e fsnotify.Event) {
		fmt.Printf("Config file:%s Op:%s\n", e.Name, e.Op)
		fmt.Println("redis port changed: ", viper.Get("redis.port"))
	})
	fmt.Println("redis port before sleep: ", viper.Get("redis.port"))
	time.Sleep(time.Second * 10)
	fmt.Println("redis port after sleep: ", viper.Get("redis.port"))

	viper.Set("redis.port", 5381) //Set()这个优先级最高，高于配置文件，env, flag里读取的, 也就是配置里修改了redis.port 为其他值后, viper.Get("redis.port") 会返回5381
	viper.Set("redis.port2", 5382)
	viper.WriteConfig()

	c := Config{}
	err = viper.Unmarshal(&c)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("config: ", c)
	fmt.Println("all settings: ", viper.AllSettings())

	//侦听退出信号
	QuitSignal := QuitSignal()
	select {
	case <-QuitSignal:
		fmt.Println("quit signal received")
	}
}
func QuitSignal() <-chan os.Signal {
	signals := make(chan os.Signal, 4)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGKILL)
	return signals
}
