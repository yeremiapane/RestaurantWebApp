package models

import (
	"time"
)

type Customer struct {
	ID         uint      `gorm:"primaryKey"`
	TableID    *uint     `gorm:"index"`
	Table      Table     `gorm:"foreignKey:TableID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	SessionKey *string   `gorm:"type:varchar(255)"`
	Status     string    `gorm:"type:varchar(20);not null;default:'inactive'"`
	CreatedAt  time.Time `gorm:"not null"`
	UpdatedAt  time.Time `gorm:"not null"`
}
