package controllers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/yeremiapane/restaurant-app/kds" // folder berisi kdsHub
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Accept all origins for ease of development and when using ngrok
		log.Printf("WebSocket connection attempted from origin: %s", r.Header.Get("Origin"))
		return true
		// Atau jika ingin lebih aman, bisa uncomment dan sesuaikan:
		// origin := r.Header.Get("Origin")
		// return origin == "http://127.0.0.1:5500" ||
		//        strings.Contains(origin, "ngrok-free.app") ||
		//        strings.Contains(origin, "localhost")
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// KDSHandler -> endpoint WebSocket
func KDSHandler(c *gin.Context) {
	roleInterface, exists := c.Get("role")
	if !exists {
		log.Printf("Unauthorized WebSocket connection attempt: no role found in context")
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	role := roleInterface.(string)
	log.Printf("WebSocket connection authorized for role: %s", role)

	// Log request headers for debugging
	for name, values := range c.Request.Header {
		for _, value := range values {
			log.Printf("Header: %s: %s", name, value)
		}
	}

	// Validasi role sudah dilakukan di middleware
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade WebSocket connection: %v", err)
		return
	}
	defer ws.Close()

	// Log successful connection
	log.Printf("WebSocket connection established for role: %s", role)

	// Register client dengan role
	kds.RegisterClient(ws, role)
	defer kds.UnregisterClient(ws)

	// Keep-alive dengan ping/pong
	ws.SetPingHandler(func(string) error {
		log.Printf("Ping received from client with role: %s", role)
		return ws.WriteControl(websocket.PongMessage, []byte{}, time.Now().Add(time.Second))
	})

	// Kirim pesan konfirmasi koneksi ke client
	connectionMsg := kds.Message{
		Event: "connection_established",
		Data: map[string]interface{}{
			"message": "WebSocket connection established successfully",
			"role":    role,
			"time":    time.Now().Format(time.RFC3339),
		},
	}

	if msgData, err := json.Marshal(connectionMsg); err == nil {
		if err := ws.WriteMessage(websocket.TextMessage, msgData); err != nil {
			log.Printf("Error sending connection confirmation: %v", err)
		} else {
			log.Printf("Connection confirmation sent to client with role: %s", role)
		}
	}

	// Setup heartbeat ticker untuk memastikan koneksi tetap aktif
	heartbeatTicker := time.NewTicker(30 * time.Second)
	defer heartbeatTicker.Stop()

	// Gunakan channel untuk menangani pesan dan heartbeat secara bersamaan
	done := make(chan struct{})

	// Goroutine untuk menangani heartbeat
	go func() {
		for {
			select {
			case <-heartbeatTicker.C:
				heartbeatMsg := kds.Message{
					Event: "heartbeat",
					Data: map[string]interface{}{
						"timestamp": time.Now().Unix(),
					},
				}

				if msgData, err := json.Marshal(heartbeatMsg); err == nil {
					if err := ws.WriteMessage(websocket.TextMessage, msgData); err != nil {
						log.Printf("Error sending heartbeat: %v", err)
						close(done)
						return
					}
					log.Printf("Heartbeat sent to client with role: %s", role)
				}
			case <-done:
				return
			}
		}
	}()

	// Message loop
	for {
		messageType, message, err := ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Error reading WebSocket message: %v", err)
			} else {
				log.Printf("WebSocket connection closed: %v", err)
			}
			close(done)
			break
		}

		log.Printf("Received WebSocket message from %s: %s", role, string(message))

		// Tangani pesan dari client
		if messageType == websocket.TextMessage {
			// Parse pesan dan kirim respon yang sesuai
			var clientMsg map[string]interface{}
			if err := json.Unmarshal(message, &clientMsg); err == nil {
				// Jika pesan heartbeat, balas dengan status koneksi
				if event, ok := clientMsg["event"].(string); ok && event == "heartbeat" {
					heartbeatResponse := kds.Message{
						Event: "heartbeat_response",
						Data: map[string]interface{}{
							"connected": true,
							"timestamp": time.Now().Unix(),
						},
					}

					if respData, err := json.Marshal(heartbeatResponse); err == nil {
						if err := ws.WriteMessage(websocket.TextMessage, respData); err != nil {
							log.Printf("Error sending heartbeat response: %v", err)
						}
					}
				} else {
					// Echo pesan kembali untuk debugging
					if err := ws.WriteMessage(messageType, message); err != nil {
						log.Printf("Error echoing message: %v", err)
						break
					}
				}
			} else {
				// Echo pesan kembali jika format tidak valid
				if err := ws.WriteMessage(messageType, message); err != nil {
					log.Printf("Error echoing message: %v", err)
					break
				}
			}
		}
	}

	log.Printf("WebSocket connection handler completed for role: %s", role)
}
