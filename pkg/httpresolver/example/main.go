package main

import (
	"io"
	"log"

	"github.com/jursonmo/practise_new/pkg/httpresolver"
)

func main() {
	client := httpresolver.NewHttpResolverClient(nil, func(host string, ips []string, inPreset, inCache bool) {
		log.Printf("host:%s resolve ips: %v, inPreset:%v, inCache:%v", host, ips, inPreset, inCache)
	})
	resp, err := client.Get("http://www.baidu.com")
	if err != nil {
		log.Printf("get err:%v", err)
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("read body err:%v", err)
		return
	}
	log.Printf("body:%s", body)

	// 设置一个公网不存在的域名来测试，测试如果系统解析不到ip, 会不会使用指定的ip, 顺便测试下后缀匹配
	httpresolver.AddDomainIp("sfasdfasdfasdf.com", "127.0.0.1")
	_, err = client.Get("https://xxx.sfasdfasdfasdf.com")
	if err != nil {
		log.Printf("get err:%v", err)
		//return
	}
	//验证结果能解析到指定的ip.
	// 2025/07/07 17:27:59 host:xx.sfasdfasdfasdf.com resolve ips: [127.0.0.1]
	// 2025/07/07 17:27:59 get err:Get "https://xx.sfasdfasdfasdf.com": dial tcp 127.0.0.1:443: connect: connection refused

	//下面验证LoadDomains
	httpresolver.LoadDomains("domains.txt")
	_, err = client.Get("https://xxx.testloaddomains.com")
	if err != nil {
		log.Printf("get err:%v", err)
		return
	}
}
