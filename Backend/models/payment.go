package models

import (
	"time"
)

type Payment struct {
	ID            uint   `gorm:"primaryKey"`
	OrderID       uint   `gorm:"not null"`
	Order         Order  `gorm:"foreignKey:OrderID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	PaymentMethod string `gorm:"type:varchar(10);not null;default:'qris'"`
	Status        string `gorm:"type:varchar(10);not null;default:'pending'"`
	PaymentTime   *time.Time
	Amount        float64   `gorm:"type:decimal(10,2);not null"`
	ReferenceID   string    `gorm:"type:varchar(255);not null"`
	CreatedAt     time.Time `gorm:"not null"`
	UpdatedAt     time.Time `gorm:"not null"`
}
