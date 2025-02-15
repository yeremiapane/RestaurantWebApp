package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/yeremiapane/restaurant-app/kds" // folder berisi kdsHub
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Set sesuai kebutuhan, mis. izinkan semua origin:
		return true
	},
}

// KDSHandler -> endpoint WebSocket
func KDSHandler(c *gin.Context) {
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	// Register connection
	kds.RegisterClient(ws)

	// Baca pesan (jika perlu), atau tunggu hingga client tutup
	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			break
		}
	}

	// Client disconnected
	kds.UnregisterClient(ws)
}
