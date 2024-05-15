package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/shankusu2017/url"
	"github.com/shankusu2017/utils"
	"math/rand"
	"time"
)

func main() {
	rand.NewSource(time.Now().UnixNano())
	utils.InitPac("./etc/cnIP.cfg")
	NodeMgrInit("./etc/nodeInfo.db")

	r := gin.Default()

	r.GET("/v1/monitor", MonitorGet)

	r.POST(fmt.Sprintf("%s", url.URL_REPEATER_SERVER), NodeRepeaterGet)
	r.POST(fmt.Sprintf("%s", url.URL_EVENT_POST), EventPost)

	// 监听并在 0.0.0.0:7080 上启动服务
	r.Run(fmt.Sprintf("%s:%d", "", url.PORT_NODEMGR)) // ":7080"
}
