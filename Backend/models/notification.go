package models

import (
	"time"
)

type Notification struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    *uint     `json:"user_id,omitempty"`
	User      *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Title     string    `gorm:"type:varchar(255)" json:"title"`
	Message   string    `gorm:"type:text" json:"message"`
	Type      string    `gorm:"type:varchar(50)" json:"type"`
	Status    string    `gorm:"type:varchar(50);default:'unread'" json:"status"`
	CreatedAt time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null" json:"updated_at"`
}
