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

//  1. 所有服务都watch , 感知其他服务的存在，并觉得哪个是leader， leader通过一定的算法，分配当前每个topic分别对应的services, 并更新到/ns/as/topics/xxx
//  2. 每个服务都订阅/ns/as/topics/信息，并保存topic和service的对应关系, 供客户端获取，因为客户端发送或者订阅某个topic时，会优先连接topic对应的service,
//     即topic和service的对应关系作为client的推荐建议，即推荐客户端对某个topic的操作
//  3. 但是客户端不一定按照推荐的来连接指定service, 比如：client连不上推荐的service,只能连任意一个service 或者部署的时候，用ngnix来作为统一负载入口, client 看到的只有一个入口
//     client 没有选service的权利, nginx有自己的负载算法来连接后端的service
//  4. 所以需要有地方记录实时的topic 被客户端订阅在哪个service上，这样service之间转发topic public数据时，才能找到对应service,再有对应的service转发给订阅的client.
//     + 如果用etcd 来记录真实实时的对应关系, 可以这样：/ns/as/ontime/topicX/service1,  /ns/as/ontime/topicX/service2, 这样就形成了topicX-->service1,service2的关系。
//     服务器需要把所有key关联一个租约，如果服务器挂了，etcd服务器自动删除该service 相关的信息，比如，/ns/as/ontime/topicX/service1，/ns/as/ontime/topicY/service1
//     其他所有服务只需要侦听/ns/as/ontime/，
//     + 如果用redis 来记录实时的对应关系，可以以topicX作为一个set key, values 是 service1,service2..., 如果service1上有client订阅topic1,那么service自己负责在set:topic1 集合里加上service1
//     当service 接受到其他服务转发过来的数据时，如果发现数据的topic没有客户端在侦听，需要去redis 删除相关信息，并利用redis的pub sub 通知其他服务。
//     + 不需要中间件来同步topic-service对应关系，用gosip？ redis cluster 用了gossip 来同步slot信息, consul 也基于gossip的实现健康检查, 区块链等项目都有使用Gossip
//     目前已经可以用gossip 来同步topic-service的对应关系. TODO: 有客户端订阅topic时，service.AddTopicState(topic) 通知其他服务，
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
	Gossip    GossipConf        `json:",optional"`
}

type GossipConf struct {
	Enabled bool   `json:",optional"`
	Addr    string `json:",optional"`
	Port    int    `json:",optional"`
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

	topicState *TopicState

	balance     *Balance
	serviceList []ServiceInfo
	topics      []string
	etcdClient  *clientv3.Client
	lease       *clientv3.LeaseGrantResponse //租约,用于撤销
	topicPerKey bool
	//topic和service的对应关系
	// topicServiceMap map[string]*ServiceConfig

	distributedTopics map[string]string
}

func (s ServiceInfo) String() string {
	return fmt.Sprintf("%s-%s", s.Name, s.Id)
}

// 服务器唯一Key,唯一标识一个服务
func (s *Service) Key() string {
	return s.sc.String()
}

func (s *Service) Endpoints() []string {
	return s.sc.Endpoints
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

	s := &Service{
		sc: sc,
		//balance: DefaultBalance, //应该每个服务创建一个Balance
		balance: NewBalance(&myConsistentHash{
			chash: hash.NewConsistentHash(),
			name:  "consistent_hash",
			desc:  "consistent hash balance alg",
		}), //默认使用一致性hash算法
		//topicServiceMap: make(map[string]*ServiceConfig),
	}

	if s.sc.Gossip.Enabled {
		topicState, err := NewTopicState(s.Key(), sc.Gossip.Addr, sc.Gossip.Port)
		if err != nil {
			panic(err)
		}
		s.topicState = topicState
	}
	return s, nil
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

	//gossip
	if s.topicState != nil {
		//记录当前topics, 不广播。如果在加入集群前, 发布自己的topics , 会发生什么。
		s.topicState.UpdateTopic(ADD, s.topics, false) //表示所有服务器都订阅了这几个topics

		//定时打印topic state
		go func() {
			for {
				time.Sleep(time.Second * 10)
				logx.Info("----topic state members:", s.topicState.Members())
				logx.Info("----topic state local topics:", s.topicState.GetLocalTopics())
				logx.Infof("----topic state global topics:%+v", s.topicState.GetTopics())
			}
		}()
	}

	return nil
}

