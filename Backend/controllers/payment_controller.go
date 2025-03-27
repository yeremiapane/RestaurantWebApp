package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/yeremiapane/restaurant-app/kds"
	"github.com/yeremiapane/restaurant-app/middlewares"
	"github.com/yeremiapane/restaurant-app/models"
	"github.com/yeremiapane/restaurant-app/services"
	"github.com/yeremiapane/restaurant-app/utils"

	"crypto/sha512"
	"encoding/hex"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// MidtransConfig menyimpan konfigurasi Midtrans
type MidtransConfig struct {
	ServerKey     string
	ClientKey     string
	IsProduction  bool
	MerchantID    string
	MerchantName  string
	MerchantEmail string
	MerchantPhone string
}

var midtransConfig = MidtransConfig{
	ServerKey:     os.Getenv("MIDTRANS_SERVER_KEY"),
	ClientKey:     os.Getenv("MIDTRANS_CLIENT_KEY"),
	IsProduction:  os.Getenv("MIDTRANS_ENV") == "production",
	MerchantID:    os.Getenv("MIDTRANS_MERCHANT_ID"),
	MerchantName:  os.Getenv("MIDTRANS_MERCHANT_NAME"),
	MerchantEmail: os.Getenv("MIDTRANS_MERCHANT_EMAIL"),
	MerchantPhone: os.Getenv("MIDTRANS_MERCHANT_PHONE"),
}

// initMidtransConfig menginisialisasi konfigurasi Midtrans
func initMidtransConfig() {
	// Debug: Print environment variables
	fmt.Printf("Loading Midtrans configuration...\n")
	fmt.Printf("MIDTRANS_SERVER_KEY from env: %s\n", os.Getenv("MIDTRANS_SERVER_KEY"))
	fmt.Printf("MIDTRANS_CLIENT_KEY from env: %s\n", os.Getenv("MIDTRANS_CLIENT_KEY"))
	fmt.Printf("MIDTRANS_ENV from env: %s\n", os.Getenv("MIDTRANS_ENV"))
	fmt.Printf("MIDTRANS_MERCHANT_ID from env: %s\n", os.Getenv("MIDTRANS_MERCHANT_ID"))
	fmt.Printf("MIDTRANS_MERCHANT_NAME from env: %s\n", os.Getenv("MIDTRANS_MERCHANT_NAME"))
	fmt.Printf("MIDTRANS_MERCHANT_EMAIL from env: %s\n", os.Getenv("MIDTRANS_MERCHANT_EMAIL"))
	fmt.Printf("MIDTRANS_MERCHANT_PHONE from env: %s\n", os.Getenv("MIDTRANS_MERCHANT_PHONE"))

	// Reinitialize configuration
	midtransConfig = MidtransConfig{
		ServerKey:     os.Getenv("MIDTRANS_SERVER_KEY"),
		ClientKey:     os.Getenv("MIDTRANS_CLIENT_KEY"),
		IsProduction:  os.Getenv("MIDTRANS_ENV") == "production",
		MerchantID:    os.Getenv("MIDTRANS_MERCHANT_ID"),
		MerchantName:  os.Getenv("MIDTRANS_MERCHANT_NAME"),
		MerchantEmail: os.Getenv("MIDTRANS_MERCHANT_EMAIL"),
		MerchantPhone: os.Getenv("MIDTRANS_MERCHANT_PHONE"),
	}
}

// ValidateMidtransConfig memvalidasi konfigurasi Midtrans
func ValidateMidtransConfig() error {
	// Reinitialize configuration before validation
	initMidtransConfig()

	// Debug: Print current configuration
	fmt.Printf("Current Midtrans Config:\n")
	fmt.Printf("ServerKey: %s\n", midtransConfig.ServerKey)
	fmt.Printf("ClientKey: %s\n", midtransConfig.ClientKey)
	fmt.Printf("IsProduction: %v\n", midtransConfig.IsProduction)
	fmt.Printf("MerchantID: %s\n", midtransConfig.MerchantID)
	fmt.Printf("MerchantName: %s\n", midtransConfig.MerchantName)
	fmt.Printf("MerchantEmail: %s\n", midtransConfig.MerchantEmail)
	fmt.Printf("MerchantPhone: %s\n", midtransConfig.MerchantPhone)

	if midtransConfig.ServerKey == "" {
		return fmt.Errorf("MIDTRANS_SERVER_KEY is not set")
	}
	if midtransConfig.ClientKey == "" {
		return fmt.Errorf("MIDTRANS_CLIENT_KEY is not set")
	}
	if midtransConfig.MerchantID == "" {
		return fmt.Errorf("MIDTRANS_MERCHANT_ID is not set")
	}
	if midtransConfig.MerchantName == "" {
		return fmt.Errorf("MIDTRANS_MERCHANT_NAME is not set")
	}
	if midtransConfig.MerchantEmail == "" {
		return fmt.Errorf("MIDTRANS_MERCHANT_EMAIL is not set")
	}
	if midtransConfig.MerchantPhone == "" {
		return fmt.Errorf("MIDTRANS_MERCHANT_PHONE is not set")
	}
	return nil
}

