package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/zeromicro/go-zero/core/logx"
)

/*
publish时，retained 参数有什么用？

在 MQTT 中，retained 参数用于指示消息是否应该被代理（broker）保存为保留消息。
其主要作用和特性如下：

1.保留消息：当你发布一条消息并将 retained 参数设置为 true 时，代理会保存这条消息并将其关联到对应的主题（topic）。
此后，任何新订阅该主题的客户端都会立即收到这条保留消息，而无需等待后续的消息发布。

2. 使用场景：retained 消息通常用于那些需要在客户端订阅后立即获得最新状态信息的场景。
例如，设备的状态、传感器的最后读数等信息，可以在客户端首次连接时快速获取。

3. 更新保留消息：如果同一主题上发布了新的保留消息，则旧的保留消息会被新的保留消息替代。
如果发布的消息内容为空（也就是 -m ""），则会删除该主题的保留消息。

4. QoS 级别：保留消息可以与任何 QoS 级别一起使用，但在 QoS 1 或 QoS 2 的情况下，代理会确保这些保留消息的传递和确认。
*/

var tlsbroker = "tls://192.168.134.128:1883"
var tcpbroker = "tcp://192.168.134.128:1883"
var alltopic = "test/#"
var topic = "test/topic/1"
var qos1 = byte(1)

func messagePubHandler(client mqtt.Client, msg mqtt.Message) {
	//message ID 是由 MQTT 协议自动生成的，用于跟踪 QoS 级别 1 或 2 的消息。
	fmt.Printf("Received messageID:%d, message: %s from topic: %s\n", msg.MessageID(), msg.Payload(), msg.Topic())
}

// 测试retained的发布, 先发布消息，再订阅消息，看能不能收到消息，收到哪条消息
func main() {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true, // 仅用于测试，生产环境中应设置为 false
		MinVersion:         tls.VersionTLS12,
	}
	_ = tlsConfig
	// 如果 InsecureSkipVerify: false, 则需要设置ca证书
	//caCert, err := os.ReadFile("/etc/mosquitto/certs/ca.crt")
	caCert, err := os.ReadFile("./ca.crt")
	if err != nil {
		log.Fatal(err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)
	tlsConfig.RootCAs = caCertPool

	clientId := "go_mqtt_client"
	opts := mqtt.NewClientOptions().AddBroker(tlsbroker).SetClientID(clientId)
	opts.SetKeepAlive(60 * time.Second)
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.SetPingTimeout(1 * time.Second)
	// /etc/mosquitto/mosquitto.conf:
	//allow_anonymous true
	//password_file /etc/mosquitto/pwdfile
	//表示允许匿名连接和使用密码文件里用户和密码可以登录，如果client opts.SetUsername("user1"), 即使用用户名密码方式验证，则用户名和密码必须正确。
	opts.SetUsername("user1")
	opts.SetPassword("user1")

	// /etc/mosquitto/mosquitto.conf:增加下面配置后，就表示启用了tls
	// cafile /etc/mosquitto/certs/ca.crt
	// certfile /etc/mosquitto/certs/server.crt
	// keyfile /etc/mosquitto/certs/server.key
	opts.SetTLSConfig(tlsConfig) // 设置 TLS 配置, 使用tls协议连接broker, tls://x.x.x:1883

	c := mqtt.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		fmt.Println("connect err:", token.Error())
		os.Exit(1)
	}
	opt := c.OptionsReader()
	urls := opt.Servers()
	logx.Infof("urls: %v\n", urls)

	// 发布not retained消息
	retained := false
	text := "Hello MQTT not retained"
	if token := c.Publish(topic, qos1, retained, text); token.Wait() && token.Error() != nil {
		fmt.Println("publish err:", token.Error())
		return
	}

	// 发布第一条retained消息
	retained = true
	text = "Hello MQTT retained"
	if token := c.Publish(topic, qos1, retained, text); token.Wait() && token.Error() != nil {
		fmt.Println("publish err:", token.Error())
		return
	}

	// 发布第二条retained消息
	retained = true
	text = "Hello MQTT retained 2"
	if token := c.Publish(topic, qos1, retained, text); token.Wait() && token.Error() != nil {
		fmt.Println("publish err:", token.Error())
		return
	}

	time.Sleep(1 * time.Second)
	//订阅alltopic
	if token := c.Subscribe(alltopic, qos1, nil); token.Wait() && token.Error() != nil {
		fmt.Println("subscribe err:", token.Error())
		return
	}
	//订阅完后，看能不能收到消息，收到哪条消息
	fmt.Println("subscribed, wait for message...")
	//目前测试的结果是，订阅后，会收到所有发布过的消息，包括最后一条retained的消息
	select {}
}
