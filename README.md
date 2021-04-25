# gin panic 拦截中间件

> 编程中有效的实践

## go 程序 panic 是什么❓

程序 panic 主要程序异常，程序不能继续向下推进时向调用者抛出的一个异常的机制，如果用户没有执行错误拦截，那么程序会一直往上抛出错误信息，
直到用户拦截，亦或是没有拦截会打印输出错误信息， 然后 `exit(2)` 退出程序

```golang
// go 程序崩溃之后，是怎么退出程序的？
func fatalpanic(msgs *_panic) {
	pc := getcallerpc() // 获取执行的函数
	sp := getcallersp() // 获取栈指针
	gp := getg()        // 获取当前执行的 g 的信息
	var docrash bool 
	systemstack(func() {
		if startpanic_m() && msgs != nil {
			atomic.Xadd(&runningPanicDefers, -1)
			printpanics(msgs)
		}
		docrash = dopanic_m(gp, pc, sp)
	})

	if docrash {
		crash()
	}

	systemstack(func() {
		exit(2)
	})

	*(*int)(nil) = 0 // not reached
}

```

## 程序什么情况下会触发异常？

一般情况下， 程序是不会有异常的，但是万事总会有个特殊的场景，比如: 

1. 访问了空指针导致的异常
2. 内存访问出错的异常(遇到过一个比较特殊的场景，fault address 相关的, 还和内存有关)
3. 访问了悬挂指针(特别是使用 unsafe 包的时候)
4. 数组越界
5. 写未初始化的 map
6. OOM(递归深度过大, 一次性读取大文件等)
7. 用户手动触发
8. 其他的场景


## 为什么需要错误拦截?

一般来说，程序编写完毕，经过测试之后， 一般会部署在其他的机器上，但是如果程序 crash 的话， 开发人员是无法在第一个时间捕捉到这个信息
再者，如果开发者知道了程序异常，但是无法复现在什么情况下，程序会异常。

## 怎么拦截异常？

在 go 程序中，异常的处理机制是 `defer + recovery` 模式， 如果是使用户手动抛出来的异常， 还可以使用 `panic` 关键字。一般来说函数调用
的最顶层使用 defer 就能捕捉到底层抛出来的异常。
问题： 如果底层抛出来多个异常怎么捕捉？
答：程序执行 panic 的时候， 是不会继续向下推进的，所以不会有两个异常抛出, 如果程序在 defer 中抛出异常， 那么底层抛出的异常是 defer 
函数中抛出的异常并且以最后一个抛出的异常为准, 比如：

> 例子中的异常以最后一个抛出的为准, 也就是 `panic("first")`

```go
package main 

import "fmt"

func dataPanic() {
	defer func() {
		panic("first")
	}()

	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
		}
		panic("second")
	}()

	defer func() {
		panic("three")
	}()

	panic("all")
}

```

## 程序异常时, 已经将信息信息发往何处?

1. 保存在日志文件里面？
2. 直接在控制台输出？

这两种都可以， 但是**为什么不把错误信息直接发送给开发者呢？**

## 需要将哪些信息抛出来？

一般来说，能把……程序定位到问题出在哪里了， 但是不一定会完整地复现出现这个问题的场景。 比如有一天， 运维人员告诉开发说:

> 以下内容纯属虚构， 如有雷同， 算我抄你：
>
> 你的程序出错了， 定位在 main.go 的第 28 行，报错原因是数组访问越界。 抓紧修复一下吧。。。。

对于这个问题， 如果只是一个数组参数可能还好解决， 但是对于多个参数， 处理逻辑比较复杂的场景下， 想要修复这个问题不是不行，而是复现这个问题的时间可能比修复这个问题的时间花的还要多得多？

。。。。

## 抛出来的息息需要屏蔽的数据

**用户数据比较敏感** 应该把用户的关键数据过滤

## gin 中怎么实现 recovery？

> recovery 有两个作用:
> 1.阻止程序崩溃
> 2.告诉用户, 程序运行到这个地方可能会 crash

gin 中的 `HandlerFunc` 基于函数式回调链的方式来实现的, 用户可以依据需求添加或者删除 `HandlerFunc` 的实现, `.Next()` 函数的机制可以使得引擎在调用回调的时候可以获取到外层业务代码的数据, 在调用的底层使用。

比如 `gin.Default()` 就默认注册了两个回调链, 其中 `gin.Logger()` 是用于记录访问日志记录的, `gin.Recovery()` 用来恢复并输出程序异常 panic 的日志的。但是这个日志, 默认是输出在控制台的, 并且无法做重放处理

> 以下代码来源于 `github.com/gin-gonic/gin`

```golang

type RecoveryFunc func(c *Context, err interface{})

func RecoveryWithWriter(out io.Writer, recovery ...RecoveryFunc) HandlerFunc {
	if len(recovery) > 0 {
		return CustomRecoveryWithWriter(out, recovery[0])
	}
	return CustomRecoveryWithWriter(out, defaultHandleRecovery)
}

```

## 对于多机部署环境, 该怎么处理这些日志呢？

是的， 多机部署，如果抛出来的日志中没有报告指明是哪台机器上的哪个程序出错, 那该怎么处理呢？ 确实， 如果不告诉是哪个程序出错，特别是单程序多进程的方式的话， 确实不太好搞。 因为需要确定是从哪台机器上哪个进程，出现了什么问题， 什么情况下会出现这个问题？

## 通知方式不能只支持一种， 而且还要支持后期的拓展

是的， 在针对需求剧烈抖动的情况下， 确实是需要把握住变与不变的场景。对于不变的数据可以硬编码。

## 如何实现上述的需求？

> 以下经验纯属虚构，如有雷同，算我抄你
>
> 所有可能会变动的地方，以后大几率都会变
>
> 所有肯产生错误的地方，以后大概率会产生错误
>
> 所有可能会引起崩溃的地方，以后大几率会崩溃

首先需要构造一个配置类:

```golang

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

```

构造函数需要可选参数

```golang

// 定义构造 Inject 类型
type InjectOption func(j *Inject)


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

```

业务变动相关的做成接口的方式, 方便后期的拓展

```golang

type Hook interface {
	Fire(ctx context.Context, entry *Entry) error
}

```

## 总结

1. 大体上描述了go 程序崩溃, 以及拦截 panic 的方式
2. 介绍了 go 程序崩溃的场景
3. 描述了 gin 回调的过程
4. 快速复现以及定位一个程序panic 最有效的方式(什么时候，哪些数据，在哪台机器，哪个程序，哪个版本的操作系统，cpu 架构，go 版本，会引起程序的哪部分panic)
5. 构造一个通知场景多变的 gin Recovery Handler 需要怎么设计

具体地实现需要参考 [github](https://github.com/guzzsek/agave/tree/master/ject)

谢谢
