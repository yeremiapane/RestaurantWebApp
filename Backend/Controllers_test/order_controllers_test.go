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

func setupTestDBForOrders() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	// Migrasi model yang dibutuhkan
	err = db.AutoMigrate(&models.Order{}, &models.OrderItem{}, &models.Menu{}, &models.Customer{})
	if err != nil {
		panic(err)
	}
	// Seed data: buat satu menu dan satu customer.
	// Pastikan ImageURL diisi (tidak nil)
	imageURL := ""
	menu := models.Menu{
		CategoryID:  1,
		Name:        "Test Food",
		Price:       10.0,
		Stock:       100,
		Description: "",
		ImageUrl:    &imageURL,
	}
	db.Create(&menu)
	customer := models.Customer{
		TableID: 1,
		Status:  "active",
	}
	db.Create(&customer)
	return db
}

func setupOrderRouter(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	orderCtrl := controllers.NewOrderController(db)
	router.POST("/orders", orderCtrl.CreateOrder)
	router.GET("/orders/:order_id", orderCtrl.GetOrderByID)
	return router
}

func TestCreateAndGetOrder(t *testing.T) {
	utils.InitLogger()
	db := setupTestDBForOrders()
	router := setupOrderRouter(db)

	// Payload untuk membuat order
	payload := map[string]interface{}{
		"customer_id": 1,
		"items": []map[string]interface{}{
			{
				"menu_id":  1,
				"quantity": 2,
			},
		},
	}
	payloadBytes, err := json.Marshal(payload)
	assert.NoError(t, err)

	req, err := http.NewRequest("POST", "/orders", bytes.NewBuffer(payloadBytes))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var createResp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &createResp)
	assert.NoError(t, err)
	assert.Equal(t, "Order created (draft)", createResp["message"])
	data := createResp["data"].(map[string]interface{})
	orderIDFloat, ok := data["ID"].(float64)
	assert.True(t, ok)
	orderID := int(orderIDFloat)

	// Uji GET order by ID
	url := "/orders/" + strconv.Itoa(orderID)
	req, err = http.NewRequest("GET", url, nil)
	assert.NoError(t, err)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var getResp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &getResp)
	assert.NoError(t, err)
	assert.Equal(t, "Order detail", getResp["message"])
	getData := getResp["data"].(map[string]interface{})
	assert.Equal(t, float64(orderID), getData["ID"].(float64))
}
