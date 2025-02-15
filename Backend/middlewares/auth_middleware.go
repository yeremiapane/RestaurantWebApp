package middlewares

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/yeremiapane/restaurant-app/utils"
	"log"
	"net/http"
	"strings"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			utils.RespondError(c, http.StatusUnauthorized, errors.New("Authorization header missing"))
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := utils.ParseToken(tokenString)
		if err != nil || claims == nil {
			utils.RespondError(c, http.StatusUnauthorized, errors.New("Invalid or expired token"))
			c.Abort()
			return
		}

		if claims.UserID == 0 {
			utils.RespondError(c, http.StatusUnauthorized, errors.New("Invalid user ID in token"))
			c.Abort()
			return
		}

		log.Printf("Decoded Claims: %+v\n", claims)

		c.Set("userID", claims.UserID)
		c.Set("role", claims.Role)

		c.Next()
	}
}
