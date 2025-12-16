package dingding

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

/*
postman 测试 ProdWebhookUrl 返回结果是 200, 内容是:

	{
	    "errcode": 90030,
	    "errmsg": "您的企业本月webhook调用量已超过限制,请登录开发者后台-资源管理-查看用量明细。"
	}

由于发生钉钉消息出错时返回http code 200, 所以下面的代码没有报错，所以还要检查errcode是否为0
正常返回结果是:

	{
	    "errcode": 0,
	    "errmsg": "ok"
	}

为了以后方便更新webhookUrl，将其定义从文件中提取出来
*/
type DingDingResp struct {
	Errcode int    `json:"errcode"`
	Errmsg  string `json:"errmsg"`
}

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

type DingDing struct {
	TestWebhook DingDingWebhook `toml:"testWebhook"`
	ProdWebhook DingDingWebhook `toml:"prodWebhook"`
}

type DingDingWebhook struct {
	WebhookUrl  string `toml:"webhookUrl"`
	MsgType     string `toml:"msgType"`     // 消息类型，默认 text
	SafeKeyWord string `toml:"safeKeyWord"` // 如果设置了安全设置中的关键词, 发消息时，必须以此开头
	DingDingAt  `toml:"at"`
}

func (d *DingDing) SendMsg(msg string, isTest bool) error {
	// 选择 Webhook URL
	var webhook DingDingWebhook
	if isTest {
		webhook = d.TestWebhook
	} else {
		webhook = d.ProdWebhook
	}

	if webhook.MsgType == "" {
		webhook.MsgType = "text"
	}

	// 创建消息内容
	DDMsg := DingTalkMsg{
		MsgType: webhook.MsgType,
		Text: DingDingText{
			Content: fmt.Sprintf("%s\n - time:%v\n - msg:%s", webhook.SafeKeyWord, time.Now().Format("2006-01-02 15:04:05"), msg),
		},
		At: webhook.DingDingAt,
	}

	msgBytes, err := json.Marshal(DDMsg)
	if err != nil {
		return err
	}

	client := &http.Client{
		Timeout: 3 * time.Second,
	}
	req, err := http.NewRequest("POST", webhook.WebhookUrl, bytes.NewBuffer(msgBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("发送失败，状态码: %d", resp.StatusCode)
	}

	// 解析响应
	var dingResp DingDingResp
	if err := json.NewDecoder(resp.Body).Decode(&dingResp); err != nil {
		return err
	}
	// http code 200 不代表发送成功，需要进一步检查response 中的 errcode 是否为0
	if dingResp.Errcode != 0 {
		return fmt.Errorf("dingding send msg failed, errcode: %d, errmsg: %s", dingResp.Errcode, dingResp.Errmsg)
	}

	return nil
}
