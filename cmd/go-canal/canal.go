package main

import (
	"fmt"

	"github.com/go-mysql-org/go-mysql/canal"
	"github.com/siddontang/go-log/log"
)

type MyEventHandler struct {
	canal.DummyEventHandler
}

func (h *MyEventHandler) OnRow(e *canal.RowsEvent) error {
	log.Infof("%s, len(row):%d, %#v, header:%#v\n", e.Action, len(e.Rows), e.Rows, e.Header)

	//此处是参考 https://github.com/gitstliu/MysqlToAll 里面的获取字段和值的方法
	for columnIndex, currColumn := range e.Table.Columns {
		//字段名，字段的索引顺序，字段对应的值
		row := fmt.Sprintf("%v, %v, %v", columnIndex, currColumn.Name, e.Rows[len(e.Rows)-1][columnIndex])
		fmt.Println("row info:", row)
	}
	return nil
}

func (h *MyEventHandler) String() string {
	return "MyEventHandler"
}

func main() {
	cfg := canal.NewDefaultConfig()
	cfg.Addr = "127.0.0.1:3306"
	cfg.User = "root"
	// We only care table canal_test in test db
	cfg.Dump.TableDB = "nelbroker"        //"test"
	cfg.Dump.Tables = []string{"clients"} //"canal_test"

	c, err := canal.NewCanal(cfg)
	if err != nil {
		log.Fatal(err)
	}

	// Register a handler to handle RowsEvent
	c.SetEventHandler(&MyEventHandler{})

	// Start canal
	c.Run()
}

/*
[2024/11/29 19:13:27] [info] binlogsyncer.go:190 create BinlogSyncer with config {ServerID:1928 Flavor:mysql Host:127.0.0.1 Port:3306 User:root Password: Localhost: Charset:utf8 SemiSyncEnabled:false RawModeEnabled:false TLSConfig:<nil> ParseTime:false TimestampStringLocation:UTC UseDecimal:false RecvBufferSize:0 HeartbeatPeriod:0s ReadTimeout:0s MaxReconnectAttempts:0 DisableRetrySync:false VerifyChecksum:false DumpCommandFlag:0 Option:<nil> Logger:0x140001bc360 Dialer:0x1007304d0 RowsEventDecodeFunc:0x1008e9a70 TableMapOptionalMetaDecodeFunc:<nil> DiscardGTIDSet:false EventCacheCount:10240 SynchronousEventHandler:<nil>}
[2024/11/29 19:13:27] [info] dump.go:171 try dump MySQL and parse
[2024/11/29 19:13:27] [info] dumper.go:309 exec mysqldump with [--host=127.0.0.1 --port=3306 --user=root --password=****** --source-data --single-transaction --skip-lock-tables --compact --skip-opt --quick --no-create-info --skip-extended-insert --skip-tz-utc --hex-blob --default-character-set=utf8 --column-statistics=0 nelbroker clients]
[2024/11/29 19:13:27] [info] cana.go:15 insert, len(row):1, [][]interface {}{[]interface {}{1, "2024-10-30 16:10:52", "2024-11-06 12:02:21", 0x0, 0, "mock-clientID-01", "mock-topic", "mock-topic-type", "1.1.7,2.2.2", "1.1.1.1/32,1.1.1.2/32"}}, header:(*replication.EventHeader)(nil)
row info: 0, id, 1
row info: 1, create_at, 2024-10-30 16:10:52
row info: 2, update_at, 2024-11-06 12:02:21
row info: 3, is_delete, 0
row info: 4, version, 0
row info: 5, ecid, mock-clientID-01
row info: 6, customer, mock-topic
row info: 7, topicType, mock-topic-type
row info: 8, nels, 1.1.7,2.2.2
row info: 9, networks, 1.1.1.1/32,1.1.1.2/32
[2024/11/29 19:13:27] [info] cana.go:15 insert, len(row):1, [][]interface {}{[]interface {}{2, "2024-11-29 15:55:28", "2024-11-29 15:55:28", 0x0, 0, "mock-clientID-02", "mock-topic", "mock-topic-type", "2.1.1", "2.1.1.1/32"}}, header:(*replication.EventHeader)(nil)
row info: 0, id, 2
row info: 1, create_at, 2024-11-29 15:55:28
row info: 2, update_at, 2024-11-29 15:55:28
row info: 3, is_delete, 0
row info: 4, version, 0
row info: 5, ecid, mock-clientID-02
row info: 6, customer, mock-topic
row info: 7, topicType, mock-topic-type
row info: 8, nels, 2.1.1
row info: 9, networks, 2.1.1.1/32
[2024/11/29 19:13:27] [info] dump.go:187 dump MySQL and parse OK, use 0.07 seconds, start binlog replication at (binlog.000170, 848)
[2024/11/29 19:13:27] [info] binlogsyncer.go:442 begin to sync binlog from position (binlog.000170, 848)
[2024/11/29 19:13:27] [info] binlogsyncer.go:408 Connected to mysql 9.0.1 server
[2024/11/29 19:13:27] [info] sync.go:22 start sync binlog at binlog file (binlog.000170, 848)
[2024/11/29 19:13:27] [info] binlogsyncer.go:869 rotate to (binlog.000170, 848)
[2024/11/29 19:13:27] [info] sync.go:59 received fake rotate event, next log name is binlog.000170
*/
