package models

import "time"

type Table struct {
	ID          uint      `gorm:"primaryKey"`
	TableNumber string    `gorm:"type:varchar(50);not null"`
	Status      string    `gorm:"type:varchar(50);not null;default:'available'"`
	CreatedAt   time.Time `gorm:"not null"`
	UpdatedAt   time.Time `gorm:"not null"`
}
