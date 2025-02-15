package models

import (
	"time"
)

type Order struct {
	ID          uint      `gorm:"primaryKey"`
	CustomerID  uint      `gorm:"not null"`
	Customer    Customer  `gorm:"foreignKey:CustomerID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	Status      string    `gorm:"type:varchar(20);not null;default:'draft'"`
	TotalAmount float64   `gorm:"type:decimal(10,2);not null;default:0.00"`
	CreatedAt   time.Time `gorm:"not null"`
	UpdatedAt   time.Time `gorm:"not null"`

	// Relasi One-to-Many ke OrderItem
	OrderItems []OrderItem `gorm:"foreignKey:OrderID"`
}
