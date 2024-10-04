package myhttp

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

/*
golang 官方提供的 http 包里的 http client 可以通过一下两种方法设置超时
（其他一些精细的超时控制比如设置 transport 里的 dial connect 的超时时间不在这里讨论）。
1. client.Timeout
// 设置1s超时
cli := http.Client{Timeout: time.Second}

2. req.WithContext
// 设置1s超时
ctx, _  := context.WithTimeout(time.Second)
req := http.NewRequest(....)
req.WithContext(ctx)

超时时间都包括 链接建立，请求发送，读取返回。如果没有及时读取resp.Body，都会引起超时错误。
*/
type ReqHandler func(req *http.Request) error

func SetContentType(contentType string) ReqHandler {
	return func(req *http.Request) error {
		req.Header.Set("Content-Type", contentType)
		return nil
	}
}

// 由ctx来控制超时
func Request(ctx context.Context, method string, reqObject interface{}, respObject interface{}, api string, reqhs ...ReqHandler) error {
	var reader io.Reader
	var err error
	var data []byte
	var resp *http.Response

	if reqObject != nil {
		data, err := json.Marshal(reqObject)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}

	//ctx 可以包含超时
	req, err := http.NewRequestWithContext(ctx, method, api, reader)
	if err != nil {
		return err
	}
	//req = req.WithContext(ctx) //生成一个新的req 对象
	//默认设置“Content-Type" 为 "application/json”, 可以通过ReqHandler来修改
	req.Header.Set("Content-Type", "application/json")
	for _, h := range reqhs {
		if err := h(req); err != nil {
			return err
		}
	}

	//client := &http.Client{Timeout: 5 * time.Second}
	client := &http.Client{}
	//client.Transport = DefaultTransport() //不需要每次都创建一个新的transport,这样请求没法复用连接

	if err = ctx.Err(); err != nil {
		return err
	}

	resp, err = client.Do(req)
	if err != nil {
		return err
	}

	if resp.Header.Get("Content-Encoding") == "gzip" {
		var reader *gzip.Reader
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			return err
		}
		data, err = io.ReadAll(reader)
	} else {
		//alway read resp.Body to reuse tcp connection
		data, err = io.ReadAll(resp.Body)
	}
	//alway read resp.Body to reuse tcp connection
	//body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return err
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode > 299 {
		err = fmt.Errorf("resp.StatusCode:%d, api:%s, data:%v", resp.StatusCode, api, string(data))
		return err
	}

	if /*method == http.MethodGet &&*/ respObject != nil {
		fmt.Println("---------------------------")
		fmt.Printf("request:%v \n response len:%d, data:%s\n", req.URL, len(data), string(data))
		fmt.Println("---------------------------")
		err = json.Unmarshal(data, respObject)
		if err != nil {
			return err
		}
	}
	return nil
}
