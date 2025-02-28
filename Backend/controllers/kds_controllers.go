package controllers

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/yeremiapane/restaurant-app/kds" // folder berisi kdsHub
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		return origin == "http://127.0.0.1:5500"
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// KDSHandler -> endpoint WebSocket
func KDSHandler(c *gin.Context) {
	roleInterface, exists := c.Get("role")
	if !exists {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	role := roleInterface.(string)

	// Validasi role sudah dilakukan di middleware
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}
	defer ws.Close()

	// Register client dengan role
	kds.RegisterClient(ws, role)
	defer kds.UnregisterClient(ws)

	// Keep-alive dengan ping/pong
	ws.SetPingHandler(func(string) error {
		return ws.WriteControl(websocket.PongMessage, []byte{}, time.Now().Add(time.Second))
	})

	// Message loop
	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error reading message: %v", err)
			}
			break
		}
	}
}
