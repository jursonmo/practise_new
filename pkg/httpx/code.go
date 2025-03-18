package httpx

import (
	"context"
	"fmt"
	"net/http"
)

const (
	CodeSuccess = 0  // 成功
	CodeError   = -1 // 未定义的错误码，用-1表示
	// 自定义的通用错误码 1000起步, 避免跟http状态码冲突产生误解
	CodeTimeout            = 1000
	CodeAuthError          = 1001
	CodeNoPermission       = 1002
	CodeNotFound           = 1003
	CodeMethodNotAllowed   = 1004
	CodeTooManyRequests    = 1005
	CodeServiceUnavailable = 1006
	CodeInternalError      = 1007
	CodeParamError         = 1008
)

var (
	ErrTimeout  = NewCodeResponse(CodeTimeout, "timeout", nil)
	ErrAuth     = NewCodeResponse(CodeAuthError, "auth error", nil)
	ErrNotFound = NewCodeResponse(CodeNotFound, "not found", nil)
)

type CodeResponser interface {
	GetCode() int
	SetCode(code int)
	GetMsg() string
	SetMsg(msg string)
	GetData() interface{}
}

type CodeResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

func NewCodeResponse(code int, msg string, data interface{}) *CodeResponse {
	return &CodeResponse{
		Code: code,
		Msg:  msg,
		Data: data,
	}
}

func (r *CodeResponse) GetCode() int {
	return r.Code
}

func (r *CodeResponse) SetCode(code int) {
	r.Code = code
}
func (r *CodeResponse) GetMsg() string {
	return r.Msg
}
func (r *CodeResponse) SetMsg(msg string) {
	r.Msg = msg
}
func (r *CodeResponse) GetData() interface{} {
	return r.Data
}
func (br *CodeResponse) Err() error {
	if br.Code != 0 {
		return fmt.Errorf("code:%d, msg:%s", br.Code, br.Msg)
	}
	return nil
}

// 如果服务器是用codeResponse的方式回应, 这里的客户端就要用codeResponse来接受回应, 请求失败或者code != 0, 都认为出错并返回err
// resp 必须是指针类型; 如果返回值err是nil, 则resp即为需要的数据，如果返回值err不为nil, 不要使用resp
func GetXXX(ctx context.Context, api string, request, resp interface{}, opts ...RequestOpt) error {
	codeResp := NewCodeResponse(0, "", resp)
	err := Request(ctx, http.MethodGet, api, request, codeResp, opts...)
	if err != nil {
		return err
	}
	return codeResp.Err() //如果code不为0, 则返回err
}

// 业务层用这种方式请求，如果err 为nil, 那么业务层还需要判断code和msg, 如果code不为0, 则需要处理错误，否则可以使用resp
func GetYYY(ctx context.Context, api string, request, resp interface{}, opts ...RequestOpt) (*CodeResponse, error) {
	codeResp := NewCodeResponse(0, "", resp)
	err := Request(ctx, http.MethodGet, api, request, codeResp, opts...)
	if err != nil {
		return nil, err
	}
	return codeResp, nil
}

func GetExample(ctx context.Context) error {
	api := "127.0.0.1:8080/example"
	resp := &struct{ Name string }{}
	r, err := GetYYY(ctx, api, nil, resp)
	if err != nil {
		return err
	}
	if r.Code != 0 {
		return fmt.Errorf("code:%d, msg:%s", r.Code, r.Msg)
	}
	//业务处理, do something with resp
	_ = resp
	return nil
}
