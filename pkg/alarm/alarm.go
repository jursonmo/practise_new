package alarm

import "github.com/jursonmo/practise_new/pkg/alarm/dingding"

type Alarm struct {
	UseDingDing bool              `toml:"useDingDing"`
	DingDing    dingding.DingDing `toml:"dingding"`
}

func (a *Alarm) DingDingAlarm(msg string, isTest bool) error {
	return a.DingDing.SendMsg(msg, isTest)
}
