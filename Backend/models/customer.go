package models

import (
	"time"
)

type Customer struct {
	ID         uint      `gorm:"primaryKey"`
	TableID    uint      `gorm:"not null"`
	Table      Table     `gorm:"foreignKey:TableID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	SessionKey *string   `gorm:"type:varchar(255)"`
	Status     string    `gorm:"type:varchar(10);not null;default:'active'"`
	CreatedAt  time.Time `gorm:"not null"`
	UpdatedAt  time.Time `gorm:"not null"`
}
