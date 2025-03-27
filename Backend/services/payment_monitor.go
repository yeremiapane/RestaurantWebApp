package services

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/yeremiapane/restaurant-app/models"
	"gorm.io/gorm"
)

// PaymentMetrics menyimpan metrik terkait pembayaran
type PaymentMetrics struct {
	TotalTransactions  int64
	SuccessfulPayments int64
	FailedPayments     int64
	PendingPayments    int64
	AvgResponseTime    int64 // dalam milisecond
}

// PaymentMonitor menangani monitoring dan retry untuk pembayaran
type PaymentMonitor struct {
	db            *gorm.DB
	metrics       PaymentMetrics
	retryQueue    []uint
	retryInterval time.Duration
	mutex         sync.Mutex
}

// NewPaymentMonitor membuat instance baru PaymentMonitor
func NewPaymentMonitor(db *gorm.DB) *PaymentMonitor {
	return &PaymentMonitor{
		db:            db,
		metrics:       PaymentMetrics{},
		retryQueue:    make([]uint, 0),
		retryInterval: 5 * time.Minute,
		mutex:         sync.Mutex{},
	}
}

// Start memulai goroutine untuk monitoring dan retry
func (pm *PaymentMonitor) Start() {
	go pm.processRetryQueue()
	log.Println("Payment monitor started")
}

// AddToRetryQueue menambahkan payment ID ke dalam antrian retry
func (pm *PaymentMonitor) AddToRetryQueue(paymentID uint) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	// Periksa apakah payment ID sudah ada di antrian
	for _, id := range pm.retryQueue {
		if id == paymentID {
			return
		}
	}

	pm.retryQueue = append(pm.retryQueue, paymentID)
	log.Printf("Added payment %d to retry queue", paymentID)
}

// processRetryQueue memproses antrian retry
func (pm *PaymentMonitor) processRetryQueue() {
	ticker := time.NewTicker(pm.retryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pm.mutex.Lock()
			if len(pm.retryQueue) == 0 {
				pm.mutex.Unlock()
				continue
			}

			log.Printf("Processing retry queue with %d payments", len(pm.retryQueue))

			// Ambil payment IDs dari antrian
			queue := make([]uint, len(pm.retryQueue))
			copy(queue, pm.retryQueue)

			// Reset antrian
			pm.retryQueue = make([]uint, 0)
			pm.mutex.Unlock()

			// Proses setiap payment dalam antrian
			for _, paymentID := range queue {
				pm.retryPayment(paymentID)
			}
		}
	}
}

// retryPayment mencoba kembali pembayaran yang gagal
func (pm *PaymentMonitor) retryPayment(paymentID uint) {
	payment := &models.Payment{}
	if err := pm.db.First(payment, paymentID).Error; err != nil {
		log.Printf("Error finding payment %d for retry: %v", paymentID, err)
		return
	}

	// Jika payment sudah berhasil, tidak perlu retry
	if payment.Status == PaymentStatusSuccess {
		log.Printf("Payment %d already succeeded, no retry needed", paymentID)
		return
	}

	// Jika payment sudah expired, tidak perlu retry
	if payment.Status == PaymentStatusExpired {
		log.Printf("Payment %d already expired, no retry needed", paymentID)
		return
	}

	// Jika payment sudah dibatalkan, tidak perlu retry
	if payment.Status == PaymentStatusCancelled {
		log.Printf("Payment %d already cancelled, no retry needed", paymentID)
		return
	}

	// Periksa status di Midtrans
	midtransService := GetMidtransService()
	status, err := midtransService.CheckTransactionStatus(fmt.Sprintf("%d", payment.OrderID))
	if err != nil {
		log.Printf("Error checking transaction status for payment %d: %v", paymentID, err)
		// Re-add to retry queue jika masih gagal
		pm.AddToRetryQueue(paymentID)
		return
	}

	// Update status payment berdasarkan respons dari Midtrans
	if status != payment.Status {
		payment.Status = status
		if err := pm.db.Save(payment).Error; err != nil {
			log.Printf("Error updating payment status: %v", err)
			pm.AddToRetryQueue(paymentID)
			return
		}

		log.Printf("Updated payment %d status to %s from retry", paymentID, status)

		// Update metrik
		pm.updateMetrics(status)
	}
}

// updateMetrics mengupdate metrik berdasarkan status
func (pm *PaymentMonitor) updateMetrics(status string) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.metrics.TotalTransactions++

	switch status {
	case PaymentStatusSuccess:
		pm.metrics.SuccessfulPayments++
	case PaymentStatusFailed, PaymentStatusExpired, PaymentStatusCancelled:
		pm.metrics.FailedPayments++
	case PaymentStatusPending:
		pm.metrics.PendingPayments++
	}
}

// GetMetrics mengembalikan metrik pembayaran saat ini
func (pm *PaymentMonitor) GetMetrics() PaymentMetrics {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	return pm.metrics
}

// UpdatePaymentStatus mengupdate status pembayaran di database
func (pm *PaymentMonitor) UpdatePaymentStatus(paymentID uint, status string) error {
	payment := &models.Payment{}
	if err := pm.db.First(payment, paymentID).Error; err != nil {
		return err
	}

	payment.Status = status
	if err := pm.db.Save(payment).Error; err != nil {
		return err
	}

	// Update metrik
	pm.updateMetrics(status)

	return nil
}
