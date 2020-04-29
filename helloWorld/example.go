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
	}).GET("/pingping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pongpong",
		})
	})
	r.Group("/v1")
	r.Run()
}
