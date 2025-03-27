package models

import (
	"time"
)

// Payment represents a payment transaction for an order
type Payment struct {
	ID            uint       `json:"id" gorm:"primaryKey"`
	OrderID       uint       `json:"order_id"`
	Order         Order      `json:"order" gorm:"foreignKey:OrderID"`
	Amount        float64    `json:"amount"`
	Status        string     `json:"status" gorm:"type:enum('pending','success','failed','expired');default:'pending'"`
	PaymentMethod string     `json:"payment_method" gorm:"type:enum('cash','qris','bank_transfer');default:'cash'"`
	PaymentType   string     `json:"payment_type"`
	ReferenceID   string     `json:"reference_id"`
	QRCode        string     `json:"qr_code"`       // Raw QR code data for QRIS
	QRImageURL    string     `json:"qr_image_url"`  // URL to QR code image
	PaymentURL    string     `json:"payment_url"`   // URL for redirect payment methods
	Details       string     `json:"details"`       // Additional payment details in JSON
	CashReceived  float64    `json:"cash_received"` // Amount of cash received for cash payments
	Change        float64    `json:"change"`        // Change amount for cash payments
	PaymentTime   *time.Time `json:"payment_time"`  // Time when payment was processed
	ExpiredAt     *time.Time `json:"expired_at"`    // Time when payment will expire (nullable)
	VerifiedBy    *uint      `json:"verified_by"`   // Staff who verified the payment
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}
