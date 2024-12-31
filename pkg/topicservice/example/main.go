package main

import (
	"context"

	topicservice "github.com/jursonmo/practise_new/pkg/topicservice"
	"github.com/zeromicro/go-zero/core/discov"
	"github.com/zeromicro/go-zero/core/logx"
)

func main() {
	sc1 := &topicservice.ServiceConfig{
		Name:      "topic_service",
		Id:        "1",
		Endpoints: []string{"tcp://127.0.0.1:8080"},

		Etcd: discov.EtcdConf{
			Hosts: []string{"127.0.0.1:2379"},
			Key:   "topic_key",
		},
	}
	service1, err := topicservice.NewService(sc1)
	if err != nil {
		logx.Error(err)
		return
	}
	sc2 := &topicservice.ServiceConfig{
		Name:      "topic_service",
		Id:        "2",
		IsLeader:  true,
		Endpoints: []string{"tcp://127.0.0.1:8080"},
		Etcd: discov.EtcdConf{
			Hosts: []string{"127.0.0.1:2379"},
			Key:   "topic_key",
		},
	}
	service2, err := topicservice.NewService(sc2)
	if err != nil {
		logx.Error(err)
		return
	}
	topics := []string{"topic1", "topic2", "topic3", "topic4", "topic5", "topic6", "topic7", "topic8", "topic9", "topic10"}
	service1.SetTopics(topics)
	service2.SetTopics(topics)

	logx.Info("start service1")
	service1.Start(context.Background())
	logx.Info("start service2")
	service2.Start(context.Background())
	select {}
}
