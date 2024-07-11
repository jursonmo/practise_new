/*
	{
	    "serial": "f63d96c6b0394a69ac650f38d4350a2a",
	    "data": [
	        {
	            "type": "idsIpsCloud",
	            "reportTime": {{reportTime}},
	            "payload": [
	                {
	                    "srcIp": "1.1.1.1",
	                    "srcPort": 11,
	                    "dstIp": "2.2.2.2",
	                    "dstPort": 22,
	                    "protocol": "UDP",
	                    "code": 101,
	                    "description": "",
	                    "time": {{reportTime}},
	                    "severity": 1
	                },
	                {
	                    "srcIp": "11.11.11.11",
	                    "srcPort": 1111,
	                    "dstIp": "22.22.22.22",
	                    "dstPort": 2222,
	                    "protocol": "TCP",
	                    "code": 202,
	                    "description": "",
	                    "time": {{reportTime}},
	                    "severity": 1
	                }
	            ]
	        }
	    ]
	}
*/
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/nxadm/tail"
)

type ReportData struct {
	Start       int64  `json:"time"` //跟温水确认是毫秒
	SrcIp       string `json:"srcIp"`
	SrcPort     int    `json:"srcPort"`
	DstIp       string `json:"dstIp"`
	DstPort     int    `json:"dstPort"`
	Protocol    string `json:"protocol"`
	AppProto    string `json:"appProto"`
	Code        int    `json:"code"`        //gid
	Description string `json:"description"` //Signature,msg
	Severity    int    `json:"severity"`    //priority?
	Action      string `json:"action"`
}

// event 格式是针对：This is Suricata version 7.0.5 RELEASE
type EveEvent struct {
	//Timestamp time.Time `json:"timestamp"` //把日志里的"2024-04-27T12:51:38.870312+0800" 加冒号改成 "2024-04-27T12:51:38.870312+08:00"
	Timestamp string `json:"timestamp"`
	InIface   string `json:"in_iface"`

	EventType string `json:"event_type"`
	SrcIP     string `json:"src_ip"`
	DstIP     string `json:"dest_ip"`
	SrcPort   int    `json:"src_port"`
	DstPort   int    `json:"dest_port"`
	Proto     string `json:"proto"`
	AppProto  string `json:"app_proto,omitempty"`

	Alert *EveAlert `json:"alert,omitempty"`
	Flow  *EveFlow  `json:"flow,omitempty"`

	HTTP       *EveHTTP                `json:"http,omitempty"`
	DNS        *EveDNS                 `json:"dns,omitempty"`
	Stats      *EveStats               `json:"stats,omitempty"`
	Payload    string                  `json:"payload,omitempty"`
	PacketInfo *map[string]interface{} `json:"packet_info,omitempty"`
}

type EveAlert struct {
	Action      string `json:"action"`
	Gid         int    `json:"gid"`
	SignatureId int    `json:"signature_id"`
	Rev         int    `json:"rev"`
	Severity    int    `json:"severity"` //priority
	Category    string `json:"category"`
	Signature   string `json:"signature"` //msg
}

type EveFlow struct {
	//Start         time.Time `json:"start"`
	Start         string `json:"start"`
	SrcIP         string `json:"src_ip"`
	DstIP         string `json:"dest_ip"`
	PktsToserver  int    `json:"pkts_toserver"`
	PktsToclient  int    `json:"pkts_toclient"`
	BytesToserver int64  `json:"bytes_toserver"`
	BytesToclient int64  `json:"bytes_toclient"`
}

type EveHTTP struct {
	HTTPMethod string `json:"http_method"`
	Host       string `json:"host"`
	URL        string `json:"url"`
}

type EveDNS struct {
	DNSQuery string `json:"dns_query"`
	DNSRCODE string `json:"dns_rcode"`
}

type EveStats struct {
	HttpEvents   int `json:"http_events"`
	DnsEvents    int `json:"dns_events"`
	AlertEvents  int `json:"alert_events"`
	FlowEvents   int `json:"flow_events"`
	SshEvents    int `json:"ssh_events"`
	TlsEvents    int `json:"tls_events"`
	SmtpEvents   int `json:"smtp_events"`
	Dnp3Events   int `json:"dnp3_events"`
	ModbusEvents int `json:"modbus_events"`
}

