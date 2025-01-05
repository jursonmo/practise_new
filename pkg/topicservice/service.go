package topicservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/jursonmo/practise_new/pkg/hash"
	"github.com/zeromicro/go-zero/core/discov"
	"github.com/zeromicro/go-zero/core/logx"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// TODO:
// 1. 每个服务都订阅topic信息，并保存下来，用于显示
// 2. 一致性hash, 如果是leader没有变，服务列表变了，计算后，部分topic 有变化，其他服务订阅topics的信息时，会认为所有的都有变化吗

type ServiceInfo = ServiceConfig
type ServiceConfig struct {
	Name      string            `json:"name,optional"`
	Id        string            `json:"id,optional"` //服务的唯一性
	Endpoints []string          `json:",optional"`   //broker service 目前是可以不用设置endpoints
	Weight    int               `json:",optional"`   //权重
	Priority  int               `json:",optional"`   //优先级
	Ns        string            `json:",optional"`
	As        string            `json:",optional"`
	Etcd      discov.EtcdConf   //`json:"-"` //注册到哪里去, 完整的注册路径: /ns/as/key/id
	IsLeader  bool              `json:",optional"` // 配置是否是leader，多个服务可以设置都设置了该配置，Id最小的那个是leader
	Metadata  map[string]string `json:",optional"`
}

const (
	TopicsSep       = ";" //topic之间的分隔符
	TopicServiceSeq = "|" //topic负载的多个service之间的分隔符
)

type Service struct {
	sync.Mutex
	ctx       context.Context
	cancel    context.CancelFunc
	sc        *ServiceConfig
	isLeader  bool //是否是leader, 只有leader才会分配topic和service(broker)的对应关系
	pubClient *discov.Publisher

	balance     *Balance
	serviceList []ServiceInfo
	topics      []string
	etcdClient  *clientv3.Client
	topicPerKey bool
	//topic和service的对应关系
	// topicServiceMap map[string]*ServiceConfig
}

func (s ServiceInfo) String() string {
	return fmt.Sprintf("%s-%s", s.Name, s.Id)
}
func NewService(sc *ServiceConfig) (*Service, error) {
	//检查参数,设置默认值
	if sc.Id == "" {
		return nil, fmt.Errorf("ServiceConfig Id is empty")
	}
	if sc.Etcd.Key == "" {
		return nil, fmt.Errorf("ServiceConfig Etcd.Key is empty")
	}
	if sc.Ns == "" {
		sc.Ns = "ns"
	}
	if sc.As == "" {
		sc.As = "as"
	}

	// 补充key的完整路径，方便后面的订阅
	sc.Etcd.Key = fmt.Sprintf("/%s/%s/%s", sc.Ns, sc.As, sc.Etcd.Key)

	return &Service{
		sc: sc,
		//balance: DefaultBalance, //应该每个服务创建一个Balance
		balance: NewBalance(&myConsistentHash{
			chash: hash.NewConsistentHash(),
			name:  "consistent_hash",
			desc:  "consistent hash balance alg",
		}), //默认使用一致性hash算法
		//topicServiceMap: make(map[string]*ServiceConfig),
	}, nil
}
func (s *Service) SetDefaultBalancer(balance ServiceBalancer) {
	s.balance.UpdateDefaultBalancer(balance)
}

func (s *Service) Start(ctx context.Context) error {
	s.ctx, s.cancel = context.WithCancel(ctx)
	//注册到etcd
	if err := s.Register(); err != nil {
		return err
	}

	//订阅services
	logx.Info("-----start discov service-------")
	if err := s.StartDiscovService(); err != nil {
		return err
	}

	//监听etcd的事件topic和service的对应关系
	logx.Info("-----start discov topics-------")
	if err := s.StartDiscovTopics(); err != nil {
		return err
	}
	return nil
}

func (s *Service) Stop() error {
	if s.pubClient != nil {
		s.pubClient.Stop()
	}
	return nil
}
func (s *Service) SetTopics(topics []string) {
	s.Lock()
	defer s.Unlock()
	s.topics = topics
}
func (s *Service) GetTopics() []string {
	s.Lock()
	defer s.Unlock()
	return s.topics
}

func (s *Service) GetServicesByTopic(topic string) ([]ServiceConfig, error) {
	if s.balance == nil {
		return nil, fmt.Errorf("balance is nil")
	}
	//根据topic获取service
	services := s.balance.GetServiceByTopic(topic)
	return services, nil
}

func (s *Service) GetServiceConfig() *ServiceConfig {
	return s.sc
}

// 1. 服务列表有变化：
//    1.1 如果是leader, 就重新计算 topic-->services，并更新到etcd
//	  1.2 如果是follower, 只需要关闭 s.etcdClient.Close()
// 2. 服务列表没有变化: 即原来是leader的还是leader, 是follower还是follower， 所以啥都不需要做。

