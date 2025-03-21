package models

import (
	"encoding/json"

	"gorm.io/gorm"
)

type Menu struct {
	gorm.Model
	CategoryID  uint         `json:"category_id"`
	Category    MenuCategory `json:"category"`
	Name        string       `json:"name"`
	Price       float64      `json:"price"`
	Stock       int          `json:"stock"`
	Description string       `json:"description"`
	ImageUrls   string       `json:"image_urls" gorm:"type:text"`
}

// BeforeCreate - Hook untuk mengatur nilai default sebelum create
func (m *Menu) BeforeCreate(tx *gorm.DB) error {
	if m.ImageUrls == "" {
		m.ImageUrls = "[]"
	}
	return nil
}

// BeforeSave - Hook untuk memastikan ImageUrls tidak pernah NULL
func (m *Menu) BeforeSave(tx *gorm.DB) error {
	if m.ImageUrls == "" {
		m.ImageUrls = "[]"
	}
	return nil
}

// Getter untuk ImageUrls
func (m *Menu) GetImageUrls() []string {
	var urls []string
	if m.ImageUrls != "" {
		json.Unmarshal([]byte(m.ImageUrls), &urls)
	}
	return urls
}

// Setter untuk ImageUrls
func (m *Menu) SetImageUrls(urls []string) error {
	jsonData, err := json.Marshal(urls)
	if err != nil {
		return err
	}
	m.ImageUrls = string(jsonData)
	return nil
}
