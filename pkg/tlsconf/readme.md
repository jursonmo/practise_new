### 生成 tls config
 GenTlsConfig 接口定义了生成 tls config 的方法。如果没有指定证书和密钥，将使用自签名证书。
 ```go
 tlsConf, err := GenTlsConfig(CertFile, KeyFile)
 ```