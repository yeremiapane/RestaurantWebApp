package middlewares

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yeremiapane/restaurant-app/utils"
	"golang.org/x/time/rate"
)

// PaymentSecurityHeaders adds security headers for payment endpoints
func PaymentSecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Content-Security-Policy", "default-src 'self'")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		c.Next()
	}
}

// PaymentRateLimiter implements rate limiting for payment endpoints
func PaymentRateLimiter() gin.HandlerFunc {
	limiter := rate.NewLimiter(rate.Every(time.Second), 10)
	return func(c *gin.Context) {
		if !limiter.Allow() {
			c.JSON(429, gin.H{
				"error":   "Too many requests",
				"message": "Please wait before making another payment request",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// ValidatePaymentRequest validates payment request data
func ValidatePaymentRequest() gin.HandlerFunc {
	return func(c *gin.Context) {
		var request struct {
			Amount      float64 `json:"amount" binding:"required,gt=0"`
			OrderID     uint    `json:"order_id" binding:"required"`
			PaymentType string  `json:"payment_type" binding:"required,oneof=cash qris"`
		}

		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(400, gin.H{
				"error":   "Invalid request",
				"message": err.Error(),
			})
			c.Abort()
			return
		}

		// Validate amount format (max 2 decimal places)
		if request.Amount != float64(int64(request.Amount*100))/100 {
			c.JSON(400, gin.H{
				"error":   "Invalid amount",
				"message": "Amount must have at most 2 decimal places",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// LogPaymentRequest logs payment request details
func LogPaymentRequest() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		duration := time.Since(start)
		status := c.Writer.Status()

		// Log payment request details
		utils.InfoLogger.Printf(
			"Payment Request - Method: %s, Path: %s, Status: %d, Duration: %v",
			method, path, status, duration,
		)
	}
}
