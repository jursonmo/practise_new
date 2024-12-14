package main

import (
	"crypto/tls"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var reconnectTimes int

func main() {
	opts := mqtt.NewClientOptions().AddBroker("tls://192.168.134.128:1883")
	opts.SetClientID("go_mqtt_client")
	opts.SetUsername("user1")
	opts.SetPassword("user1")
	opts.SetCleanSession(true)
	opts.SetTLSConfig(&tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12})
	opts.SetAutoReconnect(true)
	opts.SetConnectRetryInterval(1 * time.Second)
	opts.SetConnectTimeout(1 * time.Second)
	opts.SetKeepAlive(10 * time.Second)

	// 设置连接成功的回调
	opts.OnConnect = func(c mqtt.Client) {
		fmt.Println("Connected to broker")
		if token := c.Subscribe("test", 0, func(c mqtt.Client, m mqtt.Message) {}); token.Wait() && token.Error() != nil {
			fmt.Printf("Subscribe error: %v\n", token.Error())
			panic("subscribe error")
		}
		fmt.Printf("Subscribed to test\n")
	}

	// 设置断开连接的回调
	opts.OnConnectionLost = func(c mqtt.Client, err error) {
		fmt.Printf("Connection lost: %v\n", err)
		if reconnectTimes > 10 {
			panic("reconnectTimes > 10")
		}
		reconnectTimes++
		fmt.Printf("reconnectTimes:%d, err:%v\n", reconnectTimes, err)
		//reconnect(c) // 这里尝试重新连接，会有问题，有时服务器已经停止了mosquitto, 但是 c.Connect()返回连接成功
	}
	// 设置重新连接的回调
	opts.OnReconnecting = func(c mqtt.Client, opts *mqtt.ClientOptions) {
		fmt.Println("Reconnecting...")
	}

	client := mqtt.NewClient(opts)

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		fmt.Printf("Connect error: %v\n", token.Error())
		return
	}

	// 其他逻辑，比如订阅、发布消息等...

	// systemctl stop mosquitto, 查看是否断开连接，是否尝试重连
	// systemctl start mosquitto, 查看是否重连成功.
	// 等待程序退出
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		fmt.Println("Received signal, exiting...")
		client.Disconnect(250) // 断开连接, netstat -anp | grep 1883 查看是否断开连接
		//os.Exit(0)
	}()
	select {} // 阻塞主线程
}

func reconnect(c mqtt.Client) {
	for {
		// 这里尝试重新连接，有时会有问题，服务器已经停止了mosquitto, 但是 c.Connect()返回连接成功
		if token := c.Connect(); token.Wait() && token.Error() == nil {
			fmt.Println("Reconnected to broker ok")
			break
		}
		fmt.Println("Reconnect failed, retrying in 5 seconds...")
		time.Sleep(5 * time.Second)
	}
}