// PaymentRequest adalah struktur untuk request pembuatan pembayaran
type PaymentRequest struct {
	OrderID       uint    `json:"order_id" binding:"required"`
	PaymentMethod string  `json:"payment_method" binding:"required,oneof=cash qris"`
	Amount        float64 `json:"amount" binding:"required,min=0"`
	ReferenceID   string  `json:"reference_id" binding:"required"`
	CashReceived  float64 `json:"cash_received"`
}

// PaymentCallbackRequest adalah struktur untuk request callback dari payment gateway
type PaymentCallbackRequest struct {
	PaymentID   uint   `json:"payment_id" binding:"required"`
	Status      string `json:"status" binding:"required"`
	ReferenceID string `json:"reference_id" binding:"required"`
}

// SetupPaymentRoutes mendaftarkan rute-rute untuk payment
func SetupPaymentRoutes(router *gin.Engine) {
	paymentRouter := router.Group("/admin/payments")
	{
		paymentRouter.Use(middlewares.EnhancedAuthMiddleware())
		paymentRouter.GET("", GetPayments)
		paymentRouter.GET("/:id", GetPayment)
		paymentRouter.POST("", CreatePayment)
		paymentRouter.POST("/:id/verify", VerifyPayment)
	}
}

// GetPayments menampilkan daftar pembayaran
func GetPayments(c *gin.Context) {
	db := utils.GetDB()

	// Filter berdasarkan order_id jika ada
	orderID := c.Query("order_id")

	var payments []models.Payment

	if orderID != "" {
		// Jika order_id diberikan, filter berdasarkan order_id
		db.Preload("Order").Where("order_id = ?", orderID).Order("created_at DESC").Find(&payments)
	} else {
		// Jika tidak, ambil semua pembayaran
		db.Preload("Order").Order("created_at DESC").Find(&payments)
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   payments,
	})
}

// GetPayment menampilkan detail pembayaran berdasarkan ID
func GetPayment(c *gin.Context) {
	db := utils.GetDB()
	id := c.Param("id")

	var payment models.Payment
	if err := db.Preload("Order").First(&payment, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Payment not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   payment,
	})
}

