package httpx

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type RequestOpt func(req *http.Request) error

func Request(ctx context.Context, method string, api string, reqObject interface{}, respObject interface{}, opts ...RequestOpt) error {
	var reader io.Reader
	var err error
	if reqObject != nil {
		data, err := json.Marshal(reqObject)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, api, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for _, h := range opts {
		if err := h(req); err != nil {
			return err
		}
	}

	var data []byte
	var resp *http.Response
	client := &http.Client{Timeout: 5 * time.Second}
	//client.Transport = DefaultTransport() //不需要每次都创建一个新的transport,这样请求没法复用连接

	if strings.Contains(api, "https://") {
		// 如果服务器https, 测试时可以忽略证书验证, TODO: 生产环境需要去掉
		transport := http.DefaultTransport.(*http.Transport) //修改默认的transport, 会导致这个程序的所有请求都忽略证书验证
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true, // 跳过证书验证
		}
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
	defer resp.Body.Close()
	if err != nil {
		return err
	}

	//‌HTTP状态码的范围包括100-599，其中100-199表示信息响应，200-299表示成功响应，300-399表示重定向，400-499表示客户端错误，500-599表示服务器错误。
	if resp.StatusCode < http.StatusOK || resp.StatusCode > 299 {
		// 如果服务器采用CodeResponse的方式回应，那么什么验证失败或者其他错误，都返回CodeResponser这个对象且 http statusCode为200。
		// 如果服务器部分错误忘记用CodeResponser回应, 客户端也会返回CodeResponser这个对象且Code为未知错误CodeError。
		if respObject != nil {
			//如果respObject实现了Code接口, 则设置code和msg, 返回err nil, 业务需要根据code和msg进行进一步判断处理
			if r, ok := respObject.(CodeResponser); ok {
				//r.SetCode(resp.StatusCode)
				r.SetCode(CodeError) //如果服务器返回的statusCode不是200, 则设置code为CodeError
				r.SetMsg(fmt.Sprintf("resp.StatusCode:%d, api:%s, data:%v", resp.StatusCode, api, string(data)))
				return nil
			}
		}
		return fmt.Errorf("resp.StatusCode:%d, api:%s, data:%v", resp.StatusCode, api, string(data))
	}

	if /*method == http.MethodGet &&*/ respObject != nil {
		err = json.Unmarshal(data, respObject)
		if err != nil {
			return err
		}
	}
	return nil
}
