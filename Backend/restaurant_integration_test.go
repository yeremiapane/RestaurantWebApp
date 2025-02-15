package main

import (
	"bytes"
	"encoding/json"
	"github.com/yeremiapane/restaurant-app/utils"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	_ "time"

	"github.com/gin-gonic/gin"
	"github.com/yeremiapane/restaurant-app/models"
	"github.com/yeremiapane/restaurant-app/router"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestMain(m *testing.M) {
	utils.InitLogger() // ✅ Ensure logger is ready for tests
	os.Exit(m.Run())
}

// TestEndToEndIntegration menguji flow utama:
// 0. Buat user & menu di seed, lalu login -> token
// 1. Create order (draft)
// 2. Payment => paid
// 3. Cook item => in_progress -> ready
// 4. Cek order => ready
// 5. Mark completed
func TestEndToEndIntegration(t *testing.T) {
	// 1. Setup DB in-memory + migrate
	db := setupTestDB()
	// 2. Setup router
	r := router.SetupRouter(db)

	// 3. Login to get the token
	token := loginTest(t, r)

	// 4. Buat order (draft)
	orderID := createOrderTest(t, r, token)

	// 5. Bayar order => paid
	payOrderTest(t, r, orderID, token)

	// 6. Start cook item => finish => ready => order => ready
	checkCookingProcessTest(t, r, orderID, token)

	// 7. Mark completed
	completeOrderTest(t, r, orderID, token)
}

// setupTestDB -> migrasi model di SQLite in-memory + seed data
func setupTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to open in-memory sqlite: %v", err)
	}

	// Migrasi model
	err = db.AutoMigrate(
		&models.User{},
		&models.Customer{},
		&models.Table{},
		&models.MenuCategory{},
		&models.Menu{},
		&models.Order{},
		&models.OrderItem{},
		&models.Payment{},
		&models.CleaningLog{},
		&models.Notification{},
	)
	if err != nil {
		log.Fatalf("failed to migrate: %v", err)
	}

	// Buat admin user
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.DefaultCost)
	db.Create(&models.User{
		Name:     "Test Admin",
		Email:    "admin@example.com",
		Password: string(hashedPassword), // Pastikan password tersimpan dalam bentuk hash
		Role:     "admin",
	})

	// Buat menu dengan `image_url` yang tidak NULL
	db.Create(&models.Menu{
		Name:     "Nasi Goreng",
		Price:    15000,
		Stock:    100,
		ImageUrl: ptrString("default.jpg"), // Gunakan helper ptrString()
	})

	// Buat meja dan customer
	db.Create(&models.Table{
		TableNumber: "A1",
		Status:      "available",
	})
	db.Create(&models.Customer{
		TableID: 1,
		Status:  "active",
	})

	return db
}

