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

func setupTestDBForMenus() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	err = db.AutoMigrate(&models.Menu{}, &models.MenuCategory{})
	if err != nil {
		panic(err)
	}
	// Seed: buat satu kategori
	category := models.MenuCategory{
		Name: "Food",
	}
	db.Create(&category)
	return db
}

func setupMenuRouter(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	menuCtrl := controllers.NewMenuController(db)
	router.GET("/menus", menuCtrl.GetAllMenus)
	router.POST("/menus", menuCtrl.CreateMenu)
	router.GET("/menus/:menu_id", menuCtrl.GetMenuByID)
	router.PATCH("/menus/:menu_id", menuCtrl.UpdateMenu)
	router.DELETE("/menus/:menu_id", menuCtrl.DeleteMenu)
	return router
}

func TestMenuCRUD(t *testing.T) {
	utils.InitLogger()
	db := setupTestDBForMenus()
	router := setupMenuRouter(db)

	// Create Menu: sertakan "image_url" dengan string kosong agar tidak NULL
	payload := map[string]interface{}{
		"category_id": 1,
		"name":        "Pizza",
		"price":       12.5,
		"stock":       50,
		"description": "Delicious cheese pizza",
		"image_url":   "", // field ini diset agar tidak NULL
	}
	payloadBytes, err := json.Marshal(payload)
	assert.NoError(t, err)

	req, err := http.NewRequest("POST", "/menus", bytes.NewBuffer(payloadBytes))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	// Expected HTTP status 201 Created
	assert.Equal(t, http.StatusCreated, w.Code)

	var createResp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &createResp)
	assert.NoError(t, err)

	// Pastikan response memiliki data menu
	data, ok := createResp["data"].(map[string]interface{})
	assert.True(t, ok, "data response harus berupa map")
	menuIDFloat, ok := data["ID"].(float64)
	assert.True(t, ok, "menu ID harus berupa float64")
	menuID := int(menuIDFloat)

	// Get Menu by ID
	url := "/menus/" + strconv.Itoa(menuID)
	req, err = http.NewRequest("GET", url, nil)
	assert.NoError(t, err)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Update Menu
	updatePayload := map[string]interface{}{
		"name":  "Updated Pizza",
		"price": 15.0,
	}
	payloadBytes, err = json.Marshal(updatePayload)
	assert.NoError(t, err)
	req, err = http.NewRequest("PATCH", url, bytes.NewBuffer(payloadBytes))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Delete Menu
	req, err = http.NewRequest("DELETE", url, nil)
	assert.NoError(t, err)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
