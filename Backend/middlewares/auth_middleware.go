package middlewares

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yeremiapane/restaurant-app/utils"
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

func EnhancedAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			token = c.Query("token")
		}

		if token == "" {
			utils.RespondError(c, http.StatusUnauthorized, errors.New("token tidak ditemukan"))
			c.Abort()
			return
		}

		// Validasi format token
		if !strings.HasPrefix(token, "Bearer ") {
			utils.RespondError(c, http.StatusUnauthorized, errors.New("format token tidak valid"))
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(token, "Bearer ")
		claims, err := utils.ValidateToken(tokenString)
		if err != nil {
			utils.RespondError(c, http.StatusUnauthorized, err)
			c.Abort()
			return
		}

		// Tambahkan validasi tambahan
		if time.Now().Unix() > claims.ExpiresAt {
			utils.RespondError(c, http.StatusUnauthorized, errors.New("token kadaluarsa"))
			c.Abort()
			return
		}

		// Cek apakah token ada di daftar blacklist
		if utils.IsTokenBlacklisted(tokenString) {
			utils.RespondError(c, http.StatusUnauthorized, errors.New("token tidak valid"))
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)
		c.Next()
	}
}
