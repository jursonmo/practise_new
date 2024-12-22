以下是一个用 Go 实现 MQTT 遗嘱消息（Last Will and Testament，LWT）的简单示例。这个例子使用了 Eclipse Paho MQTT 客户端库。

示例代码
```go
package main

import (
	"fmt"
	"time"

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
```
#### 代码说明
设置遗嘱消息：
使用 SetWill(topic, payload, qos, retained) 方法设置遗嘱消息，其中：
1. topic 是遗嘱消息的主题。
2. payload 是遗嘱消息的内容。
3. qos 是消息的 QoS 等级。
4. retained 指定消息是否被保留。遗嘱 Retain 为 true 时表明遗嘱消息是保留消息。MQTT 服务器会为每个主题存储最新一条保留消息，以方便消息发布后才上线的客户端在订阅主题时仍可以接收到该消息。(莫：意思是客户端断开后，服务器会将遗嘱消息发送给订阅了该主题的其他在线客户端。设置retained为true时，服务器会保留该遗嘱消息，当有新的订阅者连接时，会立即收到最后一条遗嘱消息。)

客户端正常连接时不会发送遗嘱消息：
在客户端正常连接和断开时，遗嘱消息不会被发送。

异常断开连接时发送遗嘱消息：
如果客户端由于网络中断或进程崩溃等原因意外断开，MQTT Broker 将自动发送遗嘱消息。

#### 测试步骤
1. 运行代码，并观察是否正常连接到 test.mosquitto.org。
2. 使用其他 MQTT 客户端（如 MQTT Explorer）订阅 test/lwt 主题。
3. 当程序执行到模拟断开部分时，观察 MQTT Broker 是否发布遗嘱消息。

#### 注意事项
如果需要更强大的测试环境，可以搭建自己的 MQTT Broker。
遗嘱消息的内容和使用场景需根据业务需求设计，例如提示客户端异常断线、触发自动恢复等。
如果有其他需求，可以继续交流！

### 遗嘱消息 介绍: https://www.emqx.com/zh/blog/use-of-mqtt-will-message
#### MQTT 3.1.1 中的遗嘱消息
在 MQTT 3.1.1 中，只要网络连接在服务端没有收到 DISCONNECT 报文的情况下关闭，服务端都需要发布遗嘱消息。

由于 MQTT 3.1.1 没有 Will Delay Interval，也没有 Session Expiry Interval，所以遗嘱消息总是在网络连接关闭时立即发布。

#### MQTT 5.0, Will Delay Interval 与延迟发布
默认情况下，服务端总是在网络连接意外关闭时立即发布遗嘱消息。但是很多时候，网络连接的中断是短暂的，所以客户端往往能够重新连接并继续之前的会话。这导致遗嘱消息可能被频繁地且无意义地发送。

所以 MQTT 5.0 专门为遗嘱消息增加了一个 Will Delay Interval 属性，这个属性决定了服务端将在网络连接关闭后延迟多久发布遗嘱消息，并以秒为单位。

如果没有指定 Will Delay Interval 或者将其设置为 0，服务端将仍然在网络连接关闭时立即发布遗嘱消息。

但如果将 Will Delay Interval 设置为一个大于 0 的值，并且客户端能够在 Will Delay Interval 到期前恢复连接，那么该遗嘱消息将不会被发布。

#### 为什么没有收到遗嘱消息？
1. 连接意外关闭且 Will Delay Interval 等于 0，遗嘱消息将在网络连接关闭时立即发布
2. 连接意外关闭且 Will Delay Interval 大于 0，遗嘱消息将被延迟发布，最大延迟时间取决于 Will Delay Interval 与 Session Expiry Interval 谁先到期：
+ 客户端未能在 Will Delay Interval 或 Session Expiry Interval 到期前恢复连接，遗嘱消息将被发布。
+ 在 Will Delay Interval 或 Session Expiry Interval 到期前:
 客户端指定 Clean Start 为 0 恢复连接，遗嘱消息将不会被发布。
 客户端指定 Clean Start 为 1 恢复连接，遗嘱消息将因为 现有会话结束 而被立即发布。