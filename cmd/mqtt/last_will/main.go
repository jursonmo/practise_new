package main

import (
	"fmt"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func main() {
	broker := "tcp://test.mosquitto.org:1883"
	clientID := "go-mqtt-example"
	topic := "test/lwt"

	// 遗嘱消息
	willMessage := "Client disconnected unexpectedly"
	willTopic := topic

	// 设置客户端选项，包括遗嘱消息
	opts := mqtt.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID(clientID)
	opts.SetWill(willTopic, willMessage, 1, false)

	// 创建 MQTT 客户端
	client := mqtt.NewClient(opts)

	// 连接到 MQTT Broker
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		fmt.Printf("Failed to connect: %v\n", token.Error())
		return
	}
	fmt.Println("Connected to MQTT broker")

	// 发布正常消息
	token := client.Publish(topic, 1, false, "Client connected successfully")
	token.Wait()

	// 模拟异常退出
	fmt.Println("Simulating unexpected disconnection...")
	client.Disconnect(250)

	// 遗嘱消息将在客户端异常断开连接时由 MQTT Broker 发送
	fmt.Println("Done")
}
