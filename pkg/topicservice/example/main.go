package main

import (
	"context"
	"os"

	topicservice "github.com/jursonmo/practise_new/pkg/topicservice"
	"github.com/zeromicro/go-zero/core/discov"
	"github.com/zeromicro/go-zero/core/logx"
)

// 此文件用于测试etcd 服务的leader选举， topics 发现和分配，以及把分配的结果同步到etcd 服务上，让其他服务可以知道分配的结果。
// ./main 1 false
// ./main 2 true //看2号服务能否抢占成为leader
func init() {
	logx.DisableStat()
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
	}
	service, err := topicservice.NewService(sc)
	if err != nil {
		logx.Error(err)
		return
	}
	topics := []string{"topic1", "topic2", "topic3", "topic4", "topic5", "topic6", "topic7", "topic8", "topic9", "topic10"}
	service.SetTopics(topics)

	err = service.Start(context.Background())
	if err != nil {
		panic(err)
	}
	logx.Infof("service started, id:%s, isLeader:%v", id, isLeader)
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
