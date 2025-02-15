package Controllers_test

import (
	"bytes"
	"encoding/json"
	"github.com/yeremiapane/restaurant-app/controllers"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	_ "time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/yeremiapane/restaurant-app/models"
	"github.com/yeremiapane/restaurant-app/utils"
)

func setupTestDBForPayments() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	// Migrasi model yang diperlukan untuk Payment
	err = db.AutoMigrate(&models.Payment{}, &models.Order{}, &models.Customer{}, &models.OrderItem{}, &models.Menu{})
	if err != nil {
		panic(err)
	}
	// Seed data: buat menu, customer, order dan order item
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
	order := models.Order{
		CustomerID:  customer.ID,
		Status:      "draft",
		TotalAmount: 20.0,
	}
	db.Create(&order)
	orderItem := models.OrderItem{
		OrderID:  order.ID,
		MenuID:   menu.ID,
		Quantity: 2,
		Price:    menu.Price,
	}
	db.Create(&orderItem)
	return db
}

func setupPaymentRouter(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	paymentCtrl := controllers.NewPaymentController(db)
	router.POST("/payments", paymentCtrl.CreatePayment)
	router.GET("/payments/:payment_id", paymentCtrl.GetPaymentByID)
	return router
}

func TestCreateAndGetPayment(t *testing.T) {
	utils.InitLogger()
	db := setupTestDBForPayments()
	router := setupPaymentRouter(db)

	payload := map[string]interface{}{
		"order_id":       1,
		"payment_method": "cash",
		"amount":         20.0,
	}
	payloadBytes, err := json.Marshal(payload)
	assert.NoError(t, err)

	req, err := http.NewRequest("POST", "/payments", bytes.NewBuffer(payloadBytes))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	var createResp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &createResp)
	assert.NoError(t, err)
	assert.Equal(t, "Payment success", createResp["message"])
	data := createResp["data"].(map[string]interface{})
	paymentIDFloat, ok := data["ID"].(float64)
	assert.True(t, ok)
	paymentID := int(paymentIDFloat)

	// Uji GET Payment
	url := "/payments/" + strconv.Itoa(paymentID)
	req, err = http.NewRequest("GET", url, nil)
	assert.NoError(t, err)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var getResp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &getResp)
	assert.NoError(t, err)
	assert.Equal(t, "Payment detail", getResp["message"])
	getData := getResp["data"].(map[string]interface{})
	assert.Equal(t, float64(paymentID), getData["ID"].(float64))
}
