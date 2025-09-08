package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	hooks := &server.Hooks{}

	hooks.AddBeforeAny(func(id any, method mcp.MCPMethod, message any) {
		fmt.Printf("beforeAny: %s, %v, %v\n", method, id, message)
	})
	hooks.AddOnSuccess(func(id any, method mcp.MCPMethod, message any, result any) {
		fmt.Printf("onSuccess: %s, %v, %v, %v\n", method, id, message, result)
	})
	hooks.AddOnError(func(id any, method mcp.MCPMethod, message any, err error) {
		fmt.Printf("onError: %s, %v, %v, %v\n", method, id, message, err)
	})
	hooks.AddBeforeInitialize(func(id any, message *mcp.InitializeRequest) {
		fmt.Printf("beforeInitialize: %v, %v\n", id, message)
	})
	hooks.AddAfterInitialize(func(id any, message *mcp.InitializeRequest, result *mcp.InitializeResult) {
		fmt.Printf("afterInitialize: %v, %v, %v\n", id, message, result)
	})
	hooks.AddAfterCallTool(func(id any, message *mcp.CallToolRequest, result *mcp.CallToolResult) {
		fmt.Printf("afterCallTool: %v, %v, %v\n", id, message, result)
	})
	hooks.AddBeforeCallTool(func(id any, message *mcp.CallToolRequest) {
		fmt.Printf("beforeCallTool: %v, %v\n", id, message)
	})

	// Create MCP server
	s := server.NewMCPServer(
		"Demo ?",
		"1.0.0",
		server.WithHooks(hooks),
		server.WithLogging(),
	)
	// Add tool
	tool := mcp.NewTool("current time",
		mcp.WithDescription("Get current time with timezone, Asia/Shanghai is default"),
		mcp.WithString("timezone",
			mcp.Required(),
			mcp.Description("current time timezone"),
		),
	)
	// Add tool handler
	s.AddTool(tool, currentTimeHandler)
	// Start the stdio server
	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}

// 这个才是真正处理业务逻辑的函数，可以调用http 请求或者其他请求完成具体任务。也就是在项目中，这里是真正完成业务逻辑的地方。
func currentTimeHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	timezone, ok := request.Params.Arguments["timezone"].(string)
	if !ok {
		return nil, errors.New("timezone must be a string")
	}

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, errors.New("invalid timezone")
	}
	return mcp.NewToolResultText(fmt.Sprintf(`current time is %s`, time.Now().In(loc))), nil
}

/*
go build -o time_mcpserver main.go

自己填写传入参数，运行测试， 也能获取具体的时间
root#echo '{"jsonrpc": "2.0", "method":"tools/call", "id": 123, "params": {"name": "current time", "arguments": {"timezone": "Asia/Shanghai"}}}' | ./time_mcpserver
beforeAny: tools/call, 123, &{{tools/call {<nil>}} {current time map[timezone:Asia/Shanghai] <nil>}}
beforeCallTool: 123, &{{tools/call {<nil>}} {current time map[timezone:Asia/Shanghai] <nil>}}
onSuccess: tools/call, 123, &{{tools/call {<nil>}} {current time map[timezone:Asia/Shanghai] <nil>}}, &{{map[]} [{{<nil>} text current time is 2025-03-31 15:46:02.874226 +0800 CST}] false}
afterCallTool: 123, &{{tools/call {<nil>}} {current time map[timezone:Asia/Shanghai] <nil>}}, &{{map[]} [{{<nil>} text current time is 2025-03-31 15:46:02.874226 +0800 CST}] false}
{"jsonrpc":"2.0","id":123,"result":{"content":[{"type":"text","text":"current time is 2025-03-31 15:46:02.874226 +0800 CST"}]}}

*/
