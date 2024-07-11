package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/skip2/go-qrcode"
)

var secret *otp.Key

// 生成二维码
func generateQRCode(w http.ResponseWriter, r *http.Request) {
	var err error
	secret, err = totp.Generate(totp.GenerateOpts{
		Issuer:      "ExampleApp",
		AccountName: "example@user.com",
	})
	if err != nil {
		http.Error(w, "Error generating OTP secret", http.StatusInternalServerError)
		return
	}

	qrCodeURL := secret.URL()
	fmt.Println("qrCodeURL:", qrCodeURL)
	png, err := qrcode.Encode(qrCodeURL, qrcode.Medium, 256)
	if err != nil {
		http.Error(w, "Error generating QR code", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Write(png)
}

// 验证OTP码
func verifyOTP(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	var requestData struct {
		OTP string `json:"otp"`
	}

	err = json.Unmarshal(body, &requestData)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	valid := totp.Validate(requestData.OTP, secret.Secret())
	if valid {
		fmt.Fprintln(w, "OTP is valid. Login successful.")
	} else {
		http.Error(w, "Invalid OTP", http.StatusUnauthorized)
	}
}

func main() {
	http.HandleFunc("/generate", generateQRCode)
	http.HandleFunc("/verify", verifyOTP)

	fmt.Println("Server started at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

/*
客户端每次请求二维码，都会生成一个新的二维码，qrCodeURL 都是不一样的
qrCodeURL打印类似如下：
qrCodeURL: otpauth://totp/ExampleApp:example@user.com?algorithm=SHA1&digits=6&issuer=ExampleApp&period=30&secret=E3CZMVFTRSLDFCJMQ66P7X2VPXIBTXYS
全局变量var secret *otp.Key 只有一个，也就是每次生成新的二维码，secret 就代表最后那次二维码
验证是otp否OK的时候, 是用最后那次二维码对应是secret来验证的
所以运行客户端client.go时, 需要按提示填入当前server生成的二维码里的secret:E3CZMVFTRSLDFCJMQ66P7X2VPXIBTXYS
然后得到六位otp, 然后发给服务器端来验证。

客户端运行的打印如下：
go run client.go
QR code saved as qrcode.png. Scan it with your OTP app.
Please enter the OTP secret:
E3CZMVFTRSLDFCJMQ66P7X2VPXIBTXYS
ok, otp secret: E3CZMVFTRSLDFCJMQ66P7X2VPXIBTXYS
Generated OTP: 116785
OTP is valid. Login successful.

-------------------------测试2------------------
一般客户端是直接输入六位数字的otp code 来验证是否ok。 client2.go 就是可以直接输入otp code的程序。
那么怎么样才能不输入二维码secret的情况下，得到otp code？
那就是手机身份验证器比如Authenticator扫描二维码，添加ExampleApp这个账户,然后在Authenticator 就可以实时获得otp code
然后在client2.go 运行测试下，直接输入Authenticator上生成的otp code：
go run client2.go
Please enter the OTP code:
896599
ok, otp code: 896599
Generated OTP: 896599
OTP is valid. Login successful.

验证是成功的，
可以认为Authenticator 就是拿到二维码里的相关信息，比如algorithm=SHA1&period=30&secret=E3CZMVFTRSLDFCJMQ66P7X2VPXIBTXYS
然后定期调用otp, err := totp.GenerateCode(secret, time.Now())来生成otp code

服务器验证otp code是否合法的底层逻辑是什么，服务器和Authenticator拥有相同的SHA1，secret，period
从client.go 的totp.GenerateCode(secret, time.Now())生成code的源码里，可以猜测出：
生成code是有间隔，比如每隔30秒生成一次。客户端通过当前时间戳除以30得到一个数值，这个数值跟secret拼接后，用SHA1生成六位code
服务器在时间间隔内，也是用同样的方式来得到一个code, 如果这两个code 一样，就表示合法。

---------------------TODO: 改进，服务器为每个不同用户生成不同的二维码(不同otp.Key)，同时可以验证不同用户的otp code是否合法 ---------------
client 申请新的二维码时，带上自己用户信息，服务器返回二维码，并记录用户和otp.Key 的对应关系
client 发起code 验证时，也需要带上自己用户信息，服务器根据不同的用户，用不同的otp.Key来验证是否合法。

这样就像跳板机那样验证每个用户登录otp code了。
*/