// CreatePayment membuat pembayaran baru
func CreatePayment(c *gin.Context) {
	// Inisialisasi Midtrans config
	initMidtransConfig()

	var req PaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err)
		return
	}

	// Validasi jumlah pembayaran
	if req.Amount <= 0 {
		utils.RespondError(c, http.StatusBadRequest, errors.New("amount must be greater than 0"))
		return
	}

	// Validasi pecahan desimal (hanya boleh 2 angka dibelakang koma)
	if math.Mod(req.Amount*100, 1) != 0 {
		utils.RespondError(c, http.StatusBadRequest, errors.New("amount can only have up to 2 decimal places"))
		return
	}

	db := utils.GetDB()

	// Ambil order dari database
	var order models.Order
	if err := db.Preload("OrderItems.Menu").Preload("Customer").First(&order, req.OrderID).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, errors.New("order not found"))
		return
	}

	// Generate payment ID unik
	paymentUUID := uuid.New().String()

	// Set default expired time (15 menit dari sekarang)
	expiredAt := time.Now().Add(15 * time.Minute)

	payment := models.Payment{
		OrderID:       req.OrderID,
		Amount:        req.Amount,
		Status:        "pending",
		PaymentMethod: req.PaymentMethod,
		ReferenceID:   req.ReferenceID,
		ExpiredAt:     &expiredAt, // Set default expired time sebagai pointer
	}

	// Log payment request
	utils.InfoLogger.Printf("Creating payment for order #%d with method %s, amount: %.2f",
		payment.OrderID, payment.PaymentMethod, payment.Amount)

	// Jika pembayaran tunai, langsung sukses
	if req.PaymentMethod == "cash" {
		payment.Status = "success"
		payment.ReferenceID = "CSH-" + paymentUUID
		if req.CashReceived > 0 {
			payment.Details = fmt.Sprintf("Cash received: %.2f, Change: %.2f", req.CashReceived, req.CashReceived-req.Amount)
		}
	} else if req.PaymentMethod == "qris" {
		// Untuk QRIS (Midtrans)
		midtransService := services.GetMidtransService()

		// Buat ID unik untuk transaksi
		transactionID := fmt.Sprintf("ORDER-%d-%s", order.ID, paymentUUID[:8])

		// Create transaction di Midtrans
		resp, err := midtransService.CreateTransaction(transactionID, float64(payment.Amount), order)
		if err != nil {
			utils.ErrorLogger.Printf("Failed to create QRIS transaction: %v", err)
			utils.RespondError(c, http.StatusInternalServerError, err)
			return
		}

		// Log response untuk debugging
		utils.InfoLogger.Printf("Midtrans response for order #%d: %+v", order.ID, resp)

		// Update payment dengan data dari Midtrans
		payment.ReferenceID = resp.TransactionID
		payment.QRCode = resp.QRCodeURL // QRIS data string

		// Prioritaskan URL dari actions jika tersedia
		if len(resp.Actions) > 0 {
			for _, action := range resp.Actions {
				if action.Name == "generate-qr-code" || action.Name == "display-qr-code" {
					payment.QRImageURL = action.URL
					break
				}
			}
		}

		// Jika tidak ada URL dari actions, generate dari QR code data
		if payment.QRImageURL == "" && payment.QRCode != "" {
			payment.QRImageURL = midtransService.GenerateQRImageURL(payment.QRCode)
		}

		// Jika masih tidak ada QR image URL, coba gunakan transaction ID
		if payment.QRImageURL == "" && resp.TransactionID != "" {
			payment.QRImageURL = midtransService.GenerateQRImageURL(resp.TransactionID)
		}

		// Jika ada expiry_time dari Midtrans, gunakan itu
		if resp.ExpiryTime != "" {
			midtransExpiry, err := time.Parse("2006-01-02 15:04:05", resp.ExpiryTime)
			if err == nil {
				payment.ExpiredAt = &midtransExpiry
				utils.InfoLogger.Printf("Using Midtrans expiry time: %s", midtransExpiry.Format(time.RFC3339))
			} else {
				utils.ErrorLogger.Printf("Error parsing Midtrans expiry time: %v, using default", err)
			}
		}

		utils.InfoLogger.Printf("QR code image URL: %s", payment.QRImageURL)
	}

	// Save payment ke database
	if err := db.Create(&payment).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	// Load order untuk response
	db.Preload("OrderItems.Menu").Preload("Customer").First(&order, req.OrderID)

	// Jika QRIS, tambahkan ke retry queue
	if req.PaymentMethod == "qris" && payment.Status == "pending" {
		paymentMonitor := services.NewPaymentMonitor(db)
		paymentMonitor.AddToRetryQueue(payment.ID)
	}

	// Kirim notifikasi via WebSocket untuk QRIS
	if req.PaymentMethod == "qris" {
		// Prepare data for websocket broadcast
		go sendPaymentEvent(payment, order)
	}

	// Log payment creation
	utils.InfoLogger.Printf("Payment %d created successfully with status: %s", payment.ID, payment.Status)

	utils.RespondJSON(c, http.StatusOK, "Payment created successfully", gin.H{
		"payment":      payment,
		"order":        order,
		"qr_image_url": payment.QRImageURL,
	})
}

// VerifyPayment memverifikasi pembayaran
func VerifyPayment(c *gin.Context) {
	// Validasi role
	roleInterface, _ := c.Get("role")
	if roleInterface != "admin" && roleInterface != "staff" {
		utils.RespondError(c, http.StatusForbidden, ErrNoPermission)
		return
	}

	db := utils.GetDB()
	id := c.Param("id")

	var payment models.Payment
	if err := db.Preload("Order").First(&payment, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Payment not found",
		})
		return
	}

	// Periksa apakah pembayaran sudah diverifikasi
	if payment.Status == "success" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Payment already verified",
		})
		return
	}

	// Verifikasi pembayaran
	now := time.Now()
	payment.PaymentTime = &now
	payment.Status = "success"

	// Get user ID from JWT token
	userID, exists := c.Get("userId")
	if exists {
		uid := userID.(uint)
		payment.VerifiedBy = &uid
	}

	if err := db.Save(&payment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to verify payment: " + err.Error(),
		})
		return
	}

	// Update order status to paid
	order := payment.Order
	order.Status = "paid"
	if err := db.Save(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to update order status: " + err.Error(),
		})
		return
	}

	// Broadcast WebSocket event
	sendPaymentEvent(payment, order)

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Payment verified successfully",
		"data": gin.H{
			"payment": payment,
			"order":   order,
		},
	})
}

