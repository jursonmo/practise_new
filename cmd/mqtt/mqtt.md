### mac安装mosquitto
https://mosquitto.org/download/

brew install mosquitto
brew services start mosquitto
brew services info mosquitto  --json

 ps -ef|grep mosquitto
  501   618     1   0  9:14上午 ??         0:12.14 /opt/homebrew/opt/mosquitto/sbin/mosquitto -c /opt/homebrew/etc/mosquitto/mosquitto.conf
### linux(ubuntu)安装mosquitto
#### 给安装源增加存储库

sudo apt-add-repository ppa:mosquitto-dev/mosquitto-ppa
#### 更新安装源
sudo apt update 
#### 使用apt安装mosquiito

sudo apt-get install mosquitto
#### 启动重启和停止服务

sudo /etc/init.d/mosquitto restart/stop/start

#### 修改配置文件
sudo vim /etc/mosquitto/mosquitto.conf
listener 1883 0.0.0.0 #让mosquitto监听0.0.0.0:1883端口

### MQTT 协议中的 message ID
#### 在 MQTT 协议中，message ID 的特性在不同的 QoS 级别中有所不同。在 QoS 级别 1 时，message ID 具有以下特性：

1. 消息重复可能性：在 QoS 1（即 “至少一次”）的情况下，消息发送方确保消息至少会被成功传递到接收方一次。但是，由于接收方的应答可能因网络延迟或其他原因未及时到达发送方，发送方可能会重发消息。因此，接收方可能会收到重复的消息。

2. 消息 ID 用于去重：message ID 用于唯一标识某个 QoS 1 消息传递过程中的消息。接收方通过这个 message ID 来识别消息是否为重复消息，从而可以在应用层采取去重措施。

消息传递流程：在 QoS 1 中，消息的传递过程如下：

发送方发送包含 message ID 的 PUBLISH 报文。
接收方在接收到消息后，会以该 message ID 回复 PUBACK 报文，以确认消息已经接收。
一旦发送方收到 PUBACK，则认为消息成功传递，并停止重发。
4. message ID 范围：message ID 是一个 16 位的整数，范围在 1 到 65535 之间。MQTT 客户端会自动管理和分配这些 ID，确保每个未完成的 QoS 1 消息在发送和确认过程中具有唯一的 message ID。

5. 总结来说，在 QoS 1 下，message ID 的主要作用是确保消息至少传递一次，同时让接收方有能力检测并去除重复消息。

### publish时，retained 参数有什么用？

在 MQTT 中，retained 参数用于指示消息是否应该被代理（broker）保存为保留消息。
其主要作用和特性如下：

1.保留消息：当你发布一条消息并将 retained 参数设置为 true 时，代理会保存这条消息并将其关联到对应的主题（topic）。
此后，任何新订阅该主题的客户端都会立即收到这条保留消息，而无需等待后续的消息发布。

2. 使用场景：retained 消息通常用于那些需要在客户端订阅后立即获得最新状态信息的场景。
例如，设备的状态、传感器的最后读数等信息，可以在客户端首次连接时快速获取。

3. 更新保留消息：如果同一主题上发布了新的保留消息，则旧的保留消息会被新的保留消息替代。
如果发布的消息内容为空（也就是 -m ""），则会删除该主题的保留消息。

4. QoS 级别：保留消息可以与任何 QoS 级别一起使用，但在 QoS 1 或 QoS 2 的情况下，代理会确保这些保留消息的传递和确认。

### 使用用户名密码连接, -c 意思是删除旧的文件，创建新的passwordfile， 可以认为是覆盖。
1. mosquitto_passwd -c /path/to/passwordfile username1
在提示下输入 username1 的密码。

2. 添加其他用户
接下来，使用相同的命令但不加 -c 选项来添加其他用户：
mosquitto_passwd /path/to/passwordfile username2

3. 启用用户名和密码: 在配置文件/etc/mosquitto/mosquitto.conf中添加:
password_file /path/to/passwordfile

4. 如果在 /etc/mosquitto/mosquitto.conf 配置文件里添加了如下配置:
	allow_anonymous true
	password_file /etc/mosquitto/passwordfile
	表示允许匿名连接和使用密码文件里用户和密码可以登录，如果client opts.SetUsername("user1"), 即使用用户名密码方式验证，则用户名和密码必须正确。

#### tls 证书生成
# 创建 CA 证书
openssl genrsa -out ca.key 2048
openssl req -x509 -new -nodes -key ca.key -sha256 -out ca.crt -subj "/CN=My CA"

# 创建服务器密钥和证书请求
openssl genrsa -out server.key 2048
openssl req -new -key server.key -out server.csr -subj "/CN=YourMosquittoServer"

# 签署服务器证书
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt -sha256


#### 查看日志
/var/log/mosquitto/mosquitto.log 
