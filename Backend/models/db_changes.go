package models

import (
	"time"
)

type DBChange struct {
	ID         uint      `gorm:"primaryKey"`
	TableName  string    `gorm:"type:varchar(50);not null;index:idx_table_action"`
	RecordID   int64     `gorm:"not null"`
	ActionType string    `gorm:"type:enum('INSERT','UPDATE','DELETE');not null;index:idx_table_action"`
    ChangedAt  time.Time `gorm:"type:timestamp;default:CURRENT_TIMESTAMP();not null"`
	Processed  bool      `gorm:"default:false;index:idx_processed"`
}
