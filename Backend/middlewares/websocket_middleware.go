package middlewares

import (
	"github.com/gin-gonic/gin"
	"github.com/yeremiapane/restaurant-app/utils"
)

func WebSocketAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Query("token")
		if token == "" {
			c.AbortWithStatus(401)
			return
		}

		// Validasi token
		claims, err := utils.ValidateToken(token)
		if err != nil {
			c.AbortWithStatus(401)
			return
		}

		// Set role dan user_id ke context
		c.Set("role", claims.Role)
		c.Set("user_id", claims.UserID)

		c.Next()
	}
}