func loginTest(t *testing.T, r *gin.Engine) string {
	body := map[string]string{
		"email":    "admin@example.com",
		"password": "secret123", // Harus sesuai dengan yang di seed
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Debug response jika terjadi error
	log.Printf("Login response: Code=%d, Body=%s", w.Code, w.Body.String())

	if w.Code != http.StatusOK {
		t.Fatalf("loginTest fail: code=%d, body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Status  bool   `json:"status"`
		Message string `json:"message"`
		Data    struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)

	// Debug log tambahan
	log.Printf("Login response parsed: %+v", resp)

	if !resp.Status {
		t.Fatalf("loginTest: status=false, msg=%s", resp.Message)
	}

	if resp.Data.Token == "" {
		t.Fatalf("loginTest: token empty")
	}

	return resp.Data.Token
}

// createOrderTest -> POST /orders => status=201 => order.status=draft
func createOrderTest(t *testing.T, r *gin.Engine, token string) uint {
	bodyData := map[string]interface{}{
		"customer_id": 1,
		"items": []map[string]interface{}{
			{
				"menu_id":  1,
				"quantity": 2,
				"notes":    "Pedas",
			},
		},
	}
	bodyBytes, _ := json.Marshal(bodyData)

	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token) // Include the token

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("createOrderTest: expected 201, got %d, body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Status  bool   `json:"status"`
		Message string `json:"message"`
		Data    struct {
			ID          uint    `json:"id"`
			CustomerID  uint    `json:"customer_id"`
			Status      string  `json:"status"`
			TotalAmount float64 `json:"total_amount"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp.Status {
		t.Fatalf("createOrderTest: status=false, msg=%s", resp.Message)
	}
	if resp.Data.Status != "draft" {
		t.Fatalf("createOrderTest: expected order status 'draft', got %s", resp.Data.Status)
	}

	return resp.Data.ID
}

// payOrderTest -> POST /payments => order => paid
func payOrderTest(t *testing.T, r *gin.Engine, orderID uint, token string) {
	bodyData := map[string]interface{}{
		"order_id":       orderID,
		"payment_method": "cash",
		"amount":         30000, // misal 30k
	}
	bodyBytes, _ := json.Marshal(bodyData)

	req := httptest.NewRequest(http.MethodPost, "/payments", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token) // Include the token

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("payOrderTest: expected 201, got %d, body=%s", w.Code, w.Body.String())
	}

	// check payment response
	var resp struct {
		Status  bool   `json:"status"`
		Message string `json:"message"`
		Data    struct {
			ID     uint   `json:"id"`
			Status string `json:"status"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp.Status {
		t.Fatalf("payOrderTest: status=false, msg=%s", resp.Message)
	}
	if resp.Data.Status != "success" {
		t.Fatalf("payOrderTest: expected payment.status=success, got %s", resp.Data.Status)
	}
}

// checkCookingProcessTest -> get order => ambil item => start -> finish => order => ready
func checkCookingProcessTest(t *testing.T, r *gin.Engine, orderID uint, token string) {
	// GET /orders/:id
	reqGet := httptest.NewRequest(http.MethodGet, "/orders/"+intToString(orderID), nil)
	reqGet.Header.Set("Authorization", "Bearer "+token) // Include the token
	wGet := httptest.NewRecorder()
	r.ServeHTTP(wGet, reqGet)
	if wGet.Code != http.StatusOK {
		t.Fatalf("checkCookingProcessTest GET: want 200, got %d", wGet.Code)
	}

	var getResp struct {
		Status bool `json:"status"`
		Data   struct {
			ID         uint `json:"id"`
			OrderItems []struct {
				ID     uint   `json:"id"`
				Status string `json:"status"`
			} `json:"order_items"`
		} `json:"data"`
	}
	json.Unmarshal(wGet.Body.Bytes(), &getResp)
	if !getResp.Status || len(getResp.Data.OrderItems) < 1 {
		t.Fatalf("checkCookingProcessTest: no items or status=false")
	}
	itemID := getResp.Data.OrderItems[0].ID

	// start cooking item => /order-items/:id/start
	reqStart := httptest.NewRequest(http.MethodPost,
		"/order-items/"+intToString(itemID)+"/start", nil)
	reqStart.Header.Set("Authorization", "Bearer "+token) // Include the token
	wStart := httptest.NewRecorder()
	r.ServeHTTP(wStart, reqStart)
	if wStart.Code != http.StatusOK {
		t.Fatalf("startCooking item: code %d, body=%s", wStart.Code, wStart.Body.String())
	}

	// finish cooking item => /order-items/:id/finish
	reqFinish := httptest.NewRequest(http.MethodPost,
		"/order-items/"+intToString(itemID)+"/finish", nil)
	reqFinish.Header.Set("Authorization", "Bearer "+token) // Include the token
	wFinish := httptest.NewRecorder()
	r.ServeHTTP(wFinish, reqFinish)
	if wFinish.Code != http.StatusOK {
		t.Fatalf("finishCooking item: code %d, body=%s", wFinish.Code, wFinish.Body.String())
	}

	// parse item finish resp
	var finishResp struct {
		Status bool `json:"status"`
		Data   struct {
			ID     uint   `json:"id"`
			Status string `json:"status"`
		} `json:"data"`
	}
	json.Unmarshal(wFinish.Body.Bytes(), &finishResp)
	if !finishResp.Status {
		t.Fatalf("finishCooking item: status=false data=%v", finishResp.Data)
	}
	if finishResp.Data.Status != "ready" {
		t.Fatalf("finishCooking item: want 'ready', got %s", finishResp.Data.Status)
	}

	// re-check order => must be 'ready'
	reqGet2 := httptest.NewRequest(http.MethodGet, "/orders/"+intToString(orderID), nil)
	reqGet2.Header.Set("Authorization", "Bearer "+token) // Include the token
	wGet2 := httptest.NewRecorder()
	r.ServeHTTP(wGet2, reqGet2)
	if wGet2.Code != http.StatusOK {
		t.Fatalf("checkCookingProcessTest GET2: code %d, body=%s", wGet2.Code, wGet2.Body.String())
	}

	var getResp2 struct {
		Status bool `json:"status"`
		Data   struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	json.Unmarshal(wGet2.Body.Bytes(), &getResp2)
	if getResp2.Data.Status != "ready" {
		t.Fatalf("Expected order.status='ready', got %s", getResp2.Data.Status)
	}
}

// completeOrderTest -> staff menekan /orders/:id/complete => 'completed'
func completeOrderTest(t *testing.T, r *gin.Engine, orderID uint, token string) {
	req := httptest.NewRequest(http.MethodPost,
		"/orders/"+intToString(orderID)+"/complete", nil)
	req.Header.Set("Authorization", "Bearer "+token) // Include the token

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("completeOrderTest: code=%d, body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Status bool `json:"status"`
		Data   struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp.Status {
		t.Fatalf("completeOrderTest: status=false data=%v", resp.Data)
	}
	if resp.Data.Status != "completed" {
		t.Fatalf("completeOrderTest: want 'completed', got %s", resp.Data.Status)
	}
}

// Helper intToString
func intToString(num uint) string {
	return strconv.FormatUint(uint64(num), 10)
}

// ptrString -> helper utk bikin *string dari literal
func ptrString(s string) *string {
	return &s
}
