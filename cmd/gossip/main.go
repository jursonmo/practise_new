package main

import (
	"os"
	"strconv"
	"time"

	"github.com/hashicorp/memberlist"
	"github.com/zeromicro/go-zero/core/logx"
)

func strToInt(s string) int {
	//string to int
	i, err := strconv.Atoi(s)
	if err != nil {
		logx.Error(err)
		return 0
	}
	return i
}
func main() {
	if len(os.Args) < 4 {
		logx.Errorf("Usage: %s <name> <addr> <port> <peerPort>", os.Args[0])
		return
	}
	name := os.Args[1]
	addr := os.Args[2]
	port := strToInt(os.Args[3])
	peerPort := "8081"
	if len(os.Args) > 4 {
		peerPort = os.Args[4]
	}
	delegate := &GossipDelegate{Name: name}

	// 创建 Gossip 配置
	config := memberlist.DefaultLANConfig()
	config.Name = name
	config.BindAddr = addr
	config.BindPort = port
	config.Delegate = delegate
	//config.Events = &eventDelegate{}

	m, err := memberlist.Create(config)
	if err != nil {
		panic(err)
	}

	br := &memberlist.TransmitLimitedQueue{
		NumNodes: func() int {
			return m.NumMembers()
		},
		RetransmitMult: 3,
	}

	delegate.cluster = m
	delegate.broadcasts = br

	if name == "node2" {
		n, err := delegate.cluster.Join([]string{"192.168.1.4:" + peerPort})
		if err != nil {
			panic(err)
		}
		logx.Infof("join success, n:%d", n)
		time.Sleep(time.Second)
		delegate.broadcasts.QueueBroadcast(&broadcast{msgId: 1, msg: []byte("hello"), notify: nil})
	}

	logx.Info("started")
	select {}
}

type GossipDelegate struct {
	Name       string
	cluster    *memberlist.Memberlist
	broadcasts *memberlist.TransmitLimitedQueue
}

// NodeMeta 返回节点元数据（此处为空实现）
func (d *GossipDelegate) NodeMeta(limit int) []byte {
	return nil
}

// NotifyMsg 处理接收到的 Gossip 消息
func (d *GossipDelegate) NotifyMsg(b []byte) {
	logx.Infof("NotifyMsg, len(b):%d", len(b))
}

// GetBroadcasts 返回要广播的消息（此处为空实现）
func (d *GossipDelegate) GetBroadcasts(overhead, limit int) [][]byte {
	return d.broadcasts.GetBroadcasts(overhead, limit)
}

// LocalState 返回节点的本地状态, 用于发送给其他节点
func (d *GossipDelegate) LocalState(join bool) []byte {
	logx.Infof("LocalState, join:%v", join)
	return []byte(d.Name)
}

// MergeRemoteState 合并远程节点的状态
func (d *GossipDelegate) MergeRemoteState(buf []byte, join bool) {
	logx.Infof("MergeRemoteState, len(buf):%d, join:%v", len(buf), join)
}

type broadcast struct {
	msgId  int64
	msg    []byte
	notify chan<- struct{}
}

func (b *broadcast) Invalidates(other memberlist.Broadcast) bool {
	return false
}

func (b *broadcast) Message() []byte {
	logx.Infof("broadcast msgId:%d", b.msgId)
	return b.msg
}

func (b *broadcast) Finished() {
	logx.Infof("broadcast msgId:%d finished", b.msgId)
	if b.notify != nil {
		close(b.notify)
	}
}
