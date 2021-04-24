package main

import (
	"github.com/gin-gonic/gin"
	"github.com/guzzsek/agave/box"
	"github.com/guzzsek/agave/ject"
)

func main() {
	engine := gin.New()

	// 构造拦截器配置
	inject := ject.NewInject(ject.SetThrowPanic(false))

	// 这里填写自己申请 webhook
	const webHook = "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=99433****"
	// 加入回调钩子
	inject.AddHook(box.NewWechatMarkdownWebHook(webHook))

	engine.Use(gin.Logger())
	// 使用中间件
	engine.Use(ject.RecoveryHandlerFunc(inject))

	engine.GET("/index", func(c *gin.Context) {
		c.JSON(200, gin.H{"code": 200})
	})

	engine.GET("/index/pnc", func(c *gin.Context) {
		var ptr *int
		*ptr = 8086
	})

	engine.GET("/out/of/bound", func(c *gin.Context) {
		const outOfBound = 233
		var data = make([]int, outOfBound, 256)
		data[outOfBound] = outOfBound
	})

	engine.GET("/write/invalid/map", func(c *gin.Context) {
		var m map[string]interface{}
		m["write_invalid_map"] = "panic"
	})

	engine.Run(":8080")
}