func (s *Service) StartDiscovService() error {
	DiscovCustomService(s.sc.Etcd.Hosts, s.sc.Etcd.Key, func(list []ServiceConfig) error {
		logx.Debugf("get Custom %s service:%+v", s.sc.Etcd.Key, list)
		if reflect.DeepEqual(s.serviceList, list) {
			//服务列表没有变化: 即原来是leader的还是leader, 是follower还是follower， 所以啥都不需要做。
			logx.Info("serviceList no change")
			return nil
		}
		leader := findLeader(list)
		if leader == nil {
			return errors.New("leader == nil")
		}
		logx.Debugf("leader:%+v", leader)
		if leader.Id == s.sc.Id {
			s.isLeader = true
		} else {
			s.isLeader = false
		}
		if s.isLeader {
			logx.Infof("I am the leader, name:%s, id:%s", s.sc.Name, s.sc.Id)
			//set services leader to etcd
			s.PublishLeader()
			//更新balance的services
			s.balance.UpdateServices(list)
			//重新分配topic和service的对应关系
			logx.Info("-----------------------------------")
			topicService := make(map[string]string)
			for _, topic := range s.topics {
				services := s.balance.GetServiceByTopic(topic)
				logx.Infof("assign topic:%s, services:%+v", topic, services)
				topicService[topic] = topicData(topic, services)
			}
			logx.Info("-----------------------------------")
			//如果两个service 都在keepalive 会发生什么
			err := s.setTopicToEtcd(topicService)
			if err != nil {
				logx.Error(err)
			}
		} else {
			logx.Infof("I am not the leader, leader is %+v", leader)
			// 如果不是leader ,
			if s.etcdClient != nil {
				logx.Errorf("close %s etcdClient", s.sc.String())
				s.etcdClient.Close()
				s.etcdClient = nil
			}
		}
		return nil
	})
	return nil
}

// topic:s1,s2
func topicData(topic string, ss []ServiceInfo) string {
	var ts []string
	for _, s := range ss {
		ts = append(ts, s.String())
	}

	return topic + ":" + strings.Join(ts, TopicServiceSeq)
}

func parseTopicData(d string) (topic string, ss string, err error) {
	idx := strings.Index(d, ":")
	if idx == -1 {
		return "", "", fmt.Errorf("invalid topic data:%s", d)
	}
	topic = d[:idx]
	ss = d[idx+1:]
	return topic, ss, nil
}

func (s *Service) PublishLeader() {
	//TODO: publish leader to etcd
}
func (s *Service) IsLeader() bool {
	return s.isLeader
}

func (s *Service) Register() error {
	data, err := json.Marshal(s.sc)
	if err != nil {
		return err
	}
	//注册到etcd
	key := s.sc.Etcd.Key // 由于没有指定Etcd.ID, 所以最终注册的key是:/ns/as/key/7587883611715931480
	logx.Infof("register the service, key:%s, value:%s", key, string(data))
	s.pubClient = discov.NewPublisher(s.sc.Etcd.Hosts, key, string(data))
	if err := s.pubClient.KeepAlive(); err != nil {
		return err
	}
	return nil
}

func findLeader(list []ServiceConfig) *ServiceConfig {
	if len(list) == 0 {
		return nil
	}
	leaders := []ServiceConfig{}
	for _, v := range list {
		if v.IsLeader {
			leaders = append(leaders, v)
		}
	}
	if len(leaders) == 1 {
		return &leaders[0]
	}
	if len(leaders) == 0 {
		logx.Errorf("no leader found, chose min id")
	}

	leader := list[0]
	for _, v := range list {
		if v.Id < leader.Id {
			leader = v
		}
	}
	return &leader
}

func (s *Service) StartDiscovTopics() error {
	return DiscovTopics(s.sc.Etcd.Hosts, s.TopicsPath(), func(vals []string) {
		topicMetadata := make(map[string]string, len(vals))
		if s.topicPerKey {
			for _, v := range vals {
				topic, services, err := parseTopicData(v)
				if err != nil {
					logx.Error(err)
					continue
				}
				topicMetadata[topic] = services
			}
		} else {
			if len(vals) == 0 {
				logx.Errorf("no topic found")
				return
			}
			//所有信息都写在一个key:/ns/as/topics 里, 所以values 只有一个值。
			//以后可以考虑把所有topics信息分散在多个key里，比如/ns/as/topics/node1, /ns/as/topics/node2,这样某个node的负责的topic有变化时，只需要更新这个key即可。
			//values 是所有 topics信息, len(values) = 1, value:[topic6:topic_service-1;topic1:topic_service-1|topic_service-2]
			ts := strings.Split(vals[0], TopicsSep)
			for _, v := range ts {
				topic, services, err := parseTopicData(v) // v 格式 topic6:topic_service-1
				if err != nil {
					logx.Error(err)
					continue
				}
				topicMetadata[topic] = services
			}
		}
		logx.Infof("%s, get len:%d topicMetadata:%+v", s.sc.String(), len(topicMetadata), topicMetadata)
	})
}

