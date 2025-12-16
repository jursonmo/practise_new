package main

import (
	"log"

	"github.com/jursonmo/practise_new/pkg/alarm"
	"github.com/spf13/viper"
)

func main() {
	viper.SetConfigName("alarm")
	viper.SetConfigType("toml")
	viper.AddConfigPath("../")
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("read config file failed: %v", err)
	}

	alarm := &alarm.Alarm{}
	err = viper.Unmarshal(alarm)
	if err != nil {
		log.Fatalf("unmarshal config failed: %v", err)
	}
	log.Printf("load alarm: %+v\n", alarm)

	err = alarm.DingDingAlarm("test message", true)
	if err != nil {
		log.Printf("send dingding alarm failed: %v", err)
	}

	err = alarm.DingDingAlarm("prod message", false)
	if err != nil {
		log.Printf("send dingding alarm failed: %v", err)
	}
}