func (s *Service) Stop() error {
	if s.pubClient != nil {
		s.pubClient.Stop()
	}
	if s.cancel != nil {
		s.cancel()
	}
	if s.etcdClient != nil {
		if s.lease != nil {
			//撤销租约
			logx.Infof("revoke %s lease", s.sc.String())
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			_, err := s.etcdClient.Revoke(ctx, s.lease.ID)
			if err != nil {
				logx.Error(err)
			}
			cancel()
		}
		s.etcdClient.Close()
	}
	return nil
}

func (s *Service) AddTopicState(topic string) {
	if s.topicState != nil {
		s.topicState.AddTopic([]string{topic})
	}
}
func (s *Service) DelTopicState(topic string) {
	if s.topicState != nil {
		s.topicState.DelTopic([]string{topic})
	}
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
		s.serviceList = list

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

			// 由leader 去join 其他的service
			logx.Info("i am leader, so gossip join other service")
			err = s.ServicesJoin()
			if err != nil {
				logx.Error(err)
			}
			logx.Info("topicState members:", s.topicState.Members())
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

func (s *Service) ServicesJoin() error {
	if s.topicState == nil {
		return nil
	}

	topicStateNode := make([]string, 0, len(s.serviceList))
	for _, service := range s.serviceList {
		// 排除自己
		if service.Id == s.sc.Id {
			continue
		}
		//如果服务节点没有开启gossip， 就不加入
		if !service.Gossip.Enabled {
			continue
		}
		topicStateNode = append(topicStateNode, fmt.Sprintf("%s:%d", service.Gossip.Addr, service.Gossip.Port))
	}

	logx.Info("topicStateNode:", topicStateNode)
	if len(topicStateNode) == 0 {
		return nil
	}
	err := s.topicState.Join(topicStateNode)
	if err != nil {
		logx.Error("topicState join err:", err)
		return err
	}

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
		logx.Info("no leader found, chose min id")
	}

	leader := list[0]
	for _, v := range list {
		if v.Id < leader.Id {
			leader = v
		}
	}
	return &leader
}

func (s *Service) SetDistributedTopics(topics map[string]string) {
	s.Lock()
	defer s.Unlock()
	s.distributedTopics = topics
}

