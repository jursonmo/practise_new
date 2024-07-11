
1. 启动Golang服务器, html就连接这个服务器，由这个服务器再ssh去连接远程的机器：go run main.go
2. 打开浏览器并访问 index2.html。
3. 输入 SSH 连接信息并点击“Connect”按钮，连接成功后可以在终端中输入命令并按下回车键或Tab键，命令将发送到服务器并显示输出。
输入远程机器的ip 端口和用户名密码

说明：index.html 输入一个字符，它就发过去了； 用index2.html, 工作正常，遇到回车或者tab才发过去。 