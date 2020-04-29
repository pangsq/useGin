package main

import (
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})
	r.GET("/p/*segs", nil)
	// r.GET("/ping/:seg/2", nil)
	r.GET("/ping/:seg", nil)
	r.Group("/v1").GET("/get")
	r.Run()
}
