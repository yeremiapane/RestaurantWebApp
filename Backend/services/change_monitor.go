package services

import (
	"log"
	"time"

	"github.com/yeremiapane/restaurant-app/kds"
	"github.com/yeremiapane/restaurant-app/models"
	"gorm.io/gorm"
)

type ChangeMonitor struct {
	DB       *gorm.DB
	StopChan chan struct{}
	Interval time.Duration
}

type DBChange struct {
	ID         int64     `gorm:"column:id"`
	TableName  string    `gorm:"column:table_name"`
	RecordID   int64     `gorm:"column:record_id"`
	ActionType string    `gorm:"column:action_type"`
	ChangedAt  time.Time `gorm:"column:changed_at"`
	Processed  bool      `gorm:"column:processed"`
}

func NewChangeMonitor(db *gorm.DB) *ChangeMonitor {
	return &ChangeMonitor{
		DB:       db,
		StopChan: make(chan struct{}),
		Interval: 1 * time.Second,
	}
}

func (cm *ChangeMonitor) Start() {
	go func() {
		ticker := time.NewTicker(cm.Interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				cm.checkChanges()
			case <-cm.StopChan:
				return
			}
		}
	}()
}

func (cm *ChangeMonitor) Stop() {
	close(cm.StopChan)
}

func (cm *ChangeMonitor) checkChanges() {
	var changes []DBChange

	// Gunakan transaction untuk mencegah race condition
	tx := cm.DB.Begin()

	// Log jumlah perubahan yang belum diproses
	var count int64
	tx.Model(&DBChange{}).Where("processed = ?", false).Count(&count)
	if count > 0 {
		log.Printf("Found %d unprocessed changes", count)
	}

	// Ambil perubahan yang belum diproses
	if err := tx.Where("processed = ?", false).
		Order("changed_at ASC").
		Limit(100).
		Find(&changes).Error; err != nil {
		tx.Rollback()
		log.Printf("Error fetching changes: %v", err)
		return
	}

	for _, change := range changes {
		log.Printf("Processing change: table=%s, action=%s, record_id=%d",
			change.TableName, change.ActionType, change.RecordID)

		// Proses berdasarkan tipe tabel
		switch change.TableName {
		case "tables":
			cm.processTableChange(change)
		case "orders":
			cm.processOrderChange(change)
		case "payments":
			cm.processPaymentChange(change)
		case "receipts":
			cm.processReceiptChange(change)
		}

		// Mark sebagai processed
		if err := tx.Model(&DBChange{}).
			Where("id = ?", change.ID).
			Update("processed", true).Error; err != nil {
			tx.Rollback()
			log.Printf("Error marking change as processed: %v", err)
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		log.Printf("Error committing transaction: %v", err)
		tx.Rollback()
		return
	}

	if len(changes) > 0 {
		log.Printf("Successfully processed %d changes", len(changes))
	}
}

func (cm *ChangeMonitor) processTableChange(change DBChange) {
	var table models.Table

	if change.ActionType != "DELETE" {
		if err := cm.DB.First(&table, change.RecordID).Error; err != nil {
			log.Printf("Error fetching table: %v", err)
			return
		}
		log.Printf("Found table: %+v", table)
	}

	switch change.ActionType {
	case "INSERT":
		log.Printf("Broadcasting table create: %+v", table)
		kds.BroadcastTableCreate(table)
	case "UPDATE":
		log.Printf("Broadcasting table update: %+v", table)
		kds.BroadcastTableUpdate(table)
	case "DELETE":
		log.Printf("Broadcasting table delete for ID: %d", change.RecordID)
		kds.BroadcastTableDelete(models.Table{ID: uint(change.RecordID)})
	}
}

func (cm *ChangeMonitor) processOrderChange(change DBChange) {
	var order models.Order

	if change.ActionType != "DELETE" {
		if err := cm.DB.First(&order, change.RecordID).Error; err != nil {
			log.Printf("Error fetching order: %v", err)
			return
		}
	}

	switch change.ActionType {
	case "INSERT", "UPDATE":
		kds.BroadcastOrderUpdate(order)
	}
}

func (cm *ChangeMonitor) processPaymentChange(change DBChange) {
	var payment models.Payment

	if change.ActionType != "DELETE" {
		if err := cm.DB.First(&payment, change.RecordID).Error; err != nil {
			log.Printf("Error fetching payment: %v", err)
			return
		}
	}

	switch change.ActionType {
	case "INSERT":
		kds.BroadcastPaymentPending(payment)
	case "UPDATE":
		if payment.Status == "SUCCESS" {
			kds.BroadcastPaymentSuccess(payment)
		}
		kds.BroadcastPaymentUpdate(payment, models.Order{ID: payment.OrderID})
	}
}

func (cm *ChangeMonitor) processReceiptChange(change DBChange) {
	var receipt models.Receipt

	if change.ActionType != "DELETE" {
		if err := cm.DB.First(&receipt, change.RecordID).Error; err != nil {
			log.Printf("Error fetching receipt: %v", err)
			return
		}
	}

	switch change.ActionType {
	case "INSERT":
		kds.BroadcastGenerated(receipt)
	}
}
