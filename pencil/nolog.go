package pencil

import (
	"context"
	"fmt"
	"github.com/guzzsek/agave/pencil/config"
	"time"
)

func XLog() func() {
	var rootCore = NewCore(&config.Config{Path: "logs"})
	var newLog = rootCore.WithContext(context.TODO())
	return func() {
		newLog.WithContext(context.Background()).Infof("time:%s, greeter:%v", time.Now().String(), "jacy")
		newLog.WithContext(context.Background()).Warn("warn", "hello world")
		newLog.WithContext(context.Background()).Error("error log")
	}
}

func XLogMinute(m time.Duration) func() {
	var maxSize = uint32(32)

	var rootCore = NewCore(&config.Config{Path: "logs", Level: "debug", MaxSize: &maxSize})
	var newLog = rootCore.WithContext(context.TODO())
	return func() {
		ticker := time.NewTicker(m)
		step := time.NewTicker(m / 100)
		for {
			select {
			case _ = <-step.C:
				fmt.Print(">")
			case _ = <-ticker.C:

				goto end
			default:
				newLog.WithContext(context.Background()).Infof("time:%s, greeter:%v", time.Now().String(), "jacy")
				newLog.WithContext(context.Background()).Warn("warn", "hello world")
				newLog.WithContext(context.Background()).Error("error log")
			}

		}
	end:
		fmt.Println()
	}
}
