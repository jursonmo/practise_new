package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// 钉钉机器人消息结构体
// type DingTalkMsg struct {
// 	MsgType string `json:"msgtype"`
// 	Text    struct {
// 		Content string `json:"content"`
// 	} `json:"text"`
// 	At struct {
// 		AtMobiles []string `json:"atMobiles"` // 需要@的手机号列表
// 		IsAtAll   bool     `json:"isAtAll"`   // 是否@所有人
// 	} `json:"at"`
// }

type DingTalkMsg struct {
	MsgType string       `json:"msgtype"`
	Text    DingDingText `json:"text"`
	At      DingDingAt   `json:"at"`
}
type DingDingText struct {
	Content string `json:"content"`
}

type DingDingAt struct {
	AtMobiles []string `json:"atMobiles"` // 需要@的手机号列表
	IsAtAll   bool     `json:"isAtAll"`   // 是否@所有人
}

func main() {
	// 替换为你的钉钉机器人Webhook地址
	// webhookUrl := "https://oapi.dingtalk.com/robot/send?access_token=YOUR_TOKEN"
	webhookUrl := "https://oapi.dingtalk.com/robot/send?access_token=7d4cc198d70388163c856e8e5d6624796472589976764ede70c526850f5735b9"
	// 创建消息内容
	msg := DingTalkMsg{
		MsgType: "text",
		Text: DingDingText{
			Content: "obc：\n- 主机：192.168.1.100\n- 状态：CPU使用率超过95%\n- 时间：2023-08-25 15:30:00",
		},
		At: DingDingAt{
			AtMobiles: []string{"15013698104"}, // 要@的群成员手机号
			IsAtAll:   true,                    // 不@所有人
		},
	}

	// 将消息结构体转换为JSON
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}

	// 发送POST请求
	resp, err := http.Post(webhookUrl, "application/json", bytes.NewBuffer(msgBytes))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("发送失败，状态码: %d", resp.StatusCode)
		return
	}

	fmt.Println("告警消息已发送到钉钉群")
}
