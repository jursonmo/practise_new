package main

//服务端和客户端之间实现一个简单token 校验来验证合法性，帮我实现一个generateToken()和Verify(),
//token只有4个字节大小，为了避免轻易被识破攻击，生成的 token 是动态，把当前utc的秒用token第二第三这两个字节存储，
//把这个存储的秒数跟预设的secret  一起SHA 摘要，取部分写到token的第一个和第四个字节里。
import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"time"
)

func main() {
	secret := "my_secret_key"

	// 生成 Token
	token := generateToken(secret)
	fmt.Printf("Generated Token: %x\n", token)

	// 立即验证 (应成功)
	valid, err := verifyToken(token, secret, 1)
	if err != nil {
		log.Println(err)
	}
	if valid {
		fmt.Println("Token valid (expected)")
	} else {
		fmt.Println("Token invalid (unexpected)")
	}

	// 伪造测试 (篡改hash部分)
	fakeToken := []byte{token[0], token[1], token[2], token[3] + 1}
	valid, err = verifyToken(fakeToken, secret, 9)
	if err != nil {
		fmt.Printf("Fake token test result: valid: %t, err: %v\n", valid, err)
	}
	if !valid {
		fmt.Println("Fake token rejected (expected)")
	}

	// 测试过期 Token
	fmt.Println("\nTesting expiration...")
	expiredToken := generateToken(secret)
	time.Sleep(2 * time.Second) // 等待2秒

	// 使用1秒的短窗口验证 (应过期)
	valid, err = verifyToken(expiredToken, secret, 1)
	if err != nil {
		log.Println(err)
	}
	if !valid {
		fmt.Println("Expired token rejected (expected)")
	}
}

// generateToken 生成动态 Token (4字节)
// 参数: secret - 预设的密钥字符串
// 返回: 4字节的 token
func generateToken(secret string) []byte {
	// 获取当前 UTC 秒数
	now := time.Now().UTC().Unix()
	t := uint16(now)

	// 准备用于计算摘要的数据: secret + 完整时间戳
	data := append([]byte(secret), make([]byte, 2)...)
	binary.BigEndian.PutUint16(data[len(secret):], t)

	// 计算 SHA-256 摘要
	hash := sha256.Sum256(data)

	// 构建 4 字节 Token:
	//   [0] = 摘要首字节
	//   [1] = 时间高位字节
	//   [2] = 时间低位字节
	//   [3] = 摘要尾字节
	token := make([]byte, 4)
	token[0] = hash[0]           // 摘要头
	token[1] = byte(t >> 8)      // 时间高位
	token[2] = byte(t)           // 时间低位
	token[3] = hash[len(hash)-1] // 摘要尾

	return token
}

// verifyToken 验证 Token 有效性
// 参数:
//
//	token  - 待验证的 4 字节 token
//	secret - 预设密钥
//	window - 时间窗口(秒)，默认 10 秒
//
// 返回: 是否验证通过
func verifyToken(token []byte, secret string, window int) (bool, error) {
	if len(token) != 4 {
		return false, errors.New("invalid token length")
	}

	if window <= 0 {
		window = 10 // 默认 10秒有效期
	}

	// 从 Token 提取时间部分 (低16位)
	tokenTime := uint16(token[1])<<8 | uint16(token[2])

	// 获取当前 UTC 秒数
	now := time.Now().UTC().Unix()
	currentTime := uint16(now)

	if tokenTime-currentTime > uint16(window) {
		return false, fmt.Errorf("Token timestamp too far from now, tokenTime: %d, currentTime: %d, window: %d\n", tokenTime, currentTime, window) // 时间戳相差太远
	}

	// 准备用于计算摘要的数据: secret + 部分时间戳
	data := append([]byte(secret), make([]byte, 2)...)
	binary.BigEndian.PutUint16(data[len(secret):], currentTime)

	// 计算 SHA-256 摘要
	hash := sha256.Sum256(data)

	// 验证摘要头尾是否匹配
	if hash[0] == token[0] && hash[len(hash)-1] == token[3] {
		return true, nil // 验证通过
	}

	return false, nil // 所有候选值均验证失败
}
