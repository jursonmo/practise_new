package pkg1

import "github.com/jursonmo/practise_new/cmd/zapx/zap1/pkg/log"

func Foo() {
	log.Info("call foo", log.String("url", "https://tonybai.com"),
		log.Int("attempt", 3))
}
