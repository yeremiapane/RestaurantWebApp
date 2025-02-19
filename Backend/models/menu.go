package models

import "time"

type Menu struct {
	ID          uint         `gorm:"primaryKey"`
	CategoryID  uint         `gorm:"not null"`
	Category    MenuCategory `gorm:"foreignKey:CategoryID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	Name        string       `gorm:"type:varchar(255); not null"`
	Price       float64      `gorm:"type:decimal(10,2); not null"`
	Stock       int
	Description string    `gorm:"type:text"`
	ImageUrl    *string   `gorm:"type:varchar(255); not null"`
	CreatedAt   time.Time `gorm:"not null"`
	UpdatedAt   time.Time `gorm:"not null"`
}
