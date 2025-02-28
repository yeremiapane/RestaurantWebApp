package controllers

import (
	"fmt"
	"net/http"
	_ "strconv"

	"github.com/gin-gonic/gin"
	"github.com/yeremiapane/restaurant-app/kds"
	"github.com/yeremiapane/restaurant-app/models"
	"github.com/yeremiapane/restaurant-app/utils"
	"gorm.io/gorm"
)

type TableController struct {
	DB *gorm.DB
}

func NewTableController(db *gorm.DB) *TableController {
	return &TableController{DB: db}
}

// CreateTable -> menambahkan meja baru
func (tc *TableController) CreateTable(c *gin.Context) {
	var req struct {
		TableNumber string `json:"table_number" binding:"required"`
		Status      string `json:"status"` // optional, default "available"
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err)
		return
	}

	table := models.Table{
		TableNumber: req.TableNumber,
		Status:      "available",
	}
	if req.Status != "" {
		table.Status = req.Status
	}

	if err := tc.DB.Create(&table).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	// Broadcast dengan data lengkap
	stats := tc.getDashboardStats()
	kds.BroadcastMessage(kds.Message{
		Event: kds.EventTableCreate,
		Data: map[string]interface{}{
			"table": table,
			"stats": stats,
		},
	})

	utils.InfoLogger.Printf("New table created: %s (status=%s)", table.TableNumber, table.Status)
	utils.RespondJSON(c, http.StatusCreated, "Table created successfully", table)
}

// GetAllTables -> menampilkan seluruh meja
func (tc *TableController) GetAllTables(c *gin.Context) {
	var tables []models.Table
	if err := tc.DB.Find(&tables).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}
	utils.RespondJSON(c, http.StatusOK, "List of tables", tables)
}

// UpdateTableStatus -> update status meja
func (tc *TableController) UpdateTableStatus(c *gin.Context) {
	tableID := c.Param("table_id")
	var body struct {
		Status string `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err)
		return
	}

	var table models.Table
	if err := tc.DB.First(&table, tableID).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, err)
		return
	}

	table.Status = body.Status
	if err := tc.DB.Save(&table).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	// Broadcast dengan data lengkap
	stats := tc.getDashboardStats()
	kds.BroadcastMessage(kds.Message{
		Event: kds.EventTableUpdate,
		Data: map[string]interface{}{
			"table": table,
			"stats": stats,
		},
	})

	utils.InfoLogger.Printf("Table %d status changed to %s", table.ID, table.Status)
	utils.RespondJSON(c, http.StatusOK, "Table status updated", table)
}

// DeleteTable -> menghapus meja
func (tc *TableController) DeleteTable(c *gin.Context) {
	tableID := c.Param("table_id")
	var table models.Table

	if err := tc.DB.First(&table, tableID).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, err)
		return
	}

	if err := tc.DB.Delete(&table).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	// Broadcast dengan data lengkap
	stats := tc.getDashboardStats()
	kds.BroadcastMessage(kds.Message{
		Event: kds.EventTableDelete,
		Data: map[string]interface{}{
			"table_id": table.ID,
			"stats":    stats,
		},
	})

	utils.InfoLogger.Printf("Table %d deleted", table.ID)
	utils.RespondJSON(c, http.StatusOK, "Table deleted", gin.H{
		"id": table.ID,
	})
}

// GetTableByID -> detail satu meja (opsional)
func (tc *TableController) GetTableByID(c *gin.Context) {
	tableID := c.Param("table_id")
	var table models.Table
	if err := tc.DB.First(&table, tableID).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, err)
		return
	}
	utils.RespondJSON(c, http.StatusOK, "Table detail", table)
}

// Extra: FindTablesByStatus -> mis. list meja available
func (tc *TableController) FindTablesByStatus(c *gin.Context) {
	status := c.Query("status")
	if status == "" {
		status = "available"
	}
	var tables []models.Table
	if err := tc.DB.Where("status = ?", status).Find(&tables).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}
	utils.RespondJSON(c, http.StatusOK, "Tables with status: "+status, tables)
}

// MarkTableClean untuk Cleaner menandai meja siap digunakan
func (tc *TableController) MarkTableClean(c *gin.Context) {
	roleInterface, _ := c.Get("role")
	if roleInterface != "cleaner" && roleInterface != "staff" {
		utils.RespondError(c, http.StatusForbidden, ErrNoPermission)
		return
	}

	tableID := c.Param("table_id")

	var table models.Table
	if err := tc.DB.First(&table, tableID).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, err)
		return
	}

	if table.Status != "dirty" {
		utils.RespondError(c, http.StatusBadRequest, fmt.Errorf("table is not dirty"))
		return
	}

	table.Status = "available"
	if err := tc.DB.Save(&table).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	utils.RespondJSON(c, http.StatusOK, "Table marked as clean", table)
}

// getDashboardStats menghitung statistik dashboard
func (tc *TableController) getDashboardStats() map[string]interface{} {
	var availableCount, occupiedCount, dirtyCount int64

	tc.DB.Model(&models.Table{}).Where("status = ?", "available").Count(&availableCount)
	tc.DB.Model(&models.Table{}).Where("status = ?", "occupied").Count(&occupiedCount)
	tc.DB.Model(&models.Table{}).Where("status = ?", "dirty").Count(&dirtyCount)

	return map[string]interface{}{
		"available": availableCount,
		"occupied":  occupiedCount,
		"dirty":     dirtyCount,
		"total":     availableCount + occupiedCount + dirtyCount,
	}
}
