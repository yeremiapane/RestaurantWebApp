package models

import "time"

type Table struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	TableNumber string    `gorm:"type:varchar(50);not null" json:"number"`
	Status      string    `gorm:"type:varchar(50);not null;default:'available'" json:"status"`
	CreatedAt   time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt   time.Time `gorm:"not null" json:"updated_at"`
}
