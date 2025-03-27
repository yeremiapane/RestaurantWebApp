package controllers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yeremiapane/restaurant-app/models"
	"github.com/yeremiapane/restaurant-app/utils"
	"gorm.io/gorm"
)

type NotificationController struct {
	DB *gorm.DB
}

func NewNotificationController(db *gorm.DB) *NotificationController {
	return &NotificationController{DB: db}
}

// GetAllNotifications
func (nc *NotificationController) GetAllNotifications(c *gin.Context) {
	var notifs []models.Notification
	if err := nc.DB.Preload("User").Find(&notifs).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}
	utils.RespondJSON(c, http.StatusOK, "All notifications", notifs)
}

// CreateNotification -> broadcast atau specific user
func (nc *NotificationController) CreateNotification(c *gin.Context) {
	type reqBody struct {
		UserID  *uint  `json:"user_id"`
		Title   string `json:"title"`
		Message string `json:"message" binding:"required"`
	}
	var body reqBody
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err)
		return
	}

	notif := models.Notification{
		Title:   body.Title,
		Message: body.Message,
	}
	if body.UserID != nil {
		notif.UserID = body.UserID
	}

	if err := nc.DB.Create(&notif).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	// Di sinilah Anda bisa memicu push notif / websocket dsb.
	utils.InfoLogger.Printf("Notification created: %v", notif.Message)

	utils.RespondJSON(c, http.StatusCreated, "Notification created", notif)
}

// GetNotificationByID
func (nc *NotificationController) GetNotificationByID(c *gin.Context) {
	idStr := c.Param("notif_id")
	id, _ := strconv.Atoi(idStr)

	var notif models.Notification
	if err := nc.DB.Preload("User").First(&notif, id).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, err)
		return
	}

	utils.RespondJSON(c, http.StatusOK, "Notification detail", notif)
}

// DeleteNotification
func (nc *NotificationController) DeleteNotification(c *gin.Context) {
	idStr := c.Param("notif_id")
	id, _ := strconv.Atoi(idStr)

	if err := nc.DB.Delete(&models.Notification{}, id).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}
	utils.RespondJSON(c, http.StatusOK, "Notification deleted", gin.H{"notif_id": id})
}