// sendPaymentEvent sends a payment event to WebSocket clients
func sendPaymentEvent(payment models.Payment, order models.Order) {
	// Ensure order is loaded
	db := utils.GetDB()
	if order.ID == 0 && payment.OrderID > 0 {
		if err := db.Preload("OrderItems").Preload("OrderItems.Menu").First(&order, payment.OrderID).Error; err != nil {
			fmt.Printf("Warning: Could not load order data for payment event: %v\n", err)
		}
	}

	// Broadcast appropriate event based on payment status
	eventType := "payment_update"
	if payment.Status == "success" {
		eventType = "payment_success"
	} else if payment.Status == "pending" {
		eventType = "payment_pending"
	} else if payment.Status == "failed" {
		eventType = "payment_failed"
	} else if payment.Status == "expired" {
		eventType = "payment_expired"
	}

	// Prepare event data in a consistent format for all payment events
	eventData := map[string]interface{}{
		"payment": payment,
		"order":   order,
	}

	// Broadcast event based on type
	switch eventType {
	case "payment_success":
		kds.BroadcastPaymentSuccess(payment)
		kds.BroadcastOrderUpdate(order)
		kds.BroadcastStaffNotification(fmt.Sprintf("Payment successful for Order #%d", order.ID))
	case "payment_pending":
		kds.BroadcastPaymentPending(payment)
	case "payment_failed":
		kds.BroadcastPaymentFailed(payment)
	case "payment_expired":
		kds.BroadcastPaymentExpired(payment)
	}

	// Also broadcast to admin dashboard clients
	kds.BroadcastMessage(kds.Message{
		Event: eventType,
		Data:  eventData,
	})
}

// DeletePayment menghapus pembayaran
func DeletePayment(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)

	if err := utils.GetDB().Delete(&models.Payment{}, id).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}
	utils.RespondJSON(c, http.StatusOK, "Payment deleted", gin.H{"payment_id": id})
}

// handlePaymentCallback menangani callback dari payment gateway untuk QRIS
func handlePaymentCallback(paymentID uint, status string) {
	fmt.Printf("Processing payment callback for payment ID: %d with status: %s\n", paymentID, status)

	var payment models.Payment
	if err := utils.GetDB().Preload("Order").First(&payment, paymentID).Error; err != nil {
		fmt.Printf("Error finding payment with ID %d: %v\n", paymentID, err)
		return
	}

	fmt.Printf("Found payment record: %+v\n", payment)

	tx := utils.GetDB().Begin()

	// Update payment status
	now := time.Now()
	payment.Status = status
	payment.PaymentTime = &now
	payment.UpdatedAt = now

	if err := tx.Save(&payment).Error; err != nil {
		fmt.Printf("Error updating payment status: %v\n", err)
		tx.Rollback()
		return
	}

	// Update order status based on payment status
	var order models.Order
	if err := tx.First(&order, payment.OrderID).Error; err != nil {
		fmt.Printf("Error finding order with ID %d: %v\n", payment.OrderID, err)
		tx.Rollback()
		return
	}

	fmt.Printf("Found order record: %+v\n", order)

	// Process based on payment status
	switch status {
	case "success":
		order.Status = "paid"
		order.UpdatedAt = now

		if err := tx.Save(&order).Error; err != nil {
			fmt.Printf("Error updating order to paid: %v\n", err)
			tx.Rollback()
			return
		}

		// Commit transaction
		if err := tx.Commit().Error; err != nil {
			fmt.Printf("Error committing transaction: %v\n", err)
			tx.Rollback()
			return
		}

		fmt.Printf("Payment successful for order #%d\n", order.ID)

		// Broadcast updates for successful payment
		kds.BroadcastPaymentSuccess(payment)
		kds.BroadcastOrderUpdate(order)
		kds.BroadcastKitchenUpdate(order)
		kds.BroadcastStaffNotification(fmt.Sprintf("Payment received for Order #%d", order.ID))

	case "expired", "failed", "cancelled":
		// For expired/failed/cancelled payments, we keep the order status as pending_payment
		// but we update the payment status

		if err := tx.Commit().Error; err != nil {
			fmt.Printf("Error committing transaction for %s payment: %v\n", status, err)
			tx.Rollback()
			return
		}

		fmt.Printf("Payment %s for order #%d\n", status, order.ID)

		// Broadcast payment status update
		if status == "expired" {
			kds.BroadcastPaymentExpired(payment)
		} else {
			kds.BroadcastPaymentFailed(payment)
		}

		kds.BroadcastStaffNotification(fmt.Sprintf("Payment %s for Order #%d", status, order.ID))

	default:
		// For pending status, just commit the transaction
		if err := tx.Commit().Error; err != nil {
			fmt.Printf("Error committing transaction for pending payment: %v\n", err)
			tx.Rollback()
			return
		}

		fmt.Printf("Payment status updated to %s for order #%d\n", status, order.ID)
		kds.BroadcastPaymentPending(payment)
	}
}

