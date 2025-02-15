package Controllers_test

import (
	"bytes"
	"encoding/json"
	"github.com/yeremiapane/restaurant-app/controllers"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/yeremiapane/restaurant-app/models"
	"github.com/yeremiapane/restaurant-app/utils"
)

// setupTestDB menggunakan SQLite in-memory untuk testing
func setupTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// AutoMigrate semua model yang diperlukan
	err = db.AutoMigrate(
		&models.User{},
		&models.Table{},
		&models.Customer{},
		&models.CleaningLog{},
		&models.MenuCategory{},
		&models.Menu{},
		&models.Order{},
		&models.OrderItem{},
		&models.Payment{},
		&models.Notification{},
	)
	if err != nil {
		panic(err)
	}

	// Jika perlu, insert data awal untuk testing (seed data)
	return db
}

// setupRouterForTest mengonfigurasi router dengan endpoint yang akan diuji
func setupRouterForTest(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	// Daftarkan endpoint untuk testing user
	userCtrl := controllers.NewUserController(db)
	router.POST("/register", userCtrl.Register)
	router.POST("/login", userCtrl.Login)
	// Tambahkan endpoint lain sesuai kebutuhan

	return router
}

func TestRegisterAndLogin(t *testing.T) {
	// Inisialisasi logger (opsional) agar tidak terjadi error saat pemanggilan utils.InitLogger
	utils.InitLogger()

	db := setupTestDB()
	router := setupRouterForTest(db)

	// --- Test Register User ---
	registerPayload := map[string]string{
		"name":     "Test User",
		"email":    "test@example.com",
		"password": "password123",
		"role":     "admin",
	}
	payloadBytes, err := json.Marshal(registerPayload)
	assert.NoError(t, err)

	req, err := http.NewRequest("POST", "/register", bytes.NewBuffer(payloadBytes))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	// Recorder untuk merekam respons
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Pastikan responsnya 201 Created
	assert.Equal(t, http.StatusCreated, w.Code)

	// Parse respons JSON
	var registerResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &registerResponse)
	assert.NoError(t, err)
	assert.Equal(t, true, registerResponse["status"])
	data := registerResponse["data"].(map[string]interface{})
	assert.NotNil(t, data["user_id"])

	// --- Test Login User ---
	loginPayload := map[string]string{
		"email":    "test@example.com",
		"password": "password123",
	}
	payloadBytes, err = json.Marshal(loginPayload)
	assert.NoError(t, err)

	req, err = http.NewRequest("POST", "/login", bytes.NewBuffer(payloadBytes))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Pastikan respons login 200 OK
	assert.Equal(t, http.StatusOK, w.Code)

	var loginResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &loginResponse)
	assert.NoError(t, err)
	assert.Equal(t, true, loginResponse["status"])
	data = loginResponse["data"].(map[string]interface{})
	token, ok := data["token"].(string)
	assert.True(t, ok)
	assert.NotEmpty(t, token)
}
