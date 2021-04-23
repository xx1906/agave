// @desc:   gin 崩溃拦截器, 主要用于报告程序 panic 的具体信息
// @author: 小肥
// @date:   2021年04月23日
// @email:  <2356450144@qq.com>

package ject

import "context"

type Entry struct {
	Ctx            context.Context        `json:"-"`               // 上下文信息
	Cause          string                 `json:"cause"`           // 程序崩溃的原因
	CauseTime      string                 `json:"cause_time"`      // 程序崩溃的时间
	RequestContent string                 `json:"request_content"` // HTTP 请求的内容, 用于重放, 复现 panic 场景
	RequestID      string                 `json:"request_id"`      // 请求 ID
	RequestURI     string                 `json:"request_uri"`     // 请求路径
	Method         string                 `json:"method"`          // 请求方法
	HostName       string                 `json:"host_name"`       // 主机名, 多机部署时有用
	GOOS           string                 `json:"goos"`            // 系统
	GOARCH         string                 `json:"goarch"`          // 系统架构
	ServiceName    string                 `json:"service_name"`    // 服务名称
	GOVersion      string                 `json:"go_version"`      // golang 的版本信息
	Data           map[string]interface{} `json:"data"`            // 额外的信息
}
