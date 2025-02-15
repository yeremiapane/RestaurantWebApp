package models

import (
	"time"
)

type OrderItem struct {
	ID      uint `gorm:"primaryKey" json:"id"`
	OrderID uint `gorm:"not null" json:"order_id"`
	// Omitting Order field from JSON to avoid recursive nesting
	Order        Order      `gorm:"foreignKey:OrderID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
	MenuID       uint       `gorm:"not null" json:"menu_id"`
	Menu         Menu       `gorm:"foreignKey:MenuID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"menu"`
	Quantity     int        `gorm:"not null" json:"quantity"`
	Price        float64    `gorm:"type:decimal(10,2);not null" json:"price"`
	Notes        string     `gorm:"type:text" json:"notes"`
	ParentItemID *uint      `json:"parent_item_id,omitempty"`
	ParentItem   *OrderItem `gorm:"foreignKey:ParentItemID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"parent_item,omitempty"`
	Status       string     `gorm:"type:varchar(20);not null;default:'pending'" json:"status"`
	CreatedAt    time.Time  `gorm:"not null" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"not null" json:"updated_at"`
}
