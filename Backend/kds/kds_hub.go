package kds

import (
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/yeremiapane/restaurant-app/models"
)

// Event types
const (
	EventOrderUpdate    = "order_update"
	EventKitchenUpdate  = "kitchen_update"
	EventTableUpdate    = "table_update"
	EventStaffNotif     = "staff_notification"
	EventPaymentUpdate  = "payment_update"
	EventPaymentPending = "payment_pending"
	EventPaymentSuccess = "payment_success"
	EventReceiptUpdate  = "receipt_generated"
)

type Message struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
}

// KDSHub menampung semua client KDS (chef, staff, admin) dan channel untuk broadcast
type KDSHub struct {
	clients map[*websocket.Conn]string // conn -> role
	mutex   sync.Mutex
}

var kdsHub = KDSHub{
	clients: make(map[*websocket.Conn]string),
}

// RegisterClient -> menambahkan connection ke set dengan role
func RegisterClient(conn *websocket.Conn, role string) {
	kdsHub.mutex.Lock()
	defer kdsHub.mutex.Unlock()
	kdsHub.clients[conn] = role
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
	broadcast(Message{
		Event: EventOrderUpdate,
		Data:  order,
	})
}

// BroadcastKitchenUpdate -> update untuk chef
func BroadcastKitchenUpdate(data interface{}) {
	broadcast(Message{
		Event: EventKitchenUpdate,
		Data:  data,
	})
}

// BroadcastTableUpdate -> update status meja
func BroadcastTableUpdate(table models.Table) {
	broadcast(Message{
		Event: EventTableUpdate,
		Data:  table,
	})
}

// BroadcastStaffNotification -> notifikasi untuk staff
func BroadcastStaffNotification(message string) {
	broadcast(Message{
		Event: EventStaffNotif,
		Data:  message,
	})
}

// BroadcastPaymentUpdate -> update status pembayaran
func BroadcastPaymentUpdate(payment models.Payment, order models.Order) {
	broadcast(Message{
		Event: EventPaymentUpdate,
		Data: map[string]interface{}{
			"payment": payment,
			"order":   order,
		},
	})
}

// BroadcastPaymentPending -> notifikasi pembayaran pending
func BroadcastPaymentPending(payment models.Payment) {
	broadcast(Message{
		Event: EventPaymentPending,
		Data:  payment,
	})
}

// BroadcastPaymentSuccess -> notifikasi pembayaran berhasil
func BroadcastPaymentSuccess(payment models.Payment) {
	broadcast(Message{
		Event: EventPaymentSuccess,
		Data:  payment,
	})
}

// Broadcast ReceiptGenerated -> notifikasi struk dibuat
func BroadcastGenerated(receipt models.Receipt) {
	broadcast(Message{
		Event: EventReceiptUpdate,
		Data:  receipt,
	})
}

// BroadcastMessage -> broadcast pesan umum
func BroadcastMessage(msg Message) {
	broadcast(msg)
}

// broadcast -> fungsi internal untuk mengirim pesan
func broadcast(msg Message) {
	kdsHub.mutex.Lock()
	defer kdsHub.mutex.Unlock()

	data, _ := json.Marshal(msg)

	for conn, role := range kdsHub.clients {
		// Filter berdasarkan role jika diperlukan
		switch msg.Event {
		case EventKitchenUpdate:
			if role != "chef" && role != "admin" {
				continue
			}
		case EventStaffNotif:
			if role != "staff" && role != "admin" {
				continue
			}
		}

		conn.WriteMessage(websocket.TextMessage, data)
	}
}
