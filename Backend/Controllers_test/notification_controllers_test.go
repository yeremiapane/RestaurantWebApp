package Controllers_test

import (
	"bytes"
	"encoding/json"
	"github.com/yeremiapane/restaurant-app/controllers"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/yeremiapane/restaurant-app/models"
	"github.com/yeremiapane/restaurant-app/utils"
)

func setupTestDBForNotifications() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	err = db.AutoMigrate(&models.Notification{}, &models.User{})
	if err != nil {
		panic(err)
	}
	// Seed: buat user untuk notifikasi
	user := models.User{
		Name:     "Test User",
		Email:    "testuser@example.com",
		Password: "secret",
		Role:     "admin",
	}
	db.Create(&user)
	return db
}

func setupNotificationRouter(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	notifCtrl := controllers.NewNotificationController(db)
	router.GET("/notifications", notifCtrl.GetAllNotifications)
	router.POST("/notifications", notifCtrl.CreateNotification)
	router.GET("/notifications/:notif_id", notifCtrl.GetNotificationByID)
	router.DELETE("/notifications/:notif_id", notifCtrl.DeleteNotification)
	return router
}

func TestNotificationCRUD(t *testing.T) {
	utils.InitLogger()
	db := setupTestDBForNotifications()
	router := setupNotificationRouter(db)

	// Create Notification
	payload := map[string]interface{}{
		"user_id": 1,
		"title":   "Test Notification",
		"message": "This is a test message",
	}
	payloadBytes, err := json.Marshal(payload)
	assert.NoError(t, err)
	req, err := http.NewRequest("POST", "/notifications", bytes.NewBuffer(payloadBytes))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	var createResp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &createResp)
	assert.NoError(t, err)
	data := createResp["data"].(map[string]interface{})
	notifIDFloat, ok := data["ID"].(float64)
	assert.True(t, ok)
	notifID := int(notifIDFloat)

	// Get Notification
	url := "/notifications/" + strconv.Itoa(notifID)
	req, err = http.NewRequest("GET", url, nil)
	assert.NoError(t, err)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Delete Notification
	req, err = http.NewRequest("DELETE", url, nil)
	assert.NoError(t, err)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
