package main

import (
	"github.com/ti-mo/conntrack"
	"github.com/ti-mo/netfilter"
)

func (i *IPFIX) GetConntrack() {
	cnx, err := conntrack.Dial(nil)

	if err != nil {
		logger.Println(err)
		return
	}

	defer cnx.Close()

	opts := conntrack.DumpOptions{
		ZeroCounters: false,
	}

	flows, err := cnx.Dump(&opts)

	if err != nil {
		logger.Println(err)
		return
	}

	for _, flow := range flows {
		if (flow.Status.DstNAT() || flow.Status.SrcNAT()) && !flow.Status.Dying() {
			key := getFlowKey(&flow.TupleOrig)
			natValue := flowPool.Get().(*[4]ipfix.DecodedField)
			natValue[0] = ipfix.DecodedField{ID: 225, Value: flow.TupleReply.IP.DestinationAddress.String()}
			natValue[1] = ipfix.DecodedField{ID: 227, Value: flow.TupleReply.Proto.DestinationPort}
			natValue[2] = ipfix.DecodedField{ID: 226, Value: flow.TupleReply.IP.SourceAddress.String()}
			natValue[3] = ipfix.DecodedField{ID: 228, Value: flow.TupleReply.Proto.SourcePort}
			i.natEvent.Store(key, natValue)
		}
	}

	events := make(chan conntrack.Event, 100)
	defer close(events)

	go func() {
		for event := range events {
			if event.Flow.Status.SrcNAT() || event.Flow.Status.DstNAT() {
				key := getFlowKey(&event.Flow.TupleOrig)
				switch event.Type {
				case conntrack.EventNew:
					natValue := flowPool.Get().(*[4]ipfix.DecodedField)
					natValue[0] = ipfix.DecodedField{ID: 225, Value: event.Flow.TupleReply.IP.DestinationAddress.String()}
					natValue[1] = ipfix.DecodedField{ID: 227, Value: event.Flow.TupleReply.Proto.DestinationPort}
					natValue[2] = ipfix.DecodedField{ID: 226, Value: event.Flow.TupleReply.IP.SourceAddress.String()}
					natValue[3] = ipfix.DecodedField{ID: 228, Value: event.Flow.TupleReply.Proto.SourcePort}
					i.natEvent.Store(key, natValue)
				case conntrack.EventDestroy:
					val, ok := i.natEvent.Load(key)
					if ok {
						i.natEvent.Delete(key)
						flowPool.Put(val)
					}
				default:
					// 处理其他事件类型
				}
			}
		}
	}()

	// 订阅 conntrack 的新建和删除事件
	errChan, err := cnx.Listen(events, 1, []netfilter.NetlinkGroup{netfilter.GroupCTNew, netfilter.GroupCTDestroy})
	if err != nil {
		logger.Printf("listen conntrack err: %v", err)
		return
	}

	// 处理监听过程中的错误
	for listenErr := range errChan {
		logger.Printf("listen conntrack err: %v", listenErr)
	}

}
