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
		"/Users/will/Desktop/learnspace/golang/mcp_learn/time-mcpserver/calculate/calculate",
		//"/Users/will/Desktop/learnspace/golang/mcp_learn/time-mcpserver/time/time_mcpserver",
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

	initResult, err := c.Initialize(ctx, initRequest)
	if err != nil {
		log.Fatalf("Failed to initialize: %v", err)
	}
	fmt.Printf(
		"Initialized ProtocolVersion:%v, server: %s %s\n\n",
		initResult.ProtocolVersion,
		initResult.ServerInfo.Name,
		initResult.ServerInfo.Version,
	)
	//打印: Initialized ProtocolVersion:2024-11-05, server: Calculator Demo 1.0.0
	//意思是初始化时，返回的initResult 的mcp server的名称和版本，如果客户端不支持指定mcp server名称和版本.

	// List Tools
	fmt.Println("Listing available tools...")
	toolsRequest := mcp.ListToolsRequest{}
	tools, err := c.ListTools(ctx, toolsRequest)
	if err != nil {
		log.Fatalf("Failed to list tools: %v", err)
	}
	for _, tool := range tools.Tools {
		fmt.Printf("- %s: %s\n", tool.Name, tool.Description)
	}
	fmt.Println()

	listDirRequest := mcp.CallToolRequest{
		Request: mcp.Request{
			Method: "tools/call",
		},
	}

	listDirRequest.Params.Name = "calculate"
	listDirRequest.Params.Arguments = map[string]any{
		"operation": "add",
		"x":         1.0,
		"y":         2.0,
	}

	result, err := c.CallTool(ctx, listDirRequest)
	if err != nil {
		log.Fatalf("Failed to list allowed directories: %v", err)
	}
	fmt.Printf("result:%#v\n", result)
	//result:&mcp.CallToolResult{Result:mcp.Result{Meta:map[string]interface {}(nil)}, Content:[]mcp.Content{mcp.TextContent{Annotated:mcp.Annotated{Annotations:(*struct { Audience []mcp.Role "json:\"audience,omitempty\""; Priority float64 "json:\"priority,omitempty\"" })(nil)}, Type:"text", Text:"3.00"}}, IsError:false}
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
