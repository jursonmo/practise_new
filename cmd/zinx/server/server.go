package main

import (
	"fmt"
	"time"

	"github.com/aceld/zinx/zconf"
	"github.com/aceld/zinx/ziface"
	"github.com/aceld/zinx/znet"
)

// PingRouter MsgId=1
type PingRouter struct {
	znet.BaseRouter
}

// Ping Handle MsgId=1
func (r *PingRouter) Handle(request ziface.IRequest) {
	//read client data
	fmt.Println("recv from client : msgId=", request.GetMsgID(), "len=", len(request.GetData()), "data=", string(request.GetData()))
	fmt.Println("--------------")
	time.Sleep(time.Minute)
	fmt.Println("server handle over")
}

func main() {
	//1 Create a server service
	zconf.GlobalObject.MaxPacketSize = 100 //默认的解包是tlv, 这个设置就不起作用了，实验client发过来120的数据，也能解包成功。

	//WorkerPoolSize = 0的意思是不开启工作池，默认是开启的。不开启后,每次接受都数据，都起goroutine去处理，PingRouter.Handle()将会并发执行
	//zconf.GlobalObject.WorkerPoolSize = 0
	s := znet.NewServer()

	//2 configure routing
	s.AddRouter(1, &PingRouter{})

	//3 start service
	s.Serve()
}
