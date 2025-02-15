package models

import (
	"time"
)

type CleaningLog struct {
	ID        uint      `gorm:"primaryKey"`
	CleanerID uint      `gorm:"not null"`
	Cleaner   User      `gorm:"foreignKey:CleanerID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	TableID   uint      `gorm:"not null"`
	Table     Table     `gorm:"foreignKey:TableID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	Status    string    `gorm:"type:varchar(15);not null;default:'pending'"`
	CreatedAt time.Time `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`
}
