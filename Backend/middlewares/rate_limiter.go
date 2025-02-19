package middlewares

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type RateLimiter struct {
	rate     int
	interval time.Duration
	ips      map[string][]time.Time
	mu       sync.Mutex
}

func NewRateLimiter(rate int, interval int) *RateLimiter {
	return &RateLimiter{
		rate:     rate,
		interval: time.Duration(interval) * time.Second,
		ips:      make(map[string][]time.Time),
	}
}

func NewStrictRateLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Lebih ketat untuk endpoint login/register
		limiter := rate.NewLimiter(rate.Every(1*time.Minute), 5) // 5 requests per menit

		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Terlalu banyak percobaan, silakan tunggu beberapa saat",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

func (rl *RateLimiter) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		rl.mu.Lock()
		defer rl.mu.Unlock()

		now := time.Now()
		if _, exists := rl.ips[ip]; !exists {
			rl.ips[ip] = []time.Time{now}
			c.Next()
			return
		}

		requests := rl.ips[ip]
		cutoff := now.Add(-rl.interval)
		valid := make([]time.Time, 0)

		for _, t := range requests {
			if t.After(cutoff) {
				valid = append(valid, t)
			}
		}

		if len(valid) >= rl.rate {
			c.AbortWithStatus(http.StatusTooManyRequests)
			return
		}

		rl.ips[ip] = append(valid, now)
		c.Next()
	}
}
