package models

import (
	"time"
)

// OrderItem menyimpan item dalam suatu Order.
// Kita bisa item-level status: "pending", "in_progress", "ready"
type OrderItem struct {
	ID      uint  `gorm:"primaryKey"`
	OrderID uint  `gorm:"not null"`
	Order   Order `gorm:"foreignKey:OrderID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`

	MenuID   uint    `gorm:"not null"`
	Menu     Menu    `gorm:"foreignKey:MenuID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	Quantity int     `gorm:"not null"`
	Price    float64 `gorm:"type:decimal(10,2);not null"`

	Notes string `gorm:"type:text"` // Catatan per item

	// ParentItemID jika Add-On menempel ke item lain
	ParentItemID *uint
	ParentItem   *OrderItem `gorm:"foreignKey:ParentItemID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`

	// Status item-level
	// (Tambahkan default di field Tag agar "pending" misal)
	Status string `gorm:"type:varchar(20);not null;default:'pending'"`

	CreatedAt time.Time `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`
}
