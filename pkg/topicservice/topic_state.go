package topicservice

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/hashicorp/memberlist"
	"github.com/zeromicro/go-zero/core/logx"
)

const (
	ADD = 1
	DEL = 2
)

type TopicInfo struct {
	Op      int //1:add, 2:del
	Topic   string
	Service string //node
}

// TopicState 表示节点当前的订阅状态
type TopicState struct {
	mu           sync.Mutex
	LocalTopics  map[string]TopicInfo   // key是{topic}
	GlobalTopics map[string]*ServiceSet //map[topic]

	cluster    *memberlist.Memberlist
	broadcasts *memberlist.TransmitLimitedQueue
}

type ServiceSet struct {
	sync.RWMutex
	Services map[string]struct{} //key service
}

func NewServiceSet() *ServiceSet {
	return &ServiceSet{
		Services: make(map[string]struct{}),
	}
}

// 为了打印GlobalTopics时，不会打印成指针地址，所以这里实现String()方法
func (s *ServiceSet) String() string {
	//return fmt.Sprintf("%v", *s)
	return fmt.Sprintf("%v", s.Services)
}
func (s *ServiceSet) Add(service string) {
	s.Lock()
	defer s.Unlock()
	s.Services[service] = struct{}{}
}
func (s *ServiceSet) Del(service string) {
	s.Lock()
	defer s.Unlock()
	delete(s.Services, service)
}

// 添加或更新 topic 信息
func (s *TopicState) UpdateGlobalTopic(info TopicInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()

	//del
	if info.Op == DEL {
		sm, ok := s.GlobalTopics[info.Topic]
		if ok {
			sm.Del(info.Service)
			if len(sm.Services) == 0 {
				delete(s.GlobalTopics, info.Topic)
			}
		}
		return
	}
	//add
	sm, ok := s.GlobalTopics[info.Topic]
	if !ok {
		sm = NewServiceSet()
		sm.Add(info.Service)
		s.GlobalTopics[info.Topic] = sm
		return
	}
	sm.Add(info.Service)
}

func (s *TopicState) GetLocalTopics() map[string]TopicInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.LocalTopics
}

// 获取所有 topics
func (s *TopicState) GetTopics() map[string]*ServiceSet {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.GlobalTopics
}

func (s *TopicState) CloneTopics() map[string]*ServiceSet {
	s.mu.Lock()
	defer s.mu.Unlock()
	topicsCopy := make(map[string]*ServiceSet)
	for k, v := range s.GlobalTopics {
		tmp := ServiceSet{Services: make(map[string]struct{})}
		for k1, v1 := range v.Services {
			tmp.Services[k1] = v1
		}
		topicsCopy[k] = &tmp
	}
	return topicsCopy
}

// GossipDelegate 用于处理 Gossip 协议的事件
type GossipDelegate struct {
	state *TopicState
}

// NodeMeta 返回节点元数据（此处为空实现）
func (d *GossipDelegate) NodeMeta(limit int) []byte {
	return nil
}

// NotifyMsg 处理接收到的 Gossip 消息
func (d *GossipDelegate) NotifyMsg(b []byte) {
	var updates []TopicInfo
	//decoder := gob.NewDecoder(bytes.NewReader(b))
	decoder := json.NewDecoder(bytes.NewReader(b))
	if err := decoder.Decode(&updates); err != nil {
		logx.Errorf("NotifyMsg, Failed to decode len:%d message err:%v", len(b), err)
		return
	}

	// 更新本地的订阅信息
	logx.Infof("NotifyMsg, updates:%+v", updates)
	for _, info := range updates {
		d.state.UpdateGlobalTopic(info)
	}
}

// GetBroadcasts 返回要广播的消息（此处为空实现）.
// 莫：必须实现，如果不实现，那么就会出现广播消息没有发送出去的问题。
func (d *GossipDelegate) GetBroadcasts(overhead, limit int) [][]byte {
	return d.state.broadcasts.GetBroadcasts(overhead, limit)
}

// LocalState 返回节点的本地状态, 用于发送给其他节点
func (d *GossipDelegate) LocalState(join bool) []byte {
	d.state.mu.Lock()
	defer d.state.mu.Unlock()
	var buf bytes.Buffer
	//encoder := gob.NewEncoder(&buf)
	encoder := json.NewEncoder(&buf)
	if err := encoder.Encode(d.state.GlobalTopics); err != nil {
		log.Println("Failed to encode local state:", err)
		return nil
	}
	//join true 表示第一次同步本地的数据给对方, 数据的内容往往是本地的初始状态。
	logx.Infof("LocalState, join:%v, state:%+v", join, d.state.GlobalTopics)
	return buf.Bytes()
}

