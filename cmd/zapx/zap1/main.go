package main

import (
	"os"

	"github.com/jursonmo/practise_new/cmd/zapx/zap1/pkg/log"
	"github.com/jursonmo/practise_new/cmd/zapx/zap1/pkg/pkg1"
	"go.uber.org/zap"
)

func main() {
	file, err := os.OpenFile("./demo1.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}

	logger := log.New(file, log.InfoLevel)
	log.ResetDefault(logger)
	defer log.Sync()
	log.Info("demo1:", log.String("app", "start ok"),
		log.Int("major version", 2))
	pkg1.Foo()

	//----
	log.Debug("test debug", log.String("tester", "mjw")) //打印不出来
	log.Warn("test warn", log.String("tester", "mjw"))
	log.Error("test error level", log.String("tester", "mjw"))

	//动态修改日志级别,使得可以打印debug级别的日志：
	var atomicLevel = zap.NewAtomicLevel() //默认是info 级别
	logger = log.NewWithLevelEnabler(file, atomicLevel)
	log.ResetDefault(logger)
	log.Debug("test debug", log.String("tester", "mjw")) //现在是info 级别，所以打印不出来
	atomicLevel.SetLevel(zap.DebugLevel)
	log.Debug("test modify log level", log.String("modify", "ok"))
}
