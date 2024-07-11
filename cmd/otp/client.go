package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	//"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/pquerna/otp/totp"
)

// 从服务器获取二维码
func getQRCode() {
	resp, err := http.Get("http://localhost:8080/generate")
	if err != nil {
		log.Fatalf("Failed to get QR code: %v", err)
	}
	defer resp.Body.Close()

	//qrCode, err := ioutil.ReadAll(resp.Body)
	qrCode, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read QR code: %v", err)
	}

	err = os.WriteFile("qrcode.png", qrCode, 0644)
	if err != nil {
		log.Fatalf("Failed to save QR code: %v", err)
	}

	fmt.Println("QR code saved as qrcode.png. Scan it with your OTP app.")
}

// 向服务器发送OTP码进行验证
func verifyOTP(secret, otp string) {
	client := &http.Client{}
	requestBody, err := json.Marshal(map[string]string{
		"otp": otp,
	})
	if err != nil {
		log.Fatalf("Failed to create request body: %v", err)
	}

	req, err := http.NewRequest("POST", "http://localhost:8080/verify", io.NopCloser(bytes.NewReader(requestBody)))
	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	//responseBody, err := ioutil.ReadAll(resp.Body)
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read response: %v", err)
	}

	if resp.StatusCode == http.StatusOK {
		fmt.Println(string(responseBody))
	} else {
		fmt.Printf("Failed to verify OTP: %s\n", string(responseBody))
	}
}

func main() {
	// 生成二维码
	getQRCode()

	// 用户手动输入扫描二维码后的Secret和生成的OTP码
	fmt.Println("Please enter the OTP secret:")
	var secret string
	fmt.Scanln(&secret)
	fmt.Println("ok, otp secret:", secret)
	otp, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		log.Fatalf("Failed to generate OTP: %v", err)
	}

	fmt.Printf("Generated OTP: %s\n", otp)

	// 验证OTP码
	verifyOTP(secret, otp)
}
