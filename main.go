package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/shankusu2017/url"
	"math/rand"
	"time"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	InitNodeMgr()

	r := gin.Default()

	r.GET("/v1/monitor", monitorGet)

	r.POST(fmt.Sprintf("%s", url.URL_REPEATER_SERVER), nodeRepeaterGet)
	r.POST(fmt.Sprintf("%s", url.URL_EVENT_POST), eventPost)

	// 监听并在 0.0.0.0:7080 上启动服务
	r.Run(fmt.Sprintf("%s:%d", "", url.PORT_NODEMGR)) // ":7080"
}
