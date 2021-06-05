package pencil

import (
	"context"
	"fmt"
	"github.com/guzzsek/agave/pencil/config"
	"time"
)

func XLog() func() {
	var newLog = NewCore(&config.Config{Path: "logs"})
	return func() {
		newLog.WithContext(context.Background()).Infof("time:%s, greeter:%v", time.Now().String(), "jacy")
		newLog.WithContext(context.Background()).Warn("warn", "hello world")
		newLog.WithContext(context.Background()).Error("error log")
	}
}

func XLogMinute(m time.Duration) func() {
	var maxSize = uint32(32)
	var newLog = NewCore(&config.Config{Path: "logs", Level: "debug", MaxSize: &maxSize})

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
