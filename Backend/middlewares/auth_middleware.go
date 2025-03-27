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

		// Debug log
		log.Printf("Auth Header: %v", authHeader)

		if authHeader == "" {
			utils.RespondJSON(c, http.StatusUnauthorized, "No authorization header", gin.H{
				"status": false,
				"error":  "No authorization header",
			})
			c.Abort()
			return
		}

		// Extract the token
		tokenString := strings.Replace(authHeader, "Bearer ", "", 1)

		// Debug log
		log.Printf("Token: %v", tokenString)

		// Validate the token
		claims, err := utils.ValidateToken(tokenString)
		if err != nil {
			utils.RespondJSON(c, http.StatusUnauthorized, "Invalid token", gin.H{
				"status": false,
				"error":  err.Error(),
			})
			c.Abort()
			return
		}

		// Set user info to context
		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)

		c.Next()
	}
}

func EnhancedAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		utils.InfoLogger.Printf("Request path: %s", c.Request.URL.Path)

		token := c.GetHeader("Authorization")
		if token == "" {
			token = c.Query("token")
		}

		utils.InfoLogger.Printf("Token from request: %s", token)

		if token == "" {
			utils.ErrorLogger.Printf("No token found in request")
			utils.RespondError(c, http.StatusUnauthorized, errors.New("token tidak ditemukan"))
			c.Abort()
			return
		}

		// Validasi format token
		if !strings.HasPrefix(token, "Bearer ") {
			utils.ErrorLogger.Printf("Invalid token format: %s", token)
			utils.RespondError(c, http.StatusUnauthorized, errors.New("format token tidak valid"))
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(token, "Bearer ")
		claims, err := utils.ValidateToken(tokenString)
		if err != nil {
			utils.ErrorLogger.Printf("Token validation failed: %v", err)
			utils.RespondError(c, http.StatusUnauthorized, err)
			c.Abort()
			return
		}

		// Tambahkan validasi tambahan
		if time.Now().Unix() > claims.ExpiresAt {
			utils.ErrorLogger.Printf("Token expired for user %d", claims.UserID)
			utils.RespondError(c, http.StatusUnauthorized, errors.New("token kadaluarsa"))
			c.Abort()
			return
		}

		// Cek apakah token ada di daftar blacklist
		if utils.IsTokenBlacklisted(tokenString) {
			utils.ErrorLogger.Printf("Blacklisted token used for user %d", claims.UserID)
			utils.RespondError(c, http.StatusUnauthorized, errors.New("token tidak valid"))
			c.Abort()
			return
		}

		utils.InfoLogger.Printf("Authenticated user %d with role %s", claims.UserID, claims.Role)
		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)
		c.Next()
	}
}
