package main

import (
	"os"

	"github.com/jursonmo/practise_new/cmd/zapx/zap2/pkg/log"
)

// 测试不同级别的日志打印到不同的日志文件里
func main() {
	file1, err := os.OpenFile("./access.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	file2, err := os.OpenFile("./warn.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	file3, err := os.OpenFile("./error.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}

	var tops = []log.TeeOption{
		{
			W: file1,
			Lef: func(lvl log.Level) bool {
				return lvl <= log.InfoLevel
			},
		},
		{
			W: file2,
			Lef: func(lvl log.Level) bool {
				return lvl > log.InfoLevel && lvl <= log.WarnLevel
			},
		},
		{
			W: file3,
			Lef: func(lvl log.Level) bool {
				return lvl > log.WarnLevel && lvl <= log.ErrorLevel
			},
		},
	}

	logger := log.NewTee(tops)
	log.ResetDefault(logger)

	log.Info("demo3:", log.String("app", "start ok"),
		log.Int("major version", 3))

	log.Warn("demo3:", log.String("app", "warn"),
		log.Int("reason", -1))

	log.Error("demo3:", log.String("app", "crash"),
		log.Int("reason", -1))

}
