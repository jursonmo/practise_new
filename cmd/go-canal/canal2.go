package main

//利用基于canal的go库来监听binlog日志
//https://github.com/marcus-ma/myBlog/issues/11
import (
	"fmt"
	"reflect"
	"time"

	"github.com/go-mysql-org/go-mysql/canal"
	"github.com/go-mysql-org/go-mysql/mysql"
	"github.com/go-mysql-org/go-mysql/replication"

	"github.com/mitchellh/mapstructure"
)

type Clients struct {
	Id        int64     `json:"id"`
	CreateAt  time.Time `json:"create_at"`
	UpdateAt  time.Time `json:"update_at"`
	IsDelete  uint64    `json:"is_delete"`
	Version   int64     `json:"version"`
	Ecid      string    `json:"ecid"`
	Customer  string    `json:"customer"`
	TopicType string    `json:"topicType"`
	Nels      string    `json:"nels"`
	Networks  string    `json:"networks"`
}

type MyEventHandler struct {
	canal.DummyEventHandler
}

// 监听数据记录
func (h *MyEventHandler) OnRow(ev *canal.RowsEvent) error {
	//record := fmt.Sprintf("%s %v %v %v %s\n",e.Action,e.Rows,e.Header,e.Table,e.String())

	//库名，表名，行为，数据记录
	record := fmt.Sprintf("%v %v %s %v\n", ev.Table.Schema, ev.Table.Name, ev.Action, ev.Rows)
	fmt.Println(record)

	m := make(map[string]interface{})
	//此处是参考 https://github.com/gitstliu/MysqlToAll 里面的获取字段和值的方法
	for columnIndex, currColumn := range ev.Table.Columns {
		//字段名，字段的索引顺序，字段对应的值
		row := fmt.Sprintf("%v %v %v\n", currColumn.Name, columnIndex, ev.Rows[len(ev.Rows)-1][columnIndex])
		fmt.Println("row info:", row)
		m[currColumn.Name] = ev.Rows[len(ev.Rows)-1][columnIndex]
	}
	fmt.Printf("m:%+v\n", m)
	fmt.Println("m.create_at type:", reflect.TypeOf(m["create_at"]))

	//mapstructure.Decode 方法将map转换为结构体
	var clients Clients
	err := mapstructure.Decode(m, &clients)
	if err != nil {
		fmt.Println("err:", err)
	}
	fmt.Printf("clients:%+v\n", clients)

	var md mapstructure.Metadata
	clients = Clients{} // 重新初始化
	cfg := &mapstructure.DecoderConfig{
		Metadata: &md,
		Result:   &clients,
		TagName:  "json",
	}
	decoder, _ := mapstructure.NewDecoder(cfg)
	err = decoder.Decode(m)
	if err != nil {
		fmt.Println("err:", err)
	}
	fmt.Printf("md:%+v\n", md)
	fmt.Printf("2222 clients:%+v\n", clients) // 无法将时间类型的字段转换为time.Time类型 CreateAt:0001-01-01 00:00:00

	// 发现一个问题，无法将时间类型的字段转换为time.Time类型，所以需要自己写一个方法来转换
	// 自定义 DecodeHook
	hook := mapstructure.ComposeDecodeHookFunc(
		mapstructure.StringToTimeHookFunc("2006-01-02 15:04:05"), // 时间格式
	)

	clients = Clients{}          // 重新初始化
	md = mapstructure.Metadata{} // 重新初始化
	// 使用自定义解码器
	decoderConfig := &mapstructure.DecoderConfig{
		Metadata:   &md,
		DecodeHook: hook,
		Result:     &clients,
		TagName:    "json", // 必须指定标签名，否则无法将create_at 反序列成 CreateAt字段,  UdpateAt DeleteAt 也一样。
	}

	decoder, err = mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		panic(err)
	}
	err = decoder.Decode(m)
	if err != nil {
		panic(err)
	}
	fmt.Printf("自定义解码器 md:%+v\n", md)
	fmt.Printf("自定义解码器 clients:%+v\n\n", clients)
	return nil
}

// 创建、更改、重命名或删除表时触发，通常会需要清除与表相关的数据，如缓存。It will be called before OnDDL.
func (h *MyEventHandler) OnTableChanged(_ *replication.EventHeader, schema string, table string) error {
	//库，表
	record := fmt.Sprintf("%s %s \n", schema, table)
	fmt.Println(record)
	return nil
}

// 监听binlog日志的变化文件与记录的位置，从库同步完后binlog的记录offset后，会触发该方法。
// 可以是mysql 通过命令行  show master status 查看。
func (h *MyEventHandler) OnPosSynced(_ *replication.EventHeader, pos mysql.Position, set mysql.GTIDSet, force bool) error {
	//源码：当force为true，立即同步位置.
	record := fmt.Sprintf("%v %v \n", pos.Name, pos.Pos)
	fmt.Println("OnPosSynced", record) //OnPosSynced binlog.000170 848
	return nil
}

