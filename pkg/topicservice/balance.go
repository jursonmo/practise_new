package topicservice

import (
	"reflect"
	"sync"

	"github.com/jursonmo/practise_new/pkg/hash"
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
	oldNodes := make(map[string]struct{})
	for _, node := range h.chash.Nodes() {
		oldNodes[node] = struct{}{}
	}
	for _, v := range services {
		h.chash.Add(v)               //v 是ServiceInfo类型， 实现了String()方法，所以可以直接添加
		delete(oldNodes, v.String()) //把新的node 从oldNodes中删除，剩下的就是需要删除的node
	}
	//删除完后，剩下的就是需要添加的node
	logx.Infof("need to del oldNodes:%v", oldNodes)
	for node := range oldNodes {
		h.chash.Remove(node)
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
func NewBalance(balancer ServiceBalancer) *Balance {
	return &Balance{
		DefaultBalancer: balancer,
		Services:        []ServiceInfo{},
		TopicBalance:    make(map[string]*ServiceBalanceResult),
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
