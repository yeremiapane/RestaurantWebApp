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

func setupTestDBForCleaningLogs() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	err = db.AutoMigrate(&models.CleaningLog{}, &models.Table{}, &models.User{})
	if err != nil {
		panic(err)
	}
	// Seed data: buat satu meja dan satu cleaner
	table := models.Table{TableNumber: "D1", Status: "dirty"}
	db.Create(&table)
	cleaner := models.User{
		Name:     "Cleaner1",
		Email:    "cleaner1@example.com",
		Password: "secret", // untuk test, password plain
		Role:     "cleaner",
	}
	db.Create(&cleaner)
	return db
}

func setupCleaningLogRouter(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	cleaningCtrl := controllers.NewCleaningLogController(db)
	router.POST("/cleaning-logs", cleaningCtrl.CreateCleaningLog)
	router.GET("/cleaning-logs/:clean_id", cleaningCtrl.GetCleaningLogByID)
	return router
}

func TestCreateAndGetCleaningLog(t *testing.T) {
	utils.InitLogger()
	db := setupTestDBForCleaningLogs()
	router := setupCleaningLogRouter(db)

	payload := map[string]interface{}{
		"cleaner_id": 1,
		"table_id":   1,
		"status":     "pending",
	}
	payloadBytes, err := json.Marshal(payload)
	assert.NoError(t, err)

	req, err := http.NewRequest("POST", "/cleaning-logs", bytes.NewBuffer(payloadBytes))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	var createResp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &createResp)
	assert.NoError(t, err)
	assert.Equal(t, "Cleaning log created", createResp["message"])
	data := createResp["data"].(map[string]interface{})
	cleanIDFloat, ok := data["ID"].(float64)
	assert.True(t, ok)
	cleanID := int(cleanIDFloat)

	// Uji GET cleaning log
	url := "/cleaning-logs/" + strconv.Itoa(cleanID)
	req, err = http.NewRequest("GET", url, nil)
	assert.NoError(t, err)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var getResp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &getResp)
	assert.NoError(t, err)
	assert.Equal(t, "Cleaning log detail", getResp["message"])
	getData := getResp["data"].(map[string]interface{})
	assert.Equal(t, float64(cleanID), getData["ID"].(float64))
}
