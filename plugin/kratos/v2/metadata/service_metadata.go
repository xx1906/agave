package metadata

import (
	"os"
	"reflect"
	"runtime"
	"time"
)

func ServiceMetaData(data map[string]string) map[string]string {

	const (
		goos       = "__go_os__"       // 操作系统版本
		goVersion  = "__go_version__"  // golang 版本
		systemArch = "__system_arch__" // 系统架构
		hostName   = "__host_name__"   // 主机名

		startUP = "__start_up__" // 程序启动时间
	)

	// 如果传入无效参数
	if reflect.ValueOf(data).IsNil() {
		data = make(map[string]string, 32)
	}

	data[goos] = runtime.GOOS
	data[goVersion] = runtime.Version()
	data[systemArch] = runtime.GOARCH
	data[hostName], _ = os.Hostname()

	data[startUP] = timeFormat(time.Now())

	return data
}

func timeFormat(t time.Time) string {
	return t.Format("2006/01/02 15:04:05")
}