// 当产生新的binlog日志后触发(在达到内存的使用限制后（默认为 1GB），会开启另一个文件，每个新文件的名称后都会有一个增量。)
func (h *MyEventHandler) OnRotate(_ *replication.EventHeader, r *replication.RotateEvent) error {
	//record := fmt.Sprintf("On Rotate: %v \n",&mysql.Position{Name: string(r.NextLogName), Pos: uint32(r.Position)})
	//binlog的记录位置，新binlog的文件名
	record := fmt.Sprintf("On Rotate %v %v \n", r.Position, r.NextLogName)
	fmt.Println(record)
	return nil

}

// create alter drop truncate(删除当前表再新建一个一模一样的表结构)
func (h *MyEventHandler) OnDDL(_ *replication.EventHeader, nextPos mysql.Position, queryEvent *replication.QueryEvent) error {
	//binlog日志的变化文件与记录的位置
	record := fmt.Sprintf("%v %v\n", nextPos.Name, nextPos.Pos)
	query_event := fmt.Sprintf("%v\n %v\n %v\n %v\n %v\n",
		queryEvent.ExecutionTime,         //猜是执行时间，但测试显示0
		string(queryEvent.Schema),        //库名
		string(queryEvent.Query),         //变更的sql语句
		string(queryEvent.StatusVars[:]), //测试显示乱码
		queryEvent.SlaveProxyID)          //从库代理ID？
	fmt.Println("OnDDL:", record, query_event)
	return nil
}

func (h *MyEventHandler) String() string {
	return "MyEventHandler"
}

func main() {
	//读取toml文件格式
	//canal.NewConfigWithFile()
	cfg := canal.NewDefaultConfig()
	cfg.Addr = "127.0.0.1:3306"
	cfg.User = "root"
	cfg.Password = ""

	cfg.Dump.TableDB = "nelbroker"        //"test"
	cfg.Dump.Tables = []string{"clients"} //"canal_test"

	c, err := canal.NewCanal(cfg)
	if err != nil {
		fmt.Println("error", err)
	}

	c.SetEventHandler(&MyEventHandler{})

	fmt.Println("Go run")
	//从头开始监听
	c.Run()

	//根据位置监听
	//mysql-bin.000004, 1027
	// startPos := mysql.Position{Name: "mysql-bin.000004", Pos: 1027}
	// c.RunFrom(startPos)
}

/* 下面都是OnRow函数打印的：

nelbroker clients insert [[2 2024-11-29 15:55:28 2024-11-29 15:55:28 0 0 mock-clientID-02 mock-topic mock-topic-type 2.1.1 2.1.1.1/32]]

row info: id 0 2

row info: create_at 1 2024-11-29 15:55:28

row info: update_at 2 2024-11-29 15:55:28

row info: is_delete 3 0

row info: version 4 0

row info: ecid 5 mock-clientID-02

row info: customer 6 mock-topic

row info: topicType 7 mock-topic-type

row info: nels 8 2.1.1

row info: networks 9 2.1.1.1/32

m:map[create_at:2024-11-29 15:55:28 customer:mock-topic ecid:mock-clientID-02 id:2 is_delete:0 nels:2.1.1 networks:2.1.1.1/32 topicType:mock-topic-type update_at:2024-11-29 15:55:28 version:0]
m.create_at type: string
clients:{Id:2 CreateAt:0001-01-01 00:00:00 +0000 UTC UpdateAt:0001-01-01 00:00:00 +0000 UTC IsDelete:0 Version:0 Ecid:mock-clientID-02 Customer:mock-topic TopicType:mock-topic-type Nels:2.1.1 Networks:2.1.1.1/32}
err: 2 error(s) decoding:

* 'create_at' expected a map, got 'string'
* 'update_at' expected a map, got 'string'
md:{Keys:[id create_at update_at is_delete version ecid customer topicType nels networks] Unused:[] Unset:[]}
2222 clients:{Id:2 CreateAt:0001-01-01 00:00:00 +0000 UTC UpdateAt:0001-01-01 00:00:00 +0000 UTC IsDelete:0 Version:0 Ecid:mock-clientID-02 Customer:mock-topic TopicType:mock-topic-type Nels:2.1.1 Networks:2.1.1.1/32}
自定义解码器 md:{Keys:[id create_at update_at is_delete version ecid customer topicType nels networks] Unused:[] Unset:[]}
自定义解码器 clients:{Id:2 CreateAt:2024-11-29 15:55:28 +0000 UTC UpdateAt:2024-11-29 15:55:28 +0000 UTC IsDelete:0 Version:0 Ecid:mock-clientID-02 Customer:mock-topic TopicType:mock-topic-type Nels:2.1.1 Networks:2.1.1.1/32}

OnPosSynced binlog.000170 6782

[2024/12/04 14:39:10] [info] dump.go:187 dump MySQL and parse OK, use 0.13 seconds, start binlog replication at (binlog.000170, 6782)
[2024/12/04 14:39:10] [info] binlogsyncer.go:442 begin to sync binlog from position (binlog.000170, 6782)
[2024/12/04 14:39:10] [info] binlogsyncer.go:408 Connected to mysql 9.0.1 server
[2024/12/04 14:39:10] [info] sync.go:22 start sync binlog at binlog file (binlog.000170, 6782)
[2024/12/04 14:39:10] [info] binlogsyncer.go:869 rotate to (binlog.000170, 6782)
[2024/12/04 14:39:10] [info] sync.go:59 received fake rotate event, next log name is binlog.000170
*/