// topic map handler, topic map: topic-->services
func DiscovTopics(hosts []string, key string, handle func([]string)) error {
	update := func(sub *discov.Subscriber) {
		vals := sub.Values()
		//get key:/ns/as/topics, len:1, value:[topic4:topic_service-1;topic6:topic_service-1;...]
		logx.Debugf("get key:%s, len:%d, value:%+v", key, len(vals), vals)
		handle(vals)
	}
	return DiscovAny(hosts, key, update)
}

func DiscovAny(hosts []string, key string, update func(sub *discov.Subscriber)) error {
	sub, err := discov.NewSubscriber(hosts, key)
	if err != nil {
		return err
	}
	sub.AddListener(func() { update(sub) })
	update(sub)
	return nil
}

func DiscovCustomService(hosts []string, key string, handle func([]ServiceConfig) error) error {
	update := func(sub *discov.Subscriber) {
		vals := sub.Values()
		logx.Debugf("get key:%s, value:%+v", key, vals)
		serviceList := []ServiceConfig{}
		for _, v := range vals {
			bs := ServiceConfig{}
			err := json.Unmarshal([]byte(v), &bs)
			if err != nil {
				logx.Error(err)
				continue
			}
			serviceList = append(serviceList, bs)
		}
		logx.Debug("serviceList:", serviceList)
		handle(serviceList)
	}
	return DiscovAny(hosts, key, update)
}

func (s *Service) TopicsPath() string {
	return fmt.Sprintf("/%s/%s/topics", s.sc.Ns, s.sc.As)
}

func (s *Service) TopicKey(topic string) string {
	return fmt.Sprintf("%s/%s", s.TopicsPath(), topic)
}

func (s *Service) setTopicToEtcd(topicService map[string]string) error {
	if s.etcdClient != nil {
		s.etcdClient.Close()
		s.etcdClient = nil
	}
	// 创建etcd客户端
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   s.sc.Etcd.Hosts, // etcd 服务地址
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return err
	}
	s.etcdClient = cli

	// 定义 TTL 时间（例如 10 秒）
	ttl := int64(20) // keepalive 会在这个时间1/3内发送心跳

	// 创建一个租约
	lease, err := cli.Grant(context.Background(), ttl)
	if err != nil {
		return err
	}

	if s.topicPerKey {
		// 这种方式是，每个topic对应一个key, 坏处是，设置一次，watch方会watch 很多次变更。
		// 并且发现bug,watch变更的最终结果跟etcd上的不一致(TODO)。
		// 构建事务操作
		txn := cli.Txn(context.Background())
		ops := []clientv3.Op{}
		for topic, service := range topicService {
			ops = append(ops, clientv3.OpPut(s.TopicKey(topic), service, clientv3.WithLease(lease.ID)))
		}
		txn = txn.Then(ops...)
		// 提交事务
		txnResp, err := txn.Commit()
		if err != nil {
			return err
		}

		// 检查事务执行结果
		if txnResp.Succeeded {
			logx.Info("All keys written successfully with TTL")
			// 保持租约（可选，如果需要长期续约）
			ch, kaErr := cli.KeepAlive(context.Background(), lease.ID)
			if kaErr != nil {
				logx.Errorf("KeepAlive error: %v\n", kaErr)
				return err
			}

			// 打印 KeepAlive 响应, 正常情况下，ttl/3 秒会收到一次心跳.
			go func() {
				for ka := range ch {
					logx.Infof("KeepAlive response: TTL=%d, service:%s\n", ka.TTL, s.sc.String())
				}
				logx.Errorf("service %s keepalive quit", s.sc.String())
			}()
			return nil
		} else {
			return errors.New("transaction failed")
		}
	} else {
		//这种方式是，所有topic对应一个key
		vals := make([]string, 0, len(topicService))
		for _, v := range topicService {
			vals = append(vals, v)
		}

		// 写入键值对到 etcd
		// key := s.TopicPath() //设置 /ns/as/topics , go-zero watch 不到数据， 所以设置 /ns/as/topics/all
		key := s.TopicKey("all") //设置成 /ns/as/topics/all，go-zero watch 到数据
		value := strings.Join(vals, TopicsSep)
		_, err = cli.Put(s.ctx, key, value, clientv3.WithLease(lease.ID))
		if err != nil {
			return err
		}
		//set etcd key:/ns/as/topics/all, value:topic6:topic_service-2;topic10:topic_service-1
		logx.Infof("set etcd key:%s, value:%s, written successfully with TTL", key, value)
		// 保持租约（可选，如果需要长期续约）
		ch, kaErr := cli.KeepAlive(s.ctx, lease.ID)
		if kaErr != nil {
			logx.Errorf("KeepAlive error: %v\n", kaErr)
			return err
		}
		// 打印 KeepAlive 响应, 正常情况下，ttl/3 秒会收到一次心跳.
		go func() {
			for ka := range ch {
				logx.Infof("KeepAlive response: TTL=%d, service:%s\n", ka.TTL, s.sc.String())
			}
			logx.Errorf("service %s keepalive quit", s.sc.String())
		}()
	}
	return nil
}
