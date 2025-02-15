package controllers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yeremiapane/restaurant-app/models"
	"github.com/yeremiapane/restaurant-app/utils"
	"gorm.io/gorm"
)

type PaymentController struct {
	DB *gorm.DB
}

func NewPaymentController(db *gorm.DB) *PaymentController {
	return &PaymentController{DB: db}
}

// GetAllPayments
func (pc *PaymentController) GetAllPayments(c *gin.Context) {
	var payments []models.Payment
	if err := pc.DB.Preload("Order").Find(&payments).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}
	utils.RespondJSON(c, http.StatusOK, "All payments", payments)
}

// CreatePayment -> Contoh membayar order
func (pc *PaymentController) CreatePayment(c *gin.Context) {
	type reqBody struct {
		OrderID       uint    `json:"order_id" binding:"required"`
		PaymentMethod string  `json:"payment_method" binding:"required"` // cash, qris, dsb.
		Amount        float64 `json:"amount" binding:"required"`
	}

	var body reqBody
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err)
		return
	}

	// Pastikan order ada
	var order models.Order
	if err := pc.DB.First(&order, body.OrderID).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, err)
		return
	}

	// Buat payment record, status default 'pending'
	payment := models.Payment{
		OrderID:       body.OrderID,
		PaymentMethod: body.PaymentMethod,
		Status:        "pending",
		Amount:        body.Amount,
	}

	if err := pc.DB.Create(&payment).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	// Misalkan kita anggap pembayaran "langsung sukses"
	now := time.Now()
	payment.Status = "success"
	payment.PaymentTime = &now
	pc.DB.Save(&payment)

	// Update order status => "paid"
	order.Status = "paid"
	pc.DB.Save(&order)

	utils.RespondJSON(c, http.StatusCreated, "Payment success", payment)
}

// GetPaymentByID
func (pc *PaymentController) GetPaymentByID(c *gin.Context) {
	idStr := c.Param("payment_id")
	id, _ := strconv.Atoi(idStr)

	var payment models.Payment
	if err := pc.DB.Preload("Order").First(&payment, id).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, err)
		return
	}

	utils.RespondJSON(c, http.StatusOK, "Payment detail", payment)
}

// DeletePayment
func (pc *PaymentController) DeletePayment(c *gin.Context) {
	idStr := c.Param("payment_id")
	id, _ := strconv.Atoi(idStr)

	if err := pc.DB.Delete(&models.Payment{}, id).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	utils.RespondJSON(c, http.StatusOK, "Payment deleted", gin.H{"payment_id": id})
}
