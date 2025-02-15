package controllers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yeremiapane/restaurant-app/models"
	"github.com/yeremiapane/restaurant-app/utils"
	"gorm.io/gorm"
)

type CleaningLogController struct {
	DB *gorm.DB
}

func NewCleaningLogController(db *gorm.DB) *CleaningLogController {
	return &CleaningLogController{DB: db}
}

// GetAllCleaningLogs
func (clc *CleaningLogController) GetAllCleaningLogs(c *gin.Context) {
	var logs []models.CleaningLog
	if err := clc.DB.Preload("Cleaner").Preload("Table").Find(&logs).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	utils.RespondJSON(c, http.StatusOK, "All cleaning logs", logs)
}

// CreateCleaningLog
func (clc *CleaningLogController) CreateCleaningLog(c *gin.Context) {
	type reqBody struct {
		CleanerID uint   `json:"cleaner_id" binding:"required"`
		TableID   uint   `json:"table_id" binding:"required"`
		Status    string `json:"status"` // pending, in_progress, done
	}
	var body reqBody
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err)
		return
	}

	logEntry := models.CleaningLog{
		CleanerID: body.CleanerID,
		TableID:   body.TableID,
		Status:    "pending",
	}
	if body.Status != "" {
		logEntry.Status = body.Status
	}

	if err := clc.DB.Create(&logEntry).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	utils.RespondJSON(c, http.StatusCreated, "Cleaning log created", logEntry)
}

// GetCleaningLogByID
func (clc *CleaningLogController) GetCleaningLogByID(c *gin.Context) {
	idStr := c.Param("clean_id")
	id, _ := strconv.Atoi(idStr)

	var logEntry models.CleaningLog
	if err := clc.DB.Preload("Cleaner").Preload("Table").First(&logEntry, id).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, err)
		return
	}

	utils.RespondJSON(c, http.StatusOK, "Cleaning log detail", logEntry)
}

// UpdateCleaningLog
func (clc *CleaningLogController) UpdateCleaningLog(c *gin.Context) {
	idStr := c.Param("clean_id")
	id, _ := strconv.Atoi(idStr)

	type reqBody struct {
		CleanerID *uint  `json:"cleaner_id"`
		TableID   *uint  `json:"table_id"`
		Status    string `json:"status"`
	}
	var body reqBody
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err)
		return
	}

	var logEntry models.CleaningLog
	if err := clc.DB.First(&logEntry, id).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, err)
		return
	}

	if body.CleanerID != nil {
		logEntry.CleanerID = *body.CleanerID
	}
	if body.TableID != nil {
		logEntry.TableID = *body.TableID
	}
	if body.Status != "" {
		logEntry.Status = body.Status
	}

	if err := clc.DB.Save(&logEntry).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	// Jika status = "done", set table => 'available' (opsional)
	if body.Status == "done" {
		var table models.Table
		if err := clc.DB.First(&table, logEntry.TableID).Error; err == nil {
			table.Status = "available"
			clc.DB.Save(&table)
		}
	}

	utils.RespondJSON(c, http.StatusOK, "Cleaning log updated", logEntry)
}

// DeleteCleaningLog
func (clc *CleaningLogController) DeleteCleaningLog(c *gin.Context) {
	idStr := c.Param("clean_id")
	id, _ := strconv.Atoi(idStr)

	if err := clc.DB.Delete(&models.CleaningLog{}, id).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	utils.RespondJSON(c, http.StatusOK, "Cleaning log deleted", gin.H{"clean_id": id})
}
