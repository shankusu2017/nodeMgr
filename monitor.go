package main

import (
	"github.com/gin-gonic/gin"
)

func MonitorGet(c *gin.Context) {
	nodeLst := NodeGetAll()

	c.JSON(200, nodeLst)
}
