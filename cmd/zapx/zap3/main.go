package main

import (
	"github.com/jursonmo/practise_new/cmd/zapx/zap3/pkg/log"
	"go.uber.org/zap"
)

// 在zap2的基础上增加日志切割轮转
func main() {
	accessLogOpt := log.TeeOption{
		Filename: "access.log",
		Ropt: log.RotateOptions{
			MaxSize:    1,
			MaxAge:     1,
			MaxBackups: 3,
			Compress:   false,
		},
		Lef: func(lvl log.Level) bool {
			return lvl <= log.InfoLevel
		},
	}
	errorLogOpt := log.TeeOption{
		Filename: "error.log",
		Ropt: log.RotateOptions{
			MaxSize:    1,
			MaxAge:     1,
			MaxBackups: 3,
			Compress:   false,
		},
		Lef: func(lvl log.Level) bool {
			return lvl >= log.ErrorLevel
		},
	}
	var tops = []*log.TeeOption{
		&accessLogOpt,
		&errorLogOpt,
	}

	logger := log.NewTeeWithRotate(tops, zap.AddCaller(), zap.AddCallerSkip(1))
	log.ResetDefault(logger)

	//for i := 0; i < 2000; i++ {
	for i := 0; i < 1; i++ {
		log.Info("demo3:", log.String("app", "start ok"),
			log.Int("major version", 3))
		log.Error("demo3:", log.String("app", "crash"),
			log.Int("reason", -1))
	}
	//不同的基本打印到不同日志文件里，那么就没有动态设置日志级别的需求，关闭某个文件的打印就是关闭某个日志级别。
	//那么如何实时关闭access 日志文件的打印
	accessLogOpt.Lef = func(lvl log.Level) bool { return false }
	log.Info("demo3:", log.String("if you see this message", "something wrong happend"),
		log.Int("major version", 4))
}
