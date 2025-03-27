package services

import (
	"fmt"
	"log"
	"time"

	"github.com/yeremiapane/restaurant-app/models"
	"gorm.io/gorm"
)

// Status pembayaran
const (
	PaymentStatusPending   = "pending"
	PaymentStatusSuccess   = "success"
	PaymentStatusFailed    = "failed"
	PaymentStatusExpired   = "expired"
	PaymentStatusCancelled = "cancelled"
)

// Status order
const (
	OrderStatusPendingPayment = "pending_payment"
	OrderStatusPaid           = "paid"
	OrderStatusCancelled      = "cancelled"
	OrderStatusProcessing     = "processing"
	OrderStatusReady          = "ready"
	OrderStatusServed         = "served"
	OrderStatusCompleted      = "completed"
)

// PaymentService menangani operasi pembayaran
type PaymentService struct {
	db *gorm.DB
}

// NewPaymentService membuat instance baru PaymentService
func NewPaymentService(db *gorm.DB) *PaymentService {
	return &PaymentService{
		db: db,
	}
}

// CreatePayment membuat pembayaran baru
func (s *PaymentService) CreatePayment(payment *models.Payment) error {
	result := s.db.Create(payment)
	return result.Error
}

// GetPaymentByID mendapatkan pembayaran berdasarkan ID
func (s *PaymentService) GetPaymentByID(id uint) (*models.Payment, error) {
	var payment models.Payment
	result := s.db.First(&payment, id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &payment, nil
}

// GetPaymentByOrderID mendapatkan pembayaran berdasarkan OrderID
func (s *PaymentService) GetPaymentByOrderID(orderID uint) (*models.Payment, error) {
	var payment models.Payment
	result := s.db.Where("order_id = ?", orderID).First(&payment)
	if result.Error != nil {
		return nil, result.Error
	}
	return &payment, nil
}

// UpdatePaymentStatus mengupdate status pembayaran
func (s *PaymentService) UpdatePaymentStatus(paymentID uint, status string) error {
	// Begin transaction
	tx := s.db.Begin()

	// Update payment status
	var payment models.Payment
	if err := tx.First(&payment, paymentID).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to find payment: %w", err)
	}

	payment.Status = status
	if err := tx.Save(&payment).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update payment status: %w", err)
	}

	// Update order status
	var order models.Order
	if err := tx.First(&order, payment.OrderID).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to find order: %w", err)
	}

	switch status {
	case PaymentStatusSuccess:
		order.Status = OrderStatusPaid
	case PaymentStatusFailed, PaymentStatusExpired, PaymentStatusCancelled:
		order.Status = OrderStatusCancelled
	}

	if err := tx.Save(&order).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update order status: %w", err)
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// PaymentTimeoutChecker adalah goroutine yang memeriksa payment yang sudah mendekati waktu expired
func (s *PaymentService) PaymentTimeoutChecker() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.CheckExpiredPayments()
		}
	}
}

// CheckExpiredPayments memeriksa pembayaran yang hampir kedaluwarsa dan mengupdate statusnya
func (s *PaymentService) CheckExpiredPayments() {
	payments := make([]*models.Payment, 0)

	// Get semua payment dengan status pending
	result := s.db.Where("status = ?", PaymentStatusPending).Find(&payments)
	if result.Error != nil {
		log.Printf("Error checking expired payments: %v", result.Error)
		return
	}

	now := time.Now()

	for _, payment := range payments {
		// Jika waktu expired sudah lewat, update status menjadi expired
		if payment.ExpiredAt != nil && !payment.ExpiredAt.IsZero() && now.After(*payment.ExpiredAt) {
			// Update status payment di database
			payment.Status = PaymentStatusExpired
			result := s.db.Save(payment)
			if result.Error != nil {
				log.Printf("Error updating expired payment %d: %v", payment.ID, result.Error)
				continue
			}

			// Update status order
			order := &models.Order{}
			err := s.db.First(order, payment.OrderID).Error
			if err != nil {
				log.Printf("Error finding order %d for expired payment: %v", payment.OrderID, err)
				continue
			}

			order.Status = OrderStatusCancelled
			err = s.db.Save(order).Error
			if err != nil {
				log.Printf("Error updating order status for expired payment: %v", err)
				continue
			}

			log.Printf("Payment %d expired and order %d cancelled", payment.ID, payment.OrderID)
		} else if payment.ExpiredAt != nil && !payment.ExpiredAt.IsZero() {
			// Jika payment mendekati waktu expired (10 menit sebelum expired)
			// Cek status di Midtrans untuk memastikan status terbaru
			tenMinutesBeforeExpiry := (*payment.ExpiredAt).Add(-10 * time.Minute)
			if now.After(tenMinutesBeforeExpiry) {
				// Check status di Midtrans
				midtransService := GetMidtransService()
				status, err := midtransService.CheckTransactionStatus(fmt.Sprintf("%d", payment.OrderID))
				if err != nil {
					log.Printf("Error checking transaction status for payment %d: %v", payment.ID, err)
					continue
				}

				// Jika status dari Midtrans berbeda dengan status di database, update
				if status != payment.Status {
					payment.Status = status
					result := s.db.Save(payment)
					if result.Error != nil {
						log.Printf("Error updating payment status from Midtrans: %v", result.Error)
						continue
					}

					// Update order status
					order := &models.Order{}
					err := s.db.First(order, payment.OrderID).Error
					if err != nil {
						log.Printf("Error finding order for payment status update: %v", err)
						continue
					}

					// Update order status based on payment status
					if status == PaymentStatusSuccess {
						order.Status = OrderStatusPaid
					} else if status == PaymentStatusExpired ||
						status == PaymentStatusFailed ||
						status == PaymentStatusCancelled {
						order.Status = OrderStatusCancelled
					}

					err = s.db.Save(order).Error
					if err != nil {
						log.Printf("Error updating order status based on payment: %v", err)
						continue
					}

					log.Printf("Updated payment %d status to %s from Midtrans", payment.ID, status)
				}
			}
		}
	}
}

// StartTimeoutChecker memulai goroutine untuk memeriksa transaksi yang expired
func (s *PaymentService) StartTimeoutChecker() {
	go s.PaymentTimeoutChecker()
	log.Println("Payment timeout checker started")
}