// validateMidtransSignature memvalidasi signature dari Midtrans
func validateMidtransSignature(req interface{}, signature string) bool {
	// Convert request to map for easier handling
	reqMap, ok := req.(map[string]interface{})
	if !ok {
		fmt.Printf("Failed to convert request to map\n")
		return false
	}

	// Get required fields
	orderID, ok := reqMap["order_id"].(string)
	if !ok {
		fmt.Printf("Missing or invalid order_id\n")
		return false
	}

	statusCode, ok := reqMap["status_code"].(string)
	if !ok {
		fmt.Printf("Missing or invalid status_code\n")
		return false
	}

	grossAmount, ok := reqMap["gross_amount"].(string)
	if !ok {
		fmt.Printf("Missing or invalid gross_amount\n")
		return false
	}

	// Create signature string according to Midtrans documentation
	// Format: order_id + status_code + gross_amount + server_key
	signatureString := orderID + statusCode + grossAmount + midtransConfig.ServerKey

	// Calculate SHA512 hash
	h := sha512.New()
	h.Write([]byte(signatureString))
	calculatedSignature := hex.EncodeToString(h.Sum(nil))

	// Compare signatures
	isValid := calculatedSignature == signature

	// Log validation details for debugging
	fmt.Printf("Signature validation:\n")
	fmt.Printf("Order ID: %s\n", orderID)
	fmt.Printf("Status Code: %s\n", statusCode)
	fmt.Printf("Gross Amount: %s\n", grossAmount)
	fmt.Printf("Received Signature: %s\n", signature)
	fmt.Printf("Calculated Signature: %s\n", calculatedSignature)
	fmt.Printf("Is Valid: %v\n", isValid)

	return isValid
}

// HandlePaymentCallback handles Midtrans payment callbacks
func HandlePaymentCallback(c *gin.Context) {
	// Read raw body for signature validation
	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"message": "Failed to read request body",
		})
		return
	}

	// Parse request body
	var request struct {
		OrderID           string `json:"order_id"`
		TransactionStatus string `json:"transaction_status"`
		StatusCode        string `json:"status_code"`
		GrossAmount       string `json:"gross_amount"`
		SignatureKey      string `json:"signature_key"`
	}

	if err := json.Unmarshal(body, &request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"message": "Failed to parse request body",
		})
		return
	}

	// Validate signature
	midtrans := services.GetMidtransService()
	if !midtrans.ValidateSignature(request.OrderID, request.StatusCode, request.GrossAmount, request.SignatureKey) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Invalid signature",
			"message": "The signature is invalid",
		})
		return
	}

	// Extract payment ID from order ID
	paymentID, err := strconv.ParseUint(strings.TrimPrefix(request.OrderID, "ORDER-"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid order ID",
			"message": "Failed to parse order ID",
		})
		return
	}

	// Get payment with lock
	var payment models.Payment
	db := utils.GetDB()
	tx := db.Begin()
	if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&payment, paymentID).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Payment not found",
			"message": "The specified payment does not exist",
		})
		return
	}

	// Validate amount
	expectedAmount := fmt.Sprintf("%.2f", payment.Amount)
	if request.GrossAmount != expectedAmount {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid amount",
			"message": "The payment amount does not match",
		})
		return
	}

	// Map Midtrans status to our status
	var status string
	switch request.TransactionStatus {
	case "settlement", "capture":
		status = "success"
	case "pending", "authorize":
		status = "pending"
	case "deny", "cancel", "expire", "failure":
		status = "failed"
	default:
		status = "unknown"
	}

	// Update payment status
	payment.Status = status
	payment.UpdatedAt = time.Now()
	if err := tx.Save(&payment).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Database error",
			"message": "Failed to update payment status",
		})
		return
	}

	// Update order status if payment is successful
	if status == "success" {
		var order models.Order
		if err := tx.First(&order, payment.OrderID).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Order not found",
				"message": "The specified order does not exist",
			})
			return
		}

		order.Status = "paid"
		order.UpdatedAt = time.Now()
		if err := tx.Save(&order).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Database error",
				"message": "Failed to update order status",
			})
			return
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Database error",
			"message": "Failed to commit transaction",
		})
		return
	}

	// Add to retry queue if pending
	if status == "pending" {
		monitor := services.NewPaymentMonitor(utils.GetDB())
		monitor.AddToRetryQueue(payment.ID)
	}

	// Create notification for staff
	notification := models.Notification{
		Title:   "Payment Status Update",
		Message: fmt.Sprintf("Payment for order %d has been %s", payment.OrderID, status),
		Type:    "payment",
		Status:  "unread",
	}
	db.Create(&notification)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Payment status updated successfully",
	})
}

