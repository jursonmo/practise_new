package main

import (
	"context"
	"os"
	"strconv"
	"time"

	topicservice "github.com/jursonmo/practise_new/pkg/topicservice"
	"github.com/zeromicro/go-zero/core/discov"
	"github.com/zeromicro/go-zero/core/logx"
)

// 此文件用于测试 通过gossip 能否同步topic信息。
// ./main 1
// ./main 2
func init() {
	logx.DisableStat()
}
func strToInt(s string) int {
	//string to int
	i, err := strconv.Atoi(s)
	if err != nil {
		logx.Error(err)
		return 0
	}
	return i
}
func StartService(id string, isLeader bool) {
	sc := &topicservice.ServiceConfig{
		Name:      "topic_service",
		Id:        id,
		Endpoints: []string{"tcp://127.0.0.1:8080"},

		Etcd: discov.EtcdConf{
			Hosts: []string{"127.0.0.1:2379"},
			Key:   "services",
		},
		IsLeader: isLeader,
		Gossip:   topicservice.GossipConf{Enabled: true, Addr: "127.0.0.1", Port: 8080 + strToInt(id)}, //service1 和 service2 分别监听不同的端口8081 和 8082
	}

	service, err := topicservice.NewService(sc)
	if err != nil {
		logx.Error(err)
		return
	}

	var topics []string
	if id == "1" {
		topics = []string{"topic1", "topic2", "topic3", "topic4"}
	} else {
		topics = []string{"topic5", "topic6", "topic7", "topic8"}
	}

	service.SetTopics(topics) //给service1 和 service2 分别初始化一部分topics

	err = service.Start(context.Background())
	if err != nil {
		panic(err)
	}
	logx.Infof("service started, id:%s, isLeader:%v", id, isLeader)
	time.Sleep(time.Second * 10)
	if id == "1" {
		service.AddTopicState("topic10") //测试在service1上添加一个topic
	} else {
		service.DelTopicState("topic5") //测试在service2上删除一个已经存在的topic
	}
	select {}
}

func main() {
	if len(os.Args) < 2 {
		logx.Errorf("Usage: %s <id> <isLeader>", os.Args[0])
		return
	}

	isLeader := false
	if len(os.Args) > 2 {
		isLeader = true
	}
	StartService(os.Args[1], isLeader)
}
