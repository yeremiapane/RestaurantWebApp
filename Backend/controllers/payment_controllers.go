package controllers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/yeremiapane/restaurant-app/kds"
	"github.com/yeremiapane/restaurant-app/models"
	"github.com/yeremiapane/restaurant-app/utils"
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

// CreatePayment -> Memproses pembayaran
func (pc *PaymentController) CreatePayment(c *gin.Context) {
	type reqBody struct {
		OrderID       uint    `json:"order_id" binding:"required"`
		PaymentMethod string  `json:"payment_method" binding:"required"` // cash, qris, dll
		Amount        float64 `json:"amount" binding:"required"`
	}

	var body reqBody
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err)
		return
	}

	// Cek order
	var order models.Order
	if err := pc.DB.First(&order, body.OrderID).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, fmt.Errorf("order not found"))
		return
	}

	// Validasi status order
	if order.Status != "pending_payment" {
		utils.RespondError(c, http.StatusBadRequest, fmt.Errorf("invalid order status for payment"))
		return
	}

	// Buat payment dengan status 'pending'
	payment := models.Payment{
		OrderID:       body.OrderID,
		PaymentMethod: body.PaymentMethod,
		Status:        "pending",
		Amount:        body.Amount,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := pc.DB.Create(&payment).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	// Broadcast payment pending ke admin & staff
	kds.BroadcastPaymentPending(payment)

	// Jika metode pembayaran QRIS
	if body.PaymentMethod == "qris" {
		// Implementasi integrasi payment gateway di sini
		// Contoh mock response payment gateway
		go func() {
			// Simulasi delay pembayaran
			time.Sleep(5 * time.Second)
			pc.handlePaymentCallback(payment.ID, "success")
		}()
	}

	utils.RespondJSON(c, http.StatusCreated, "Payment initiated", payment)
}

// handlePaymentCallback menangani callback dari payment gateway
func (pc *PaymentController) handlePaymentCallback(paymentID uint, status string) {
	var payment models.Payment
	if err := pc.DB.First(&payment, paymentID).Error; err != nil {
		return
	}

	tx := pc.DB.Begin()

	// Update payment status
	now := time.Now()
	payment.Status = status
	payment.PaymentTime = &now
	payment.UpdatedAt = now

	if err := tx.Save(&payment).Error; err != nil {
		tx.Rollback()
		return
	}

	// Jika pembayaran sukses, update order
	if status == "success" {
		var order models.Order
		if err := tx.First(&order, payment.OrderID).Error; err != nil {
			tx.Rollback()
			return
		}

		order.Status = "paid"
		order.UpdatedAt = now

		if err := tx.Save(&order).Error; err != nil {
			tx.Rollback()
			return
		}

		// Commit transaction
		tx.Commit()

		// Broadcast updates
		kds.BroadcastPaymentSuccess(payment)
		kds.BroadcastOrderUpdate(order)
		kds.BroadcastKitchenUpdate(order)
		kds.BroadcastStaffNotification(fmt.Sprintf("Payment received for Order #%d", order.ID))
	}
}

// VerifyPayment untuk admin memverifikasi pembayaran cash
func (pc *PaymentController) VerifyPayment(c *gin.Context) {
	roleInterface, _ := c.Get("role")
	if roleInterface != "admin" && roleInterface != "staff" {
		utils.RespondError(c, http.StatusForbidden, ErrNoPermission)
		return
	}

	paymentID := c.Param("payment_id")

	var payment models.Payment
	if err := pc.DB.First(&payment, paymentID).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, err)
		return
	}

	// Verifikasi pembayaran cash
	if payment.PaymentMethod == "cash" {
		pc.handlePaymentCallback(payment.ID, "success")
	}

	// Buat receipt controller langsung
	receiptCtrl := &ReceiptController{DB: pc.DB}
	receiptCtrl.GenerateReceipt(c)

	utils.RespondJSON(c, http.StatusOK, "Payment verified", payment)
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