// GetMidtransConfig returns the Midtrans configuration for client-side
func GetMidtransConfig(c *gin.Context) {
	// Only return client-safe config parameters
	clientConfig := gin.H{
		"client_key":     os.Getenv("MIDTRANS_CLIENT_KEY"),
		"is_production":  os.Getenv("MIDTRANS_ENV") == "production",
		"merchant_id":    os.Getenv("MIDTRANS_MERCHANT_ID"),
		"merchant_name":  os.Getenv("MIDTRANS_MERCHANT_NAME"),
		"merchant_email": os.Getenv("MIDTRANS_MERCHANT_EMAIL"),
		"merchant_phone": os.Getenv("MIDTRANS_MERCHANT_PHONE"),
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  true,
		"message": "Midtrans configuration",
		"data":    clientConfig,
	})
}

// CheckPaymentStatus memeriksa status pembayaran di Midtrans dan memperbarui di database
func CheckPaymentStatus(c *gin.Context) {
	db := utils.GetDB()
	id := c.Param("payment_id")

	utils.InfoLogger.Printf("Checking payment status for payment ID: %s", id)

	var payment models.Payment
	if err := db.First(&payment, id).Error; err != nil {
		utils.ErrorLogger.Printf("Payment not found: %v", err)
		utils.RespondError(c, http.StatusNotFound, errors.New("payment not found"))
		return
	}

	utils.InfoLogger.Printf("Found payment: ID=%d, Method=%s, Status=%s, RefID=%s",
		payment.ID, payment.PaymentMethod, payment.Status, payment.ReferenceID)

	// Hanya periksa pembayaran QRIS dengan status pending
	if payment.PaymentMethod != "qris" || payment.Status != "pending" {
		utils.InfoLogger.Printf("Payment cannot be checked: method=%s, status=%s",
			payment.PaymentMethod, payment.Status)
		utils.RespondError(c, http.StatusBadRequest, errors.New("can only check pending qris payments"))
		return
	}

	// Dapatkan order ID dari referenceID payment
	orderRefID := payment.ReferenceID
	if orderRefID == "" {
		utils.ErrorLogger.Printf("Payment reference ID is empty for payment ID: %s", id)
		utils.RespondError(c, http.StatusBadRequest, errors.New("payment reference ID is empty"))
		return
	}

	// Jika referenceID mengandung prefix ORDER-, kita perlu mengekstrak ID sebenarnya
	var transactionID string
	if strings.HasPrefix(orderRefID, "ORDER-") {
		// Format: ORDER-<order_id>-<uuid>
		parts := strings.Split(orderRefID, "-")
		if len(parts) >= 3 {
			// Kita ambil uuid di bagian terakhir sebagai transaction ID
			transactionID = parts[len(parts)-1]
			utils.InfoLogger.Printf("Extracted transaction ID %s from reference ID %s",
				transactionID, orderRefID)
		} else {
			transactionID = orderRefID
		}
	} else {
		transactionID = orderRefID
	}

	utils.InfoLogger.Printf("Checking transaction status in Midtrans for transaction ID: %s", transactionID)

	// Cek status di Midtrans
	midtransService := services.GetMidtransService()
	status, err := midtransService.CheckTransactionStatus(transactionID)
	if err != nil {
		utils.ErrorLogger.Printf("Error checking Midtrans status: %v", err)
		utils.RespondError(c, http.StatusInternalServerError, fmt.Errorf("error checking midtrans status: %v", err))
		return
	}

	utils.InfoLogger.Printf("Midtrans transaction %s status: %s", transactionID, status)

	// Update payment status jika status berbeda
	if status != payment.Status {
		utils.InfoLogger.Printf("Status changed from %s to %s, updating database", payment.Status, status)
		tx := db.Begin()

		now := time.Now()
		payment.Status = status
		payment.UpdatedAt = now

		// Jika payment sukses, set payment time
		if status == "success" {
			payment.PaymentTime = &now
			utils.InfoLogger.Printf("Payment successful, setting payment time to %s", now.Format(time.RFC3339))
		}

		if err := tx.Save(&payment).Error; err != nil {
			tx.Rollback()
			utils.ErrorLogger.Printf("Failed to update payment status: %v", err)
			utils.RespondError(c, http.StatusInternalServerError, fmt.Errorf("failed to update payment status: %v", err))
			return
		}

		// Update order status berdasarkan status pembayaran
		if status == "success" {
			var order models.Order
			if err := tx.First(&order, payment.OrderID).Error; err != nil {
				tx.Rollback()
				utils.ErrorLogger.Printf("Failed to find order %d: %v", payment.OrderID, err)
				utils.RespondError(c, http.StatusInternalServerError, fmt.Errorf("failed to find order: %v", err))
				return
			}

			utils.InfoLogger.Printf("Updating order #%d status from %s to paid", order.ID, order.Status)
			order.Status = "paid"
			order.UpdatedAt = now

			if err := tx.Save(&order).Error; err != nil {
				tx.Rollback()
				utils.ErrorLogger.Printf("Failed to update order status: %v", err)
				utils.RespondError(c, http.StatusInternalServerError, fmt.Errorf("failed to update order status: %v", err))
				return
			}
		}

		if err := tx.Commit().Error; err != nil {
			utils.ErrorLogger.Printf("Failed to commit transaction: %v", err)
			utils.RespondError(c, http.StatusInternalServerError, fmt.Errorf("failed to commit transaction: %v", err))
			return
		}

		utils.InfoLogger.Printf("Successfully updated payment status to %s", status)

		// Reload payment dan order untuk notifikasi
		db.Preload("Order").First(&payment, id)

		// Broadcast payment update
		utils.InfoLogger.Printf("Broadcasting payment event for payment %d with status %s", payment.ID, status)
		sendPaymentEvent(payment, payment.Order)
	} else {
		utils.InfoLogger.Printf("Payment status unchanged (%s), no update needed", status)
	}

	utils.RespondJSON(c, http.StatusOK, "Payment status checked", gin.H{
		"payment_id":     payment.ID,
		"status":         status,
		"reference_id":   orderRefID,
		"transaction_id": transactionID,
		"was_updated":    status != payment.Status,
	})
}

