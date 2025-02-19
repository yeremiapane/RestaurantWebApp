package middlewares

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yeremiapane/restaurant-app/utils"
)

func RoleCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.Param("role")
		userRole, exists := c.Get("role")

		if !exists {
			utils.RespondError(c, http.StatusUnauthorized, fmt.Errorf("unauthorized"))
			c.Abort()
			return
		}

		// Validasi role
		switch role {
		case "admin":
			if userRole != "admin" {
				utils.RespondError(c, http.StatusForbidden, fmt.Errorf("admin access required"))
				c.Abort()
				return
			}
		case "chef":
			if userRole != "chef" && userRole != "admin" {
				utils.RespondError(c, http.StatusForbidden, fmt.Errorf("chef access required"))
				c.Abort()
				return
			}
		case "staff":
			if userRole != "staff" && userRole != "admin" {
				utils.RespondError(c, http.StatusForbidden, fmt.Errorf("staff access required"))
				c.Abort()
				return
			}
		}

		c.Next()
	}
}
