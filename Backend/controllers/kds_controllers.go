package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/yeremiapane/restaurant-app/kds" // folder berisi kdsHub
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Sesuaikan dengan kebutuhan keamanan
	},
}

// KDSHandler -> endpoint WebSocket
func KDSHandler(c *gin.Context) {
	// Ambil role dari token/auth
	roleInterface, exists := c.Get("role")
	if !exists {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	role := roleInterface.(string)

	// Validasi role
	if role != "chef" && role != "staff" && role != "admin" {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	// Register dengan role
	kds.RegisterClient(ws, role)

	// Baca pesan (jika perlu)
	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			break
		}
	}

	// Unregister saat disconnect
	kds.UnregisterClient(ws)
}
