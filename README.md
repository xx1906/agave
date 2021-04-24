# go 编码工具箱

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
2. 内存访问出错访问的异常(遇到过一个比较特殊的场景，fault address 相关的, 还和内存有关)
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

这两种都可以， 但是为什么不把错误信息直接发送给开发者呢？ 
