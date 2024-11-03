package main

import (
	"fmt"
	"os"
	"os/signal"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var broker = "tcp://localhost:1883"
var alltopic = "test/#"
var topic = "test/topic/1"
var topic2 = "test/topic/2"
var qos1 = byte(1)
var qos2 = byte(2)

func main() {
	clientId := "go_mqtt_client"
	opts := mqtt.NewClientOptions().AddBroker(broker).SetClientID(clientId)
	opts.SetKeepAlive(60 * time.Second)
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.SetPingTimeout(1 * time.Second)

	c := mqtt.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		fmt.Println("connect err:", token.Error())
		os.Exit(1)
	}

	// 开始订阅
	if token := c.Subscribe(alltopic, qos1, nil); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}
	fmt.Printf("Subscribed to %s\n", alltopic)

	sysTopic := "$SYS/broker/clients/" + clientId
	if token := c.Subscribe(sysTopic, qos1, nil); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}

	fmt.Printf("Subscribed to %s\n", sysTopic)

	// 发布消息
	text := "Hello MQTT"
	token := c.Publish(topic, qos1, false, text)
	token.Wait()
	fmt.Printf("Published %s message: %s\n", topic, text)

	text = "Hello MQTT 222"
	token = c.Publish(topic2, qos1, false, text)
	token.Wait()
	fmt.Printf("Published %s message: %s\n", topic2, text)

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	<-ch

	// 取消订阅
	if token := c.Unsubscribe(topic); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}
	fmt.Println("Unsubscribed from", topic)

	c.Disconnect(250)
	fmt.Println("Client Disconnected")
}

func messagePubHandler(client mqtt.Client, msg mqtt.Message) {
	//message ID 是由 MQTT 协议自动生成的，用于跟踪 QoS 级别 1 或 2 的消息。
	fmt.Printf("Received messageID:%d, message: %s from topic: %s\n", msg.MessageID(), msg.Payload(), msg.Topic())
}
