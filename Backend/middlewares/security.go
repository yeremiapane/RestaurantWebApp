package middlewares

import (
	"github.com/gin-gonic/gin"
)

func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Security headers
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-XSS-Protection", "1; mode=block")

		// Izinkan operasi WebSocket dan permintaan dari ngrok
		c.Header("Content-Security-Policy", "default-src 'self' 'unsafe-inline' 'unsafe-eval' https://*.ngrok-free.app; img-src 'self' data: https:; connect-src 'self' https://*.ngrok-free.app wss://*.ngrok-free.app; frame-ancestors 'self' https://*.ngrok-free.app;")

		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		c.Next()
	}
}