// MergeRemoteState 合并远程节点的状态
func (d *GossipDelegate) MergeRemoteState(buf []byte, join bool) {
	var remoteTopics map[string]*ServiceSet
	//decoder := gob.NewDecoder(bytes.NewReader(buf))
	decoder := json.NewDecoder(bytes.NewReader(buf))
	if err := decoder.Decode(&remoteTopics); err != nil {
		log.Println("Failed to decode remote state:", err)
		return
	}
	//join 为true 表示是新节点加入, 数据的内容是新节点的初始状态, 算是增量数据(内容只是新节点自己的数据)。
	//join 为false 表示数据内容不是新节点第一次发送的数据，数据内容很可能是集群的全量。
	logx.Infof("MergeRemoteState, join:%v remoteTopics:%+v", join, remoteTopics)
	// 更新本地状态
	for topic, info := range remoteTopics {
		//logx.Infof("MergeRemoteState, topic:%s, info:%+v", topic, info)
		for service := range info.Services {
			topicInfo := TopicInfo{
				Op:      ADD,
				Topic:   topic,
				Service: service,
			}
			//logx.Infof("MergeRemoteState, topicInfo:%+v", topicInfo)
			d.state.UpdateGlobalTopic(topicInfo)
		}
	}
}

type eventDelegate struct{}

func (ed *eventDelegate) NotifyJoin(node *memberlist.Node) {
	logx.Info("A gossip node has joined: " + node.String())
}

func (ed *eventDelegate) NotifyLeave(node *memberlist.Node) {
	logx.Info("A gossip node has left: " + node.String())
}

func (ed *eventDelegate) NotifyUpdate(node *memberlist.Node) {
	logx.Info("A gossip node was updated: " + node.String())
}

func ParseAddress(address string) (string, int, error) {
	// 去掉协议前缀
	prefix := "://"
	if !strings.HasPrefix(address, prefix) {
		return "", 0, fmt.Errorf("invalid address format, expected prefix %q", prefix)
	}
	address = strings.TrimPrefix(address, prefix)

	// 使用 net.SplitHostPort 分离 IP 和端口
	host, portStr, err := net.SplitHostPort(address)
	if err != nil {
		return "", 0, fmt.Errorf("failed to parse address: %w", err)
	}

	// 将端口转换为整型
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return "", 0, fmt.Errorf("invalid port: %w", err)
	}

	return host, port, nil
}

func NewTopicState(ID string, addr string, port int) (*TopicState, error) {
	s := &TopicState{
		LocalTopics:  make(map[string]TopicInfo),
		GlobalTopics: make(map[string]*ServiceSet),
	}

	// 创建 Gossip 配置
	config := memberlist.DefaultLANConfig()
	config.Name = ID
	config.BindAddr = addr
	config.BindPort = port
	config.Delegate = &GossipDelegate{state: s}
	config.Events = &eventDelegate{}

	m, err := memberlist.Create(config)
	if err != nil {
		return nil, err
	}

	br := &memberlist.TransmitLimitedQueue{
		NumNodes: func() int {
			return m.NumMembers()
		},
		RetransmitMult: 3,
	}

	s.cluster = m
	s.broadcasts = br
	return s, nil
}

// x.x.x.x:8080
func (s *TopicState) Join(members []string) error {
	// 加入集群
	_, err := s.cluster.Join(members)
	if err != nil {
		return err
	}

	return nil
}
func (s *TopicState) Members() []string {
	nodes := s.cluster.Members()
	memberNames := make([]string, 0, len(nodes))
	for _, node := range nodes {
		memberNames = append(memberNames, fmt.Sprintf("%s(%s:%d)", node.Name, node.Addr, node.Port))
	}
	return memberNames
}

func (s *TopicState) AddTopic(topics []string) {
	s.UpdateTopic(ADD, topics, true)
}
func (s *TopicState) DelTopic(topics []string) {
	s.UpdateTopic(DEL, topics, true)
}

var broadcastNum int64

func (s *TopicState) UpdateTopic(op int, topics []string, needBroadcast bool) error {
	if len(topics) == 0 {
		return errors.New("topics is empty")
	}
	topicsInfo := make([]TopicInfo, 0, len(topics))
	for _, topic := range topics {
		topicInfo := TopicInfo{
			Op:      op,
			Topic:   topic,
			Service: s.cluster.LocalNode().Name,
		}
		s.UpdateGlobalTopic(topicInfo)
		topicsInfo = append(topicsInfo, topicInfo)
	}

	if !needBroadcast {
		return nil
	}

	// 广播更新
	jsonBytes, err := json.Marshal(topicsInfo)
	if err != nil {
		logx.Error("Failed to encode message:", err)
		return err
	}

	atomic.AddInt64(&broadcastNum, 1)
	logx.Infof("gossip Broadcast %d message:%s, members:%v, len(members):%d",
		atomic.LoadInt64(&broadcastNum),
		string(jsonBytes),
		s.cluster.Members(), s.cluster.NumMembers())

	s.broadcasts.QueueBroadcast(&broadcast{
		msgId:  atomic.LoadInt64(&broadcastNum),
		msg:    jsonBytes,
		notify: nil,
	})
	return nil
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
	//logx.Infof("broadcast msgId:%d", b.msgId)
	return b.msg
}

func (b *broadcast) Finished() {
	logx.Infof("broadcast msgId:%d finished", b.msgId)
	if b.notify != nil {
		close(b.notify)
	}
}