func main() {
	// file, err := os.OpenFile("./eve.json", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println(file.Name())
	// defer file.Close()

	so := struct {
		StartAt time.Time
	}{time.Now()}
	d, _ := json.Marshal(so)
	fmt.Printf("%s\n", string(d))

	now := time.Now()
	fmt.Printf("now.UnixMilli():%d, time.Now().UTC().UnixMicro():%d\n", now.UnixMilli(), now.UTC().UnixMicro())
	//ReOpen 的作用是，删除文件后，会等待文件重新创建
	/*
		2024/05/05 23:52:25 Re-opening moved/deleted file ./eve.json ...
		2024/05/05 23:52:25 Waiting for ./eve.json to appear...
		2024/05/05 23:52:34 Successfully reopened ./eve.json
	*/
	t, err := tail.TailFile("./eve.json", tail.Config{Location: &tail.SeekInfo{Offset: 0, Whence: io.SeekEnd}, Follow: true, ReOpen: true})
	if err != nil {
		panic(err)
	}

	var eve EveEvent
	for line := range t.Lines {
		fmt.Printf("JSON: " + line.Text + "\n")

		err := json.Unmarshal([]byte(line.Text), &eve)
		if err != nil {
			panic(err)
		}
		if eve.EventType == "alert" {
			rd := eve.TransToReportData()
			fmt.Println(rd)
		}
		eve.Print()
	}
}

func (e *EveEvent) TransToReportData() ReportData {
	action := e.Alert.Action
	if action != "blocked" {
		action = "alert"
	}
	return ReportData{
		Start:       time.Now().UnixMilli(),
		SrcIp:       e.SrcIP,
		SrcPort:     e.SrcPort,
		DstIp:       e.DstIP,
		DstPort:     e.DstPort,
		Protocol:    e.Proto,
		AppProto:    e.AppProto,
		Code:        e.Alert.Gid,
		Description: e.Alert.Signature,
		Severity:    e.Alert.Severity,
		Action:      action,
	}
}

// 打印每个事件
func (event *EveEvent) Print() {
	fmt.Println("Timestamp:", event.Timestamp)
	fmt.Println("Event Type:", event.EventType)
	fmt.Println("Source IP:", event.SrcIP)
	fmt.Println("Destination IP:", event.DstIP)
	fmt.Println("Source Port:", event.SrcPort)
	fmt.Println("Destination Port:", event.DstPort)
	fmt.Println("Protocol:", event.Proto)

	if event.Alert != nil {
		fmt.Println("Alert Action:", event.Alert.Action)
		fmt.Println("Alert Gid:", event.Alert.Gid)
		fmt.Println("Alert SignatureId:", event.Alert.SignatureId)

		fmt.Println("Alert Action:", event.Alert.Action)
		fmt.Println("Alert Category:", event.Alert.Category)
		fmt.Println("Alert Signature:", event.Alert.Signature)
	}

	if event.HTTP != nil {
		fmt.Println("HTTP Method:", event.HTTP.HTTPMethod)
		fmt.Println("Host:", event.HTTP.Host)
		fmt.Println("URL:", event.HTTP.URL)
	}

	if event.DNS != nil {
		fmt.Println("DNS Query:", event.DNS.DNSQuery)
		fmt.Println("DNS RCODE:", event.DNS.DNSRCODE)
	}

	if event.Flow != nil {
		fmt.Println("Packets to Server:", event.Flow.PktsToserver)
		fmt.Println("Packets to Client:", event.Flow.PktsToclient)
	}

	if event.Stats != nil {
		fmt.Println("HTTP Events:", event.Stats.HttpEvents)
		fmt.Println("DNS Events:", event.Stats.DnsEvents)
		fmt.Println("Alert Events:", event.Stats.AlertEvents)
		fmt.Println("Flow Events:", event.Stats.FlowEvents)
		fmt.Println("SSH Events:", event.Stats.SshEvents)
		fmt.Println("TLS Events:", event.Stats.TlsEvents)
		fmt.Println("SMTP Events:", event.Stats.SmtpEvents)
		fmt.Println("DNP3 Events:", event.Stats.Dnp3Events)
		fmt.Println("Modbus Events:", event.Stats.ModbusEvents)
	}

	fmt.Println("Payload:", event.Payload)
	fmt.Println("Packet Info:", event.PacketInfo)
	fmt.Println("Application Protocol:", event.AppProto)

	fmt.Println("----------------------------------")

}
