package middlewares

import "github.com/gin-gonic/gin"

func CORSMiddlewares() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "http://127.0.0.1:5500")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, Sec-WebSocket-Protocol, Sec-WebSocket-Version, Sec-WebSocket-Key, Upgrade")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, PATCH, DELETE")

		if c.Request.Method == "OPTIONS" {
			if c.GetHeader("Upgrade") == "websocket" {
				c.Writer.Header().Set("Connection", "Upgrade")
				c.Writer.Header().Set("Upgrade", "websocket")
			}
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
