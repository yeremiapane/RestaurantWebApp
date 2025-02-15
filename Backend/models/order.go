package models

import (
	"time"
)

type Order struct {
	ID          uint     `gorm:"primaryKey"`
	CustomerID  uint     `gorm:"not null"`
	Customer    Customer `gorm:"foreignKey:CustomerID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	Status      string   `gorm:"type:varchar(20);not null;default:'pending_payment'"`
	TotalAmount float64  `gorm:"type:decimal(10,2);not null;default:0.00"`

	// Tambahan:
	ChefID            *uint      `gorm:"index"` // boleh null jika belum assign ke chef
	Chef              *User      `gorm:"foreignKey:ChefID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	StartCookingTime  *time.Time // waktu mulai masak (status => in_progress)
	FinishCookingTime *time.Time // waktu selesai masak (status => ready)

	CreatedAt time.Time `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`

	// Relasi One To Many Order Item
	OrderItems []OrderItem `gorm:"foreignKey:OrderID"`
}
