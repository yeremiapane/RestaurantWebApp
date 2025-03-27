package utils

import (
	"sync"

	"gorm.io/gorm"
)

var (
	db   *gorm.DB
	once sync.Once
	mu   sync.RWMutex
)

// InitDB initializes database connection
func InitDB(database *gorm.DB) {
	once.Do(func() {
		db = database
	})
}

// GetDB returns the database connection
func GetDB() *gorm.DB {
	mu.RLock()
	defer mu.RUnlock()
	return db
}
