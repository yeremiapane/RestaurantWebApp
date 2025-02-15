package models

import "time"

type User struct {
	ID        uint   `gorm:"primaryKey"`
	Name      string `gorm:"type:varchar(255); not null"`
	Email     string `gorm:"type:varchar(255); unique;not null"`
	Password  string `gorm:"type:varchar(255); not null"`
	Role      string `gorm:"type:varchar(255); not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
