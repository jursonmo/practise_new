package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aceld/zinx/ziface"
	"github.com/aceld/zinx/zlog"
	"github.com/jursonmo/practise_new/cmd/zinx/clientv2"
)

func pingLoop(conn ziface.IConnection) {
	for {
		data := strings.Repeat("a", 10)
		//err := conn.SendMsg(1, []byte("Ping...Ping...Ping...[FromClient]"))
		err := conn.SendMsg(1, []byte(data))
		if err != nil {
			zlog.Error(err)
			break
		}

		time.Sleep(2 * time.Second)
		err = conn.SendMsg(1, []byte(data))
		if err != nil {
			zlog.Error(err)
			break
		}
	}
	zlog.Error("pingLoop exit")
}

// Executed when a connection is created
func onClientStart(conn ziface.IConnection) {
	fmt.Println("onClientStart is Called ... ")
	go pingLoop(conn)
}

func onClientStop(conn ziface.IConnection) {
	fmt.Println("onClientStop is Called ... ")
	//do something, clean up
}

func main() {
	client := clientv2.NewClient("127.0.0.1", 8999)

	client.SetOnConnStart(onClientStart)
	client.SetOnConnStop(onClientStop)
	client.StartWithContext(context.Background())
	//err := client.Connect(context.Background())
	//err := client.ConnectWithTimeout(5 * time.Second)
	// if err != nil {
	// 	zlog.Error("connect failed, err:", err)
	// 	return
	// }

	zlog.Info("connect success")

	select {}
}
