package middlewares

import (
	"github.com/gin-gonic/gin"
	"github.com/yeremiapane/restaurant-app/utils"
	"time"
)

func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		if raw != "" {
			path = path + "?" + raw
		}

		utils.InfoLogger.Printf("%s | %3d | %13v | %15s | %s", c.Request.Method, status, latency, latency/1000, path)
	}
}
