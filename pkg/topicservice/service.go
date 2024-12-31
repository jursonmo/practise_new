package topicservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/jursonmo/practise_new/pkg/hash"
	"github.com/zeromicro/go-zero/core/discov"
	"github.com/zeromicro/go-zero/core/logx"
)

type BalanceInfo struct {
	BalanceName string
	Desc        string
	BalanceFunc func(topic string, services []ServiceInfo) []ServiceInfo
}

type ServiceBalancer interface {
	Init(services []ServiceInfo)
	Name() string
	Desc() string
	Balance(topic string, services []ServiceInfo) []ServiceInfo
}

type myConsistentHash struct {
	chash *hash.ConsistentHash
	name  string
	desc  string
}

func (h *myConsistentHash) Name() string {
	return h.name
}
func (h *myConsistentHash) Desc() string {
	return h.desc
}
func (h *myConsistentHash) Init(services []ServiceInfo) {
	//h.chash.Add node 时, 如果node已经存在,先删除再添加。
	logx.Infof("before init nodes:%v", h.chash.Nodes())
	for _, v := range services {
		h.chash.Add(v) //v 是ServiceInfo类型， 实现了String()方法，所以可以直接添加
	}
	logx.Infof("after init nodes:%v", h.chash.Nodes())
}
func (h *myConsistentHash) Balance(topic string, services []ServiceInfo) []ServiceInfo {
	v, ok := h.chash.Get(topic)
	if !ok {
		return nil
	}
	broker := v.(ServiceInfo)
	return []ServiceInfo{broker}
}

type Balance struct {
	sync.Mutex
	DefaultBalancer ServiceBalancer
	Services        []ServiceInfo                    //all Services
	TopicBalance    map[string]*ServiceBalanceResult //key: topic, topic 对应的负载算法以及结果(对应的service/broker列表)
}

type ServiceBalanceResult struct {
	Topic string
	ServiceBalancer
	ServiceList []ServiceInfo //负载算法计算后的结果 result
}

var DefaultBalance *Balance

func init() {
	// 初始化balances
	//注册默认的负载算法是一致性hash算法
	DefaultBalance = &Balance{
		DefaultBalancer: &myConsistentHash{
			chash: hash.NewConsistentHash(),
			name:  "consistent_hash",
			desc:  "consistent hash balance alg",
		},
		Services:     []ServiceInfo{},
		TopicBalance: make(map[string]*ServiceBalanceResult),
	}
}

// 当服务列表发生变化，需要重新就算并更新已经存在缓存中的负载结果
func (b *Balance) UpdateServices(services []ServiceInfo) {
	b.Lock()
	defer b.Unlock()
	if reflect.DeepEqual(b.Services, services) {
		//service list not changed, do nothing, return
		return
	}
	logx.Infof("balance services change:%+v", services)
	b.Services = services
	//update default balancer
	b.DefaultBalancer.Init(b.Services)
	//update topic balance , service list 发生更新后，重新更新已经存在缓存中的负载结果
	for topic, bb := range b.TopicBalance {
		//根据最新的 Services 和 原来缓存的 ServiceBalancer 来重新计算结果
		bb.ServiceList = bb.ServiceBalancer.Balance(topic, b.Services)
	}
}
func (b *Balance) UpdateTopicBalancer(topic string, balancer ServiceBalancer) {
	//TODO: 当topic对应的负载算法发生变化时，需要重新计算结果

}

func (b *Balance) UpdateDefaultBalancer(balancer ServiceBalancer) {
	b.Lock()
	defer b.Unlock()
	oldDefaultBalancer := b.DefaultBalancer
	b.DefaultBalancer = balancer
	for topic, bb := range b.TopicBalance {
		//如何缓存里的负载算法是旧的oldDefaultBalancer, 那么就更新负载算法，并重新计算结果。
		//如果不是，那么就是特别对该topic指定的负载器，不需要重新计算结果，保持原样
		if bb.ServiceBalancer == oldDefaultBalancer {
			bb.ServiceBalancer = b.DefaultBalancer
			//根据最新的 Services 和 原来缓存的 ServiceBalancer 来重新计算结果
			bb.ServiceList = bb.ServiceBalancer.Balance(topic, b.Services)
		}
	}
}

func (b *Balance) GetServiceByTopic(topic string) []ServiceInfo {
	bb := b.GetBalanceResult(topic)
	if bb == nil {
		return nil
	}
	return bb.ServiceList
}

// 根据topic获取对应的负载算法以及结果
func (b *Balance) GetBalanceResult(topic string) *ServiceBalanceResult {
	b.Lock()
	defer b.Unlock()
	//先从缓存中获取
	if bb, ok := b.TopicBalance[topic]; ok {
		return bb
	}
	//缓存中没有，那么就计算并缓存
	bb := &ServiceBalanceResult{
		Topic:           topic,
		ServiceBalancer: b.DefaultBalancer,
		ServiceList:     b.DefaultBalancer.Balance(topic, b.Services),
	}
	b.TopicBalance[topic] = bb
	return bb
}

// ------------------------------------------------------------
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

type Service struct {
	sync.Mutex
	sc        *ServiceConfig
	isLeader  bool //是否是leader, 只有leader才会分配topic和service(broker)的对应关系
	pubClient *discov.Publisher

	balance *Balance
	topics  []string
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
		sc:      sc,
		balance: DefaultBalance, //默认使用一致性hash算法
		//topicServiceMap: make(map[string]*ServiceConfig),
	}, nil
}
func (s *Service) SetDefaultBalancer(balance ServiceBalancer) {
	s.balance.UpdateDefaultBalancer(balance)
}

func (s *Service) Start(ctx context.Context) error {
	//注册到etcd
	if err := s.Register(); err != nil {
		return err
	}
	//订阅etcd
	if err := s.StartDiscov(); err != nil {
		return err
	}
	//监听etcd的事件
	//TODO: 监听etcd的事件，当leader发生变化时，重新分配topic和service的对应关系
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

func (s *Service) StartDiscov() error {
	DiscovCustomService(s.sc.Etcd, func(list []ServiceConfig) error {
		logx.Debugf("get Custom %s service:%+v", s.sc.Etcd.Key, list)
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
			for _, topic := range s.topics {
				services := s.balance.GetServiceByTopic(topic)
				logx.Infof("assign topic:%s, services:%+v", topic, services)
			}
			logx.Info("-----------------------------------")
		} else {
			logx.Infof("I am not the leader, leader is %+v", leader)
		}
		return nil
	})
	return nil
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

func DiscovCustomService(c discov.EtcdConf, handle func([]ServiceConfig) error) error {
	sub, err := discov.NewSubscriber(c.Hosts, c.Key)
	if err != nil {
		return err
	}

	update := func() {
		logx.Infof("watch Custom service of %s update", c.Key)
		vals := sub.Values()
		logx.Debugf("get Custom service:%+v", vals)
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
	sub.AddListener(update)
	update()
	return nil
}
