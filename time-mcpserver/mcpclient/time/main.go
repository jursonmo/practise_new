package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

func main() {
	// c, err := client.NewStdioMCPClient(
	// 	"npx",
	// 	[]string{}, // Empty ENV
	// 	"-y",
	// 	"@modelcontextprotocol/server-filesystem",
	// 	"/tmp",
	// )
	c, err := client.NewStdioMCPClient(
		// "/Users/will/Desktop/learnspace/golang/mcp_learn/time-mcpserver/calculate/calculate",
		"/Users/will/Desktop/learnspace/golang/mcp_learn/time-mcpserver/time/time_mcpserver",
		[]string{}, // Empty ENV
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer c.Close()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Initialize the client
	fmt.Println("Initializing client...")
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "example-client",
		Version: "1.0.0",
	}

	// 其实就是向mcp server发送一个的请求,请求的data 类似于 {"jsonrpc": "2.0", "method":"initialize", "id": xxx }
	// mcp-go 的服务器默认实现 "method":"initialize" 和 "method":"tools/list" 等等方法。
	initResult, err := c.Initialize(ctx, initRequest)
	if err != nil {
		log.Fatalf("Failed to initialize: %v", err)
	}
	fmt.Printf(
		"Initialized with server: %s %s\n\n",
		initResult.ServerInfo.Name,
		initResult.ServerInfo.Version,
	)

	// List Tools
	fmt.Println("Listing available tools...")
	toolsRequest := mcp.ListToolsRequest{}
	// 其实就是向mcp server发送一个的请求,请求的data 类似于:
	//{"jsonrpc": "2.0", "method":"tools/list", "id": xxx, "params":{"cursor": xx}}
	// 用mcp-go 库实现的mcp server默认实现 "method":"tools/list" 方法。并且这个方法返回的是s.AddTool()注册的所有的工具。
	tools, err := c.ListTools(ctx, toolsRequest)
	if err != nil {
		log.Fatalf("Failed to list tools: %v", err)
	}
	for _, tool := range tools.Tools {
		//打印:- current time: Get current time with timezone, Asia/Shanghai is default
		fmt.Printf("- %s: %s\n", tool.Name, tool.Description)
	}
	fmt.Println()

	request := mcp.CallToolRequest{
		Request: mcp.Request{
			Method: "tools/call",
		},
	}

	request.Params.Name = "current time"
	request.Params.Arguments = map[string]any{
		"timezone": "Asia/Shanghai",
	}
	// 每个请求的Params 不一样：
	//{"jsonrpc": "2.0", "method":"tools/call", "id": xxx, "params":{"name": "xxx tool", "arguments":{"timezone": "xxx"}} }
	result, err := c.CallTool(ctx, request)
	if err != nil {
		log.Fatalf("Failed to list allowed directories: %v", err)
	}
	fmt.Printf("result:%#v\n", result)
	printToolResult(result)
	fmt.Println()
}

// Helper function to print tool results
func printToolResult(result *mcp.CallToolResult) {
	for _, content := range result.Content {
		if textContent, ok := content.(mcp.TextContent); ok {
			fmt.Println(textContent.Text) //打印: 3.00
		} else {
			jsonBytes, _ := json.MarshalIndent(content, "", "  ")
			fmt.Println(string(jsonBytes))
		}
	}
}

/*
go run main.go

Initializing client...
Initialized with server: Demo ? 1.0.0

Listing available tools...
- current time: Get current time with timezone, Asia/Shanghai is default

result:&mcp.CallToolResult{Result:mcp.Result{Meta:map[string]interface {}(nil)}, Content:[]mcp.Content{mcp.TextContent{Annotated:mcp.Annotated{Annotations:(*struct { Audience []mcp.Role "json:\"audience,omitempty\""; Priority float64 "json:\"priority,omitempty\"" })(nil)}, Type:"text", Text:"current time is 2025-04-09 14:10:59.152355 +0800 CST"}}, IsError:false}
current time is 2025-04-09 14:10:59.152355 +0800 CST
*/
