https://modelcontextprotocol.io/docs/tools/inspector
npx -y @modelcontextprotocol/inspector ./calculate

可以验证mcp server 实现有没有成功, 可以list tools  查看哪些工具, call tools 输入参数，可以查看调用传的数据是怎么样的，返回结果的数据是什么样的, 

比如 tools/call:
reqquest:
```json
{
  "method": "tools/call",
  "params": {
    "name": "calculate",
    "arguments": {
      "operation": "add",
      "x": 1,
      "y": 2
    },
    "_meta": {
      "progressToken": 0
    }
  }
}
```
response:
```json
{
  "content": [
    {
      "type": "text",
      "text": "3.00"
    }
  ]
}
```