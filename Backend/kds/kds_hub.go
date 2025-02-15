package kds

import (
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/yeremiapane/restaurant-app/models"
)

// KDSHub menampung semua client KDS (chef) dan channel untuk broadcast
type KDSHub struct {
	clients map[*websocket.Conn]bool
	mutex   sync.Mutex
}

var kdsHub = KDSHub{
	clients: make(map[*websocket.Conn]bool),
}

// RegisterClient -> menambahkan connection ke set
func RegisterClient(conn *websocket.Conn) {
	kdsHub.mutex.Lock()
	defer kdsHub.mutex.Unlock()
	kdsHub.clients[conn] = true
}

// UnregisterClient -> melepaskan connection
func UnregisterClient(conn *websocket.Conn) {
	kdsHub.mutex.Lock()
	defer kdsHub.mutex.Unlock()
	delete(kdsHub.clients, conn)
	conn.Close()
}

// BroadcastOrderUpdate -> menyiarkan update order ke semua client
func BroadcastOrderUpdate(order models.Order) {
	kdsHub.mutex.Lock()
	defer kdsHub.mutex.Unlock()

	data, _ := json.Marshal(struct {
		Event string       `json:"event"`
		Order models.Order `json:"order"`
	}{
		Event: "order_update",
		Order: order,
	})

	// kirim ke semua client
	for conn := range kdsHub.clients {
		conn.WriteMessage(websocket.TextMessage, data)
	}
}
