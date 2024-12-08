package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
)

// https 服务，不同请求的域名对应不同证书
func main() {
	// 设置 TLS 配置
	tlsConfig := &tls.Config{
		GetCertificate: func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
			// 根据 SNI (Server Name Indication) 选择证书
			switch info.ServerName {
			case "example.com":
				cert, err := tls.LoadX509KeyPair("example.com.crt", "example.com.key")
				if err != nil {
					return nil, err
				}
				return &cert, nil
			case "another-example.com":
				cert, err := tls.LoadX509KeyPair("another-example.com.crt", "another-example.com.key")
				if err != nil {
					return nil, err
				}
				return &cert, nil
			default:
				return nil, fmt.Errorf("no certificate found for domain %s", info.ServerName)
			}
		},
	}

	server := &http.Server{
		Addr:      ":443",
		TLSConfig: tlsConfig,
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, you've reached %s!\n", r.Host)
	})

	log.Println("Starting server on :443")
	err := server.ListenAndServeTLS("", "") // 使用空字符串，TLS 配置中已经处理了证书加载
	if err != nil {
		log.Fatalf("server failed: %s", err)
	}
}

/*
from chatgpt
代码说明：
	tls.Config 的 GetCertificate 回调函数会在每次客户端请求时被调用，传入的 tls.ClientHelloInfo 包含了请求的域名信息。
	你可以根据 info.ServerName 来选择相应的证书。
	使用 tls.LoadX509KeyPair 加载指定域名的证书和私钥。
	http.Server 的 TLSConfig 设置为自定义的 tls.Config，并使用 ListenAndServeTLS("", "") 启动 HTTPS 服务。
	注意事项：
	确保服务器上有对应域名的有效 TLS 证书和私钥文件。
	你需要为每个域名指定合适的证书路径。
	这个方案适合在同一台服务器上托管多个 HTTPS 域名，并确保每个域名都有不同的证书。
*/
