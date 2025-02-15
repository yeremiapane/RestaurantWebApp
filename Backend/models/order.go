package models

import (
	"time"
)

type Order struct {
	ID                uint        `gorm:"primaryKey" json:"id"`
	CustomerID        uint        `gorm:"not null" json:"customer_id"`
	Customer          Customer    `gorm:"foreignKey:CustomerID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"customer"`
	Status            string      `gorm:"type:varchar(20);not null;default:'pending_payment'" json:"status"`
	TotalAmount       float64     `gorm:"type:decimal(10,2);not null;default:0.00" json:"total_amount"`
	ChefID            *uint       `gorm:"index" json:"chef_id,omitempty"`
	Chef              *User       `gorm:"foreignKey:ChefID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"chef,omitempty"`
	StartCookingTime  *time.Time  `json:"start_cooking_time,omitempty"`
	FinishCookingTime *time.Time  `json:"finish_cooking_time,omitempty"`
	CreatedAt         time.Time   `gorm:"not null" json:"created_at"`
	UpdatedAt         time.Time   `gorm:"not null" json:"updated_at"`
	OrderItems        []OrderItem `gorm:"foreignKey:OrderID" json:"order_items"`
}