func (s *Service) StartDiscovTopics() error {
	return DiscovTopics(s.sc.Etcd.Hosts, s.TopicsPath(), func(vals []string) {
		// 这是leader 经过负载算法计算后推荐的 topic-->service 信息
		distributedTopics := make(map[string]string, len(vals))
		if s.topicPerKey {
			for _, v := range vals {
				topic, services, err := parseTopicData(v)
				if err != nil {
					logx.Error(err)
					continue
				}
				distributedTopics[topic] = services
			}
		} else {
			if len(vals) == 0 {
				logx.Errorf("no topic found")
				return
			}
			//所有信息都写在一个key:/ns/as/topics/all 里, 所以values 只有一个值。
			//以后可以考虑把所有topics信息分散在多个key里，比如/ns/as/topics/node1, /ns/as/topics/node2,这样某个node的负责的topic有变化时，只需要更新这个key即可。
			//values 是所有 topics信息, len(values) = 1, value:[topic6:topic_service-1;topic1:topic_service-1|topic_service-2]
			ts := strings.Split(vals[0], TopicsSep)
			for _, v := range ts {
				topic, services, err := parseTopicData(v) // v 格式 topic6:topic_service-1
				if err != nil {
					logx.Error(err)
					continue
				}
				distributedTopics[topic] = services
			}
		}
		logx.Infof("%s, get len:%d distributedTopics:%+v", s.sc.String(), len(distributedTopics), distributedTopics)
		//保存指派的topic和service对应关系
		s.SetDistributedTopics(distributedTopics)
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
	var err error
	if s.etcdClient == nil {
		// 创建etcd客户端
		s.etcdClient, err = clientv3.New(clientv3.Config{
			Endpoints:   s.sc.Etcd.Hosts, // etcd 服务地址
			DialTimeout: 5 * time.Second,
		})
		if err != nil {
			return err
		}

	}
	if s.lease != nil {
		// 需要撤销之前租约, 相当于关闭keepalive, 同时删除关联的key.
		//否则会出现watch 相同key的内容的bug。即会出现 topic4:topic_service-1;topic4:topic_service-2
		_, err = s.etcdClient.Revoke(s.ctx, s.lease.ID)
		if err != nil {
			panic(err)
		}
		time.Sleep(time.Second)
		logx.Infof("=============== revoke lease:%d =================", s.lease.ID)
	}

	// 删除以 "/ns/as/topics" 开头的所有键, 避免其他服务残留数据, 如果不删除，直接更新已经存在的key, disov watch 到的内容也有点问题，同一个key的新旧value都能读到。
	deleteResp, err := s.etcdClient.Delete(s.ctx, s.TopicsPath(), clientv3.WithPrefix())
	if err != nil {
		logx.Error(err)
		return err
	}
	logx.Infof("Deleted %d keys with prefix %s", deleteResp.Deleted, s.TopicsPath())

	// 定义 TTL 时间（例如 10 秒）
	ttl := int64(20) // keepalive 会在这个时间1/3内发送心跳

	// 创建一个租约
	s.lease, err = s.etcdClient.Grant(s.ctx, ttl)
	if err != nil {
		return err
	}

	cli := s.etcdClient
	lease := s.lease

	//s.topicPerKey = true // 测试每个topic对应一个key。
	if s.topicPerKey {
		var successed bool
		// 这种方式是，每个topic对应一个key, 坏处是，设置一次，watch方会watch 很多次变更。
		// 并且发现bug, 用事务的方式提交或者多次put更新, watch变更的最终结果跟etcd上的不一致(TODO)。
		// (done: 修复bug的方法是先撤销租约，把所有关联key删除，再设置所有key)
		txn := true
		if txn {
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
			successed = txnResp.Succeeded
		} else {
			// 多次put更新, 出错也不会回滚。可能会部分成功, 部分失败的情况，导致数据不一致，这是不推荐的。
			for topic, service := range topicService {
				//time.Sleep(time.Millisecond * 100)
				_, err = cli.Put(s.ctx, s.TopicKey(topic), service, clientv3.WithLease(lease.ID))
				if err != nil {
					return err
				}
			}
			successed = true
		}
		// 检查事务执行结果
		if successed {
			logx.Info("All keys written successfully with TTL")
			// 保持租约（可选，如果需要长期续约）
			ch, kaErr := cli.KeepAlive(s.ctx, lease.ID)
			if kaErr != nil {
				logx.Errorf("KeepAlive error: %v\n", kaErr)
				return err
			}

			// 打印 KeepAlive 响应, 正常情况下，ttl/3 秒会收到一次心跳.
			go func() {
				var ka = &clientv3.LeaseKeepAliveResponse{}
				for ka = range ch {
					logx.Infof("KeepAlive response: ID:%d, TTL=%d, service:%s\n", ka.ID, ka.TTL, s.sc.String())
				}
				logx.Errorf("service %s lease id:%d keepalive quit", s.sc.String(), ka.ID)
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
			var ka = &clientv3.LeaseKeepAliveResponse{}
			for ka = range ch {
				logx.Infof("KeepAlive response: TTL=%d, service:%s\n", ka.TTL, s.sc.String())
			}
			logx.Errorf("service %s lease id:%d keepalive quit", s.sc.String(), ka.ID)
		}()
	}
	return nil
}
