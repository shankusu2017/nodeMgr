package main

import "github.com/gin-gonic/gin"

func monitorGet(c *gin.Context) {
	nodeLst := NodeGetAll()

	c.JSON(200, nodeLst)
}
