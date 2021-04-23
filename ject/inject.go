// @desc:   gin 奔溃拦截器, 主要用于报告程序 panic 的具体信息
// @author: 小肥
// @date:   2021年04月23日
// @email:  <2356450144@qq.com>

package ject

import (
	"context"
	"net/http"
	"net/http/httputil"
	"os"
	"runtime"
	"sync"
	"time"
)

const (
	requestID   = "X-Trace-Id"
	serviceName = "https://github.com"
)

// 配置信息
type Inject struct {
	mu *sync.Mutex // 互斥锁

	GOOS        string `json:"goos"`         // 系统
	GOARCH      string `json:"goarch"`       // 架构信息
	HostName    string `json:"host_name"`    // 主机名
	ServiceName string `json:"service_name"` // 服务名
	GOVersion   string `json:"go_version"`   // golang 的版本信息

	TimeFormatter     func(t time.Time) string     `json:"-"` // 日期格式化
	PurgeRequest      func(s string) string        `json:"-"` // 清洗请求信息
	GetRequestID      func(r *http.Request) string `json:"-"` // 获取请求 ID
	GetRequestContent func(r *http.Request) string

	Hooks      []Hook `json:"-"` // 钩子函数
	ThrowPanic bool   // 是否继续向外抛出异常
}

// 定义构造 Inject 类型
type InjectOption func(j *Inject)

// 设置 方法名
func SetServiceName(srv string) InjectOption {
	return func(c *Inject) {
		c.ServiceName = srv
	}
}

// 设置格式化时间函数
func SetTimeFormatter(tf func(t time.Time) string) InjectOption {
	return func(c *Inject) {
		c.TimeFormatter = tf
	}
}

// 设置是否继续向外排除异常
func SetThrowPanic(tp bool) InjectOption {
	return func(c *Inject) {
		c.ThrowPanic = tp
	}
}

// 设置用户信息敏感内容处理函数
func SetPurgeRequest(f func(string) string) InjectOption {
	return func(c *Inject) {
		c.PurgeRequest = f
	}
}

// 获取请求追踪的 id
func SetGetRequestId(f func(r *http.Request) string) InjectOption {
	return func(c *Inject) {
		c.GetRequestID = f
	}
}

// 获取请求内容
func SetGetRequestContent(f func(r *http.Request) string) InjectOption {
	return func(c *Inject) {
		c.GetRequestContent = f
	}
}

func (c *Inject) AddHook(h Hook) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Hooks = append(c.Hooks, h)
}

// 构造 Inject 函数, opt 是可选的构造项
func NewInject(opt ...InjectOption) *Inject {
	cj := Inject{}
	hostname, _ := os.Hostname()

	cj.mu = &sync.Mutex{}
	cj.GOOS = runtime.GOOS
	cj.GOARCH = runtime.GOARCH
	cj.HostName = hostname
	cj.ServiceName = serviceName
	cj.GOVersion = runtime.Version()

	cj.TimeFormatter = defaultTimeFormatter
	cj.PurgeRequest = defaultPurgeRequest
	cj.GetRequestID = defaultGetRequestId
	cj.GetRequestContent = defaultGetRequestContent

	cj.Hooks = make([]Hook, 0, 4)
	cj.ThrowPanic = false

	for _, ijOpt := range opt {
		if ijOpt == nil {
			continue
		}

		ijOpt(&cj)
	}

	return &cj
}

func (c *Inject) NewEntry(ctx context.Context, r *http.Request, cause string) *Entry {

	return &Entry{
		Ctx:            ctx,
		CauseTime:      c.TimeFormatter(time.Now()),
		RequestID:      c.GetRequestID(r),
		RequestContent: c.PurgeRequest(c.GetRequestContent(r)),
		RequestURI:     r.RequestURI,
		Method:         r.Method,
		HostName:       c.HostName,
		GOOS:           c.GOOS,
		GOARCH:         c.GOARCH,
		ServiceName:    c.ServiceName,
		GOVersion:      c.GOVersion,
		Data:           make(map[string]interface{}, 4),
	}
}

var (
	_ = SetTimeFormatter
	_ = SetPurgeRequest
	_ = SetGetRequestId
	_ = NewInject
	_ = SetServiceName
	_ = SetThrowPanic
	_ = SetGetRequestContent
)

// 默认不过滤用户敏感信息
func defaultPurgeRequest(s string) string {
	return s
}

// 获取请求 id
func defaultGetRequestId(r *http.Request) string {
	if r == nil {
		return ""
	}
	return r.Header.Get(requestID)
}

// 格式化时间
func defaultTimeFormatter(t time.Time) string {
	timeString := t.Format("2006-01-02 15:04:05")
	return timeString
}

// 获取请求内容
func defaultGetRequestContent(r *http.Request) string {
	data, _ := httputil.DumpRequest(r, true)
	return string(data)
}
