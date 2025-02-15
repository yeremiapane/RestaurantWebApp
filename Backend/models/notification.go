package models

import (
	"time"
)

type Notification struct {
	ID        uint `gorm:"primaryKey"`
	UserID    *uint
	User      User      `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	Title     *string   `gorm:"type:varchar(100)"`
	Message   string    `gorm:"type:text;not null"`
	CreatedAt time.Time `gorm:"not null"`
}
