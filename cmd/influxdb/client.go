package main

import (
	"context"
	"fmt"
	"log"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api/query"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
)

type MyData struct {
	Serial  string
	Iface   string
	RxBytes float64 `json:""`
	TxBytes float64
	Time    time.Time
}

var org = "test-org"
var bucket = "test-bucket"
var client influxdb2.Client

func init() {
	//token := os.Getenv("INFLUXDB_TOKEN")
	token := "RzlY2G37L9XmPhd_rWoFDra1gzC5ST8G3lVxCpVv0Gn73CQNgnnUzdS-iEOKxqHNWGIkMRW6-trzBBKLyvsajA=="
	url := "http://192.168.132.10:8086"
	client = influxdb2.NewClient(url, token)
}

func main() {
	writeAPI := client.WriteAPIBlocking(org, bucket)
	go func() {
		//每10秒写数据
		for {
			for i := 0; i < 2; i++ {
				tags := map[string]string{
					"iface":  "wan0",
					"serial": fmt.Sprintf("serial-%d", i),
				}
				fields := map[string]interface{}{
					"rxbytes": 800 + 100*i,
					"txbytes": 400 + 100*i,
				}
				//func NewPoint(measurement string,tags map[string]string,fields map[string]interface{},ts time.Time,)
				point := write.NewPoint("net_stat", tags, fields, time.Now())

				if err := writeAPI.WritePoint(context.Background(), point); err != nil {
					log.Fatal(err)
				}
			}
			time.Sleep(10 * time.Second)
		}
	}()

	time.Sleep(time.Second)
	go func() {
		//每分钟读一次test-bucket 最近一分钟的数据，聚合查询一下最近1分钟数据的平均值，
		//由于已经创建一个task 任务，每分钟把最近一分钟的数据聚合平均后，写入test-bucket-1d, 所以这里也查询test-bucket-1d 最近一分钟的数据。
		for {
			fmt.Println("-------test-bucket Query-----")
			queryAPI := client.QueryAPI(org)
			query := `from(bucket: "test-bucket")
            |> range(start: -1m)
            |> filter(fn: (r) => r._measurement == "net_stat")`
			results, err := queryAPI.Query(context.Background(), query)
			if err != nil {
				log.Fatal(err)
			}

			for results.Next() {
				fmt.Println(results.Record())
			}
			if err := results.Err(); err != nil {
				log.Fatal(err)
			}

			fmt.Println("-------Execute an Aggregate Query-----")

			query = `from(bucket: "test-bucket")
              |> range(start: -1m)
              |> filter(fn: (r) => r._measurement == "net_stat")
              |> mean()`
			results, err = queryAPI.Query(context.Background(), query)
			if err != nil {
				log.Fatal(err)
			}
			for results.Next() {
				record := results.Record()
				fmt.Println(record)
				//Transfer(record, record.Values()) //聚合查询没有时间点_time
			}
			if err := results.Err(); err != nil {
				log.Fatal(err)
			}

			fmt.Println("-------test-bucket-1d net_stat Query-----")
			queryAPI = client.QueryAPI(org)
			query = `from(bucket: "test-bucket-1d")
            |> range(start: -1m)
            |> filter(fn: (r) => r._measurement == "net_stat")`
			results, err = queryAPI.Query(context.Background(), query)
			if err != nil {
				log.Fatal(err)
			}

			for results.Next() {
				record := results.Record()
				fmt.Println(record)

			}
			if err := results.Err(); err != nil {
				log.Fatal(err)
			}
			getTraffic(true, "serial-1", time.Second*10)
			fmt.Println("------------------------------------\n\n")
			time.Sleep(time.Minute)
		}
	}()

	for {
		time.Sleep(time.Hour)
	}

}

// Query data with Flux: https://docs.influxdata.com/influxdb/v2/query-data/flux/
func getTraffic(in bool, serial string, t time.Duration) {
	field := "txbytes"
	if in {
		field = "rxbytes"
	}
	fmt.Printf("-----------------query %s data from %v ago --------------\n", serial, t)
	queryAPI := client.QueryAPI(org)
	query := fmt.Sprintf(`from(bucket: "test-bucket")
	|> range(start: -1m)
	|> filter(fn: (r) => r._measurement == "net_stat" and r.serial == "%s")
	|> filter(fn: (r) => r._field == "%s")`, serial, field)
	fmt.Printf("query:%s\n", query)
	results, err := queryAPI.Query(context.Background(), query)
	if err != nil {
		log.Fatal(err)
	}

	for results.Next() {
		record := results.Record()
		fmt.Println(record)
		Transfer(record, record.Values())
	}

	if err := results.Err(); err != nil {
		log.Fatal(err)
	}
	//_start 和 _stop 是查询的跨度，_time 是这条记录的时间点， iface 和 serial是 tag
	//_field:rxbytes,_measurement:net_stat,_start:2024-08-20 09:28:42.520121153 +0000 UTC,_stop:2024-08-20 09:29:42.520121153 +0000 UTC,
	//_time:2024-08-20 09:29:41.521051 +0000 UTC,_value:900,iface:wan0,result:_result,serial:serial-1,table:0
}

func Transfer(r *query.FluxRecord, values map[string]any) MyData {
	fmt.Printf("record values:%+v\n", values)
	d := MyData{}
	d.Serial = r.ValueByKey("serial").(string)
	d.Iface = r.ValueByKey("iface").(string)
	switch r.Field() {
	case "rxbytes":
		d.RxBytes = r.Value().(float64)
	case "txbytes":
		d.TxBytes = r.Value().(float64)
	}

	d.Time = r.Time()
	fmt.Printf("MyData:%+v\n", d)
	return d
}