// CheckOrderPaymentStatus memeriksa status pembayaran untuk order tertentu
func CheckOrderPaymentStatus(c *gin.Context) {
	db := utils.GetDB()
	orderID := c.Param("order_id")

	utils.InfoLogger.Printf("Checking payment status for order ID: %s", orderID)

	// Temukan pembayaran terbaru untuk order ini
	var payments []models.Payment
	if err := db.Where("order_id = ?", orderID).Order("created_at DESC").Find(&payments).Error; err != nil {
		utils.ErrorLogger.Printf("Failed to find payments for order ID %s: %v", orderID, err)
		utils.RespondError(c, http.StatusInternalServerError, fmt.Errorf("failed to find payments: %v", err))
		return
	}

	if len(payments) == 0 {
		utils.InfoLogger.Printf("No payments found for order ID: %s", orderID)
		utils.RespondError(c, http.StatusNotFound, errors.New("no payments found for this order"))
		return
	}

	// Ambil pembayaran terakhir (terbaru)
	payment := payments[0]
	utils.InfoLogger.Printf("Found latest payment: ID=%d, Method=%s, Status=%s, RefID=%s",
		payment.ID, payment.PaymentMethod, payment.Status, payment.ReferenceID)

	// Jika pembayaran bukan QRIS, kembalikan status saat ini
	if payment.PaymentMethod != "qris" {
		utils.RespondJSON(c, http.StatusOK, "Payment status retrieved", gin.H{
			"payment_id":   payment.ID,
			"order_id":     payment.OrderID,
			"status":       payment.Status,
			"reference_id": payment.ReferenceID,
			"was_updated":  false,
		})
		return
	}

	// Hanya lanjutkan pemeriksaan untuk payment yang pending
	if payment.Status != "pending" {
		utils.InfoLogger.Printf("Payment %d status is %s, not checking with Midtrans",
			payment.ID, payment.Status)
		utils.RespondJSON(c, http.StatusOK, "Payment status retrieved", gin.H{
			"payment_id":   payment.ID,
			"order_id":     payment.OrderID,
			"status":       payment.Status,
			"reference_id": payment.ReferenceID,
			"was_updated":  false,
		})
		return
	}

	// Dapatkan order ID dari referenceID payment
	orderRefID := payment.ReferenceID
	if orderRefID == "" {
		utils.ErrorLogger.Printf("Payment reference ID is empty for payment ID: %d", payment.ID)
		utils.RespondError(c, http.StatusBadRequest, errors.New("payment reference ID is empty"))
		return
	}

	// Jika referenceID mengandung prefix ORDER-, kita perlu mengekstrak ID sebenarnya
	var transactionID string
	if strings.HasPrefix(orderRefID, "ORDER-") {
		// Format: ORDER-<order_id>-<uuid>
		parts := strings.Split(orderRefID, "-")
		if len(parts) >= 3 {
			// Kita ambil uuid di bagian terakhir sebagai transaction ID
			transactionID = parts[len(parts)-1]
			utils.InfoLogger.Printf("Extracted transaction ID %s from reference ID %s",
				transactionID, orderRefID)
		} else {
			transactionID = orderRefID
		}
	} else {
		transactionID = orderRefID
	}

	utils.InfoLogger.Printf("Checking transaction status in Midtrans for transaction ID: %s", transactionID)

	// Cek status di Midtrans
	midtransService := services.GetMidtransService()
	status, err := midtransService.CheckTransactionStatus(transactionID)
	if err != nil {
		utils.ErrorLogger.Printf("Error checking Midtrans status: %v", err)
		utils.RespondError(c, http.StatusInternalServerError, fmt.Errorf("error checking midtrans status: %v", err))
		return
	}

	utils.InfoLogger.Printf("Midtrans transaction %s status: %s", transactionID, status)

	// Update payment status jika status berbeda
	wasUpdated := false
	if status != payment.Status {
		wasUpdated = true
		utils.InfoLogger.Printf("Status changed from %s to %s, updating database", payment.Status, status)
		tx := db.Begin()

		now := time.Now()
		payment.Status = status
		payment.UpdatedAt = now

		// Jika payment sukses, set payment time
		if status == "success" {
			payment.PaymentTime = &now
			utils.InfoLogger.Printf("Payment successful, setting payment time to %s", now.Format(time.RFC3339))
		}

		if err := tx.Save(&payment).Error; err != nil {
			tx.Rollback()
			utils.ErrorLogger.Printf("Failed to update payment status: %v", err)
			utils.RespondError(c, http.StatusInternalServerError, fmt.Errorf("failed to update payment status: %v", err))
			return
		}

		// Update order status berdasarkan status pembayaran
		if status == "success" {
			var order models.Order
			if err := tx.First(&order, payment.OrderID).Error; err != nil {
				tx.Rollback()
				utils.ErrorLogger.Printf("Failed to find order %d: %v", payment.OrderID, err)
				utils.RespondError(c, http.StatusInternalServerError, fmt.Errorf("failed to find order: %v", err))
				return
			}

			utils.InfoLogger.Printf("Updating order #%d status from %s to paid", order.ID, order.Status)
			order.Status = "paid"
			order.UpdatedAt = now

			if err := tx.Save(&order).Error; err != nil {
				tx.Rollback()
				utils.ErrorLogger.Printf("Failed to update order status: %v", err)
				utils.RespondError(c, http.StatusInternalServerError, fmt.Errorf("failed to update order status: %v", err))
				return
			}
		}

		if err := tx.Commit().Error; err != nil {
			utils.ErrorLogger.Printf("Failed to commit transaction: %v", err)
			utils.RespondError(c, http.StatusInternalServerError, fmt.Errorf("failed to commit transaction: %v", err))
			return
		}

		utils.InfoLogger.Printf("Successfully updated payment status to %s", status)

		// Reload payment dan order untuk notifikasi
		db.Preload("Order").First(&payment, payment.ID)

		// Broadcast payment update
		utils.InfoLogger.Printf("Broadcasting payment event for payment %d with status %s", payment.ID, status)
		sendPaymentEvent(payment, payment.Order)
	} else {
		utils.InfoLogger.Printf("Payment status unchanged (%s), no update needed", status)
	}

	utils.RespondJSON(c, http.StatusOK, "Payment status checked", gin.H{
		"payment_id":     payment.ID,
		"order_id":       payment.OrderID,
		"status":         status,
		"reference_id":   orderRefID,
		"transaction_id": transactionID,
		"was_updated":    wasUpdated,
	})
}
