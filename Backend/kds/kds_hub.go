package kds

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/yeremiapane/restaurant-app/models"
)

// Event types
const (
	EventOrderUpdate     = "order_update"
	EventKitchenUpdate   = "kitchen_update"
	EventTableUpdate     = "table_update"
	EventStaffNotif      = "staff_notification"
	EventPaymentUpdate   = "payment_update"
	EventPaymentPending  = "payment_pending"
	EventPaymentSuccess  = "payment_success"
	EventPaymentExpired  = "payment_expired"
	EventPaymentFailed   = "payment_failed"
	EventReceiptUpdate   = "receipt_generated"
	EventTableCreate     = "table_create"
	EventTableDelete     = "table_delete"
	EventDashboardUpdate = "dashboard_update"
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
	// Tambahkan informasi aksi jika belum ada
	orderData := map[string]interface{}{
		"id":         order.ID,
		"status":     order.Status,
		"created_at": order.CreatedAt,
		"updated_at": order.UpdatedAt,
		"action":     "update", // Default action
	}

	// Salin properti lain dari order
	// Preload OrderItems jika belum dimuat
	if len(order.OrderItems) > 0 {
		orderData["order_items"] = order.OrderItems
	}
	if order.TableID > 0 {
		orderData["table_id"] = order.TableID
	}
	if order.Table.ID > 0 {
		orderData["table"] = order.Table
	}
	if order.CustomerID > 0 {
		orderData["customer_id"] = order.CustomerID
	}
	if order.Customer.ID > 0 {
		orderData["customer"] = order.Customer
	}

	broadcast(Message{
		Event: EventOrderUpdate,
		Data:  orderData,
	})

	// Juga broadcast ke dashboard_update dengan data statistik pesanan
	dashboardData := map[string]interface{}{
		"order_stats": map[string]interface{}{
			"total_orders": 1, // Increment indicator
			"order_status": order.Status,
		},
		"updated_order": orderData,
		"data_type":     "order_update",
	}

	// Tambahkan informasi pendapatan jika order sudah dibayar/selesai
	if order.Status == "completed" || order.Status == "paid" {
		var totalAmount float64
		for _, item := range order.OrderItems {
			totalAmount += float64(item.Quantity) * item.Price
		}

		dashboardData["revenue"] = map[string]interface{}{
			"amount":   totalAmount,
			"order_id": order.ID,
		}
	}

	broadcast(Message{
		Event: EventDashboardUpdate,
		Data:  dashboardData,
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
	// Broadcast table_update event
	broadcast(Message{
		Event: EventTableUpdate,
		Data:  table,
	})

	// Juga broadcast ke dashboard_update dengan data untuk statistik meja
	// Dapatkan statistik tabel untuk dashboard
	var availableTables, occupiedTables, dirtyTables int

	// Hitung berdasarkan tabel yang diperbarui
	if table.Status == "available" {
		availableTables = 1
	} else if table.Status == "occupied" {
		occupiedTables = 1
	} else if table.Status == "dirty" {
		dirtyTables = 1
	}

	dashboardData := map[string]interface{}{
		"table_stats": map[string]interface{}{
			"available": availableTables,
			"occupied":  occupiedTables,
			"dirty":     dirtyTables,
		},
		"updated_table": table,
		"data_type":     "table_update",
	}

	broadcast(Message{
		Event: EventDashboardUpdate,
		Data:  dashboardData,
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

// BroadcastPaymentExpired -> notifikasi pembayaran kadaluarsa
func BroadcastPaymentExpired(payment models.Payment) {
	broadcast(Message{
		Event: EventPaymentExpired,
		Data:  payment,
	})
}

// BroadcastPaymentFailed -> notifikasi pembayaran gagal
func BroadcastPaymentFailed(payment models.Payment) {
	broadcast(Message{
		Event: EventPaymentFailed,
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

// BroadcastTableCreate -> notifikasi tabel baru dibuat
func BroadcastTableCreate(table models.Table) {
	broadcast(Message{
		Event: EventTableCreate,
		Data:  table,
	})
}

// BroadcastTableDelete -> notifikasi tabel dihapus
func BroadcastTableDelete(table models.Table) {
	broadcast(Message{
		Event: EventTableDelete,
		Data:  table,
	})
}

// BroadcastDashboardUpdate -> update dashboard
func BroadcastDashboardUpdate(data interface{}) {
	broadcast(Message{
		Event: EventDashboardUpdate,
		Data:  data,
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

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return
	}

	log.Printf("Broadcasting message: %s to %d clients", string(data), len(kdsHub.clients))

	// Koleksi koneksi yang tidak valid untuk dihapus nanti
	var invalidConnections []*websocket.Conn

	for conn, role := range kdsHub.clients {
		log.Printf("Sending to client with role %s", role)

		// Cek apakah koneksi valid terlebih dahulu
		if conn == nil || conn.WriteMessage == nil {
			invalidConnections = append(invalidConnections, conn)
			continue
		}

		err := conn.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			log.Printf("Error sending message to client: %v", err)

			// Tandai koneksi yang error untuk dihapus
			invalidConnections = append(invalidConnections, conn)
			continue
		}
		log.Printf("Successfully sent message to client with role %s", role)
	}

	// Hapus koneksi yang tidak valid
	for _, conn := range invalidConnections {
		delete(kdsHub.clients, conn)
		if conn != nil {
			conn.Close()
		}
	}
}

// BroadcastPaymentFailure -> notifikasi pembayaran gagal
func BroadcastPaymentFailure(payment models.Payment) {
	broadcast(Message{
		Event: EventPaymentFailed,
		Data:  payment,
	})
}

// BroadcastToRole broadcasts a message to clients with a specific role
func BroadcastToRole(role string, eventType string, data interface{}) {
	kdsHub.mutex.Lock()
	defer kdsHub.mutex.Unlock()

	msg := Message{
		Event: eventType,
		Data:  data,
	}

	msgData, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return
	}

	log.Printf("Broadcasting message to role %s: %s", role, string(msgData))

	// Koleksi koneksi yang tidak valid untuk dihapus nanti
	var invalidConnections []*websocket.Conn

	for conn, clientRole := range kdsHub.clients {
		if clientRole == role {
			// Validasi koneksi
			if conn == nil || conn.WriteMessage == nil {
				invalidConnections = append(invalidConnections, conn)
				continue
			}

			err := conn.WriteMessage(websocket.TextMessage, msgData)
			if err != nil {
				log.Printf("Error sending message to client: %v", err)
				invalidConnections = append(invalidConnections, conn)
				continue
			}
			log.Printf("Successfully sent message to client with role %s", role)
		}
	}

	// Hapus koneksi yang tidak valid
	for _, conn := range invalidConnections {
		delete(kdsHub.clients, conn)
		if conn != nil {
			conn.Close()
		}
	}
}
