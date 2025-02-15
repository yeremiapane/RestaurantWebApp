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

// setupTestDBForTables menggunakan SQLite in-memory khusus untuk TableController
func setupTestDBForTables() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	err = db.AutoMigrate(&models.Table{})
	if err != nil {
		panic(err)
	}
	return db
}

func setupTableRouter(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	tableCtrl := controllers.NewTableController(db)
	router.GET("/tables", tableCtrl.GetAllTables)
	router.PATCH("/tables/:table_id", tableCtrl.UpdateTableStatus)
	return router
}

func TestGetAllTables(t *testing.T) {
	utils.InitLogger()
	db := setupTestDBForTables()

	// Seed data: buat dua meja
	table1 := models.Table{TableNumber: "A1", Status: "available"}
	table2 := models.Table{TableNumber: "B1", Status: "occupied"}
	db.Create(&table1)
	db.Create(&table2)

	router := setupTableRouter(db)
	req, err := http.NewRequest("GET", "/tables", nil)
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	// Perbarui harapan sesuai dengan controller: "List of tables"
	assert.Equal(t, "List of tables", response["message"])

	// Data diharapkan berupa slice meja
	data := response["data"].([]interface{})
	assert.GreaterOrEqual(t, len(data), 2)
}

func TestUpdateTableStatus(t *testing.T) {
	utils.InitLogger()
	db := setupTestDBForTables()

	// Buat satu meja
	table := models.Table{TableNumber: "C1", Status: "available"}
	db.Create(&table)

	router := setupTableRouter(db)

	// Ubah status menjadi "occupied"
	payload := map[string]string{"status": "occupied"}
	payloadBytes, err := json.Marshal(payload)
	assert.NoError(t, err)

	url := "/tables/" + strconv.Itoa(int(table.ID))
	req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(payloadBytes))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Table status updated", response["message"])
	data := response["data"].(map[string]interface{})
	// Perhatikan bahwa field model tidak memiliki tag JSON, sehingga nama field tetap sama (misalnya "Status")
	assert.Equal(t, "occupied", data["Status"])
}
