package controllers

import (
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yeremiapane/restaurant-app/models"
	"github.com/yeremiapane/restaurant-app/utils"
	"gorm.io/gorm"
)

type ReceiptController struct {
	DB *gorm.DB
}

func NewReceiptController(db *gorm.DB) *ReceiptController {
	return &ReceiptController{DB: db}
}

// GenerateReceipt membuat struk pembayaran
func (rc *ReceiptController) GenerateReceipt(c *gin.Context) {
	paymentID := c.Param("payment_id")

	// Ambil data payment dengan relasi
	var payment models.Payment
	if err := rc.DB.Preload("Order").
		Preload("Order.OrderItems").
		Preload("Order.OrderItems.Menu").
		Preload("Order.Customer").
		Preload("Order.Customer.Table").
		First(&payment, paymentID).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, err)
		return
	}

	// Pastikan payment sudah success
	if payment.Status != "success" {
		utils.RespondError(c, http.StatusBadRequest, fmt.Errorf("payment belum selesai"))
		return
	}

	// Generate receipt number
	receiptNumber := fmt.Sprintf("RCP/%s/%06d",
		time.Now().Format("20060102"),
		payment.ID)

	// Buat receipt
	receipt := models.Receipt{
		OrderID:       payment.OrderID,
		PaymentID:     payment.ID,
		ReceiptNumber: receiptNumber,
		CreatedAt:     time.Now(),
	}

	if err := rc.DB.Create(&receipt).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	// Format data untuk struk dengan detail harga lengkap
	receiptData := struct {
		RestaurantInfo struct {
			Name    string `json:"name"`
			Address string `json:"address"`
			Phone   string `json:"phone"`
		} `json:"restaurant_info"`
		ReceiptInfo struct {
			Number      string    `json:"number"`
			DateTime    time.Time `json:"date_time"`
			TableNumber string    `json:"table_number"`
			Cashier     string    `json:"cashier"`
		} `json:"receipt_info"`
		OrderDetails struct {
			Items []struct {
				Name      string  `json:"name"`
				Quantity  int     `json:"quantity"`
				UnitPrice float64 `json:"unit_price"`
				Subtotal  float64 `json:"subtotal"`
				Notes     string  `json:"notes,omitempty"`
				Addons    []struct {
					Name     string  `json:"name"`
					Price    float64 `json:"price"`
					Quantity int     `json:"quantity"`
				} `json:"addons,omitempty"`
			} `json:"items"`
			PriceDetails struct {
				Subtotal      float64 `json:"subtotal"`
				ServiceCharge float64 `json:"service_charge"`
				Tax           float64 `json:"tax"`
				Total         float64 `json:"total"`
				RoundedTotal  float64 `json:"rounded_total"`
			} `json:"price_details"`
		} `json:"order_details"`
		PaymentDetails struct {
			Method     string  `json:"method"`
			Amount     float64 `json:"amount_paid"`
			Change     float64 `json:"change"`
			Time       string  `json:"time"`
			Status     string  `json:"status"`
			References string  `json:"references,omitempty"` // untuk QRIS/kartu
		} `json:"payment_details"`
		Footer struct {
			ThankYouNote string `json:"thank_you_note"`
			Terms        string `json:"terms"`
		} `json:"footer"`
	}{}

	// Isi informasi restoran
	receiptData.RestaurantInfo = struct {
		Name    string `json:"name"`
		Address string `json:"address"`
		Phone   string `json:"phone"`
	}{
		Name:    "Restaurant Name",
		Address: "Restaurant Address",
		Phone:   "Restaurant Phone",
	}

	// Isi informasi struk
	receiptData.ReceiptInfo = struct {
		Number      string    `json:"number"`
		DateTime    time.Time `json:"date_time"`
		TableNumber string    `json:"table_number"`
		Cashier     string    `json:"cashier"`
	}{
		Number:      receipt.ReceiptNumber,
		DateTime:    receipt.CreatedAt,
		TableNumber: payment.Order.Customer.Table.TableNumber,
		Cashier:     "Cashier Name", // Bisa diambil dari context user yang login
	}

	// Hitung detail harga
	var subtotal float64
	receiptData.OrderDetails.Items = make([]struct {
		Name      string  `json:"name"`
		Quantity  int     `json:"quantity"`
		UnitPrice float64 `json:"unit_price"`
		Subtotal  float64 `json:"subtotal"`
		Notes     string  `json:"notes,omitempty"`
		Addons    []struct {
			Name     string  `json:"name"`
			Price    float64 `json:"price"`
			Quantity int     `json:"quantity"`
		} `json:"addons,omitempty"`
	}, len(payment.Order.OrderItems))

	// Isi detail item dan harga
	for i, item := range payment.Order.OrderItems {
		itemSubtotal := float64(item.Quantity) * item.Price
		subtotal += itemSubtotal

		// Struktur item dengan harga
		receiptData.OrderDetails.Items[i] = struct {
			Name      string  `json:"name"`
			Quantity  int     `json:"quantity"`
			UnitPrice float64 `json:"unit_price"`
			Subtotal  float64 `json:"subtotal"`
			Notes     string  `json:"notes,omitempty"`
			Addons    []struct {
				Name     string  `json:"name"`
				Price    float64 `json:"price"`
				Quantity int     `json:"quantity"`
			} `json:"addons,omitempty"`
		}{
			Name:      item.Menu.Name,
			Quantity:  item.Quantity,
			UnitPrice: item.Price,
			Subtotal:  itemSubtotal,
			Notes:     item.Notes,
		}

		// Jika ada addon, tambahkan ke item
		if item.ParentItemID != nil {
			// Logic untuk addon
		}
	}

	// Hitung detail harga final
	serviceCharge := subtotal * 0.05 // 5% service charge
	tax := subtotal * 0.10           // 10% tax
	total := subtotal + serviceCharge + tax
	roundedTotal := math.Ceil(total/1000) * 1000 // Pembulatan ke atas (1000)

	receiptData.OrderDetails.PriceDetails = struct {
		Subtotal      float64 `json:"subtotal"`
		ServiceCharge float64 `json:"service_charge"`
		Tax           float64 `json:"tax"`
		Total         float64 `json:"total"`
		RoundedTotal  float64 `json:"rounded_total"`
	}{
		Subtotal:      subtotal,
		ServiceCharge: serviceCharge,
		Tax:           tax,
		Total:         total,
		RoundedTotal:  roundedTotal,
	}

	// Isi detail pembayaran
	change := payment.Amount - roundedTotal
	if change < 0 {
		change = 0
	}

	receiptData.PaymentDetails = struct {
		Method     string  `json:"method"`
		Amount     float64 `json:"amount_paid"`
		Change     float64 `json:"change"`
		Time       string  `json:"time"`
		Status     string  `json:"status"`
		References string  `json:"references,omitempty"`
	}{
		Method: payment.PaymentMethod,
		Amount: payment.Amount,
		Change: change,
		Time:   payment.PaymentTime.Format("15:04:05"),
		Status: payment.Status,
		References: func() string {
			if payment.PaymentMethod == "qris" {
				return "QRIS REF: xxx"
			}
			return ""
		}(),
	}

	// Isi footer
	receiptData.Footer = struct {
		ThankYouNote string `json:"thank_you_note"`
		Terms        string `json:"terms"`
	}{
		ThankYouNote: "Terima kasih atas kunjungan Anda!",
		Terms:        "Struk ini adalah bukti pembayaran yang sah",
	}

	utils.RespondJSON(c, http.StatusOK, "Receipt generated", receiptData)
}

// GetReceiptByID mengambil detail struk berdasarkan ID
func (rc *ReceiptController) GetReceiptByID(c *gin.Context) {
	receiptID := c.Param("receipt_id")

	var receipt models.Receipt
	if err := rc.DB.Preload("Order").Preload("ReceiptItems").First(&receipt, receiptID).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, err)
		return
	}

	utils.RespondJSON(c, http.StatusOK, "Receipt detail", receipt)
}
