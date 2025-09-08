
#### mcp server:
在实际应用中，把mcp server 编译成可执行程序，放在mcp client 本地，方便mcp client 直接运行这个可执行程序，并传入参数
mcp server 可执行程序 根据具体参数回调函数，执行相应的业务逻辑，这个回调函数可以做很多事情，比如发起http 请求远程服务器。
 一个mcp server 可执行程序里可以有多个工具，根据传入参数调用相应的工具的回调函数。