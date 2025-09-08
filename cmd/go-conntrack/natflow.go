package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"

	"github.com/ti-mo/conntrack"
	"github.com/ti-mo/netfilter"
)

func getFlowKey(tuple *conntrack.Tuple) string {
	// 根据 IP 和端口构建 key，预先分配长度可减少分配开销
	return tuple.IP.SourceAddress.String() + "-" + strconv.Itoa(int(tuple.Proto.SourcePort)) +
		"-" + tuple.IP.DestinationAddress.String() + "-" + strconv.Itoa(int(tuple.Proto.DestinationPort)) +
		"-" + strconv.Itoa(int(tuple.Proto.Protocol))
}

type FlowMgr struct {
	NatFlow sync.Map
	conn    *conntrack.Conn
}

func main() {
	logger := log.New(os.Stderr, "conntrack: ", log.LstdFlags)
	conn, err := conntrack.Dial(nil)
	if err != nil {
		logger.Println(err)
		return
	}
	defer conn.Close()

	f := &FlowMgr{
		conn: conn,
	}
	f.DumpFlows()
	f.WatchFlow()

}

func (f *FlowMgr) DumpFlows() {
	c := f.conn
	opts := conntrack.DumpOptions{
		ZeroCounters: false,
	}

	flows, err := c.Dump(&opts)
	// 用来过滤 conntrack 表中的 mark ctmark flows
	// flows, err := c.DumpFilter(conntrack.Filter{
	// 	Mark: 0,
	// 	Mask: 0,
	// }, &opts)

	if err != nil {
		fmt.Printf("get conntrack flows err: %v", err)
		return
	}

	for _, flow := range flows {
		if (flow.Status.DstNAT() || flow.Status.SrcNAT()) && !flow.Status.Dying() {
			key := getFlowKey(&flow.TupleOrig)
			f.NatFlow.Store(key, flow)
		}
	}
}

func (f *FlowMgr) WatchFlow() {
	logger := log.New(os.Stderr, "conntrack: ", log.LstdFlags)
	c := f.conn

	events := make(chan conntrack.Event, 100)
	defer close(events)

	go func() {
		for event := range events {
			if event.Flow == nil {
				continue
			}
			// // 过滤 mark ctmark flows
			// if event.Flow.Mark == 0 {
			// 	continue
			// }
			if event.Flow.Status.SrcNAT() || event.Flow.Status.DstNAT() {
				switch event.Type {
				case conntrack.EventNew:
					logger.Printf("new nat flow: %v", event.Flow)
					key := getFlowKey(&event.Flow.TupleOrig)
					f.NatFlow.Store(key, event.Flow)
				case conntrack.EventDestroy:
					logger.Printf("destroy nat flow: %v", event.Flow)
					key := getFlowKey(&event.Flow.TupleOrig)
					f.NatFlow.Delete(key)
				}
			}
		}
	}()

	// 订阅 conntrack 的新建和删除事件, 这里没法只侦听 mark ctmark flows, 只能拿到所有的事件再过滤。
	errChan, err := c.Listen(events, 1, []netfilter.NetlinkGroup{netfilter.GroupCTNew, netfilter.GroupCTDestroy})
	if err != nil {
		logger.Printf("listen conntrack err: %v", err)
		return
	}

	// 处理监听过程中的错误
	for listenErr := range errChan {
		logger.Printf("listen conntrack err: %v", listenErr)
		os.Exit(1)
	}
	logger.Printf("watch conntrack event")

}
