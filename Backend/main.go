package main

import (
	_ "fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/yeremiapane/restaurant-app/config"
	"github.com/yeremiapane/restaurant-app/models"
	"github.com/yeremiapane/restaurant-app/router"
	"github.com/yeremiapane/restaurant-app/utils"
	"gorm.io/gorm"
)

func main() {
	godotenv.Load() // load file .env jika ada

	// Init logger
	utils.InitLogger()
	utils.InfoLogger.Println("Starting Restaurant App...")

	db, err := config.InitDB()
	if err != nil {
		utils.ErrorLogger.Fatalf("Cannot connect DB: %v", err)
	}

	autoMigrate(db)

	r := router.SetupRouter(db)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	utils.InfoLogger.Printf("Listening on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}

func autoMigrate(db *gorm.DB) {
	err := db.AutoMigrate(
		&models.User{},
		&models.Table{},
		&models.Customer{},
		&models.CleaningLog{},
		&models.MenuCategory{},
		&models.Menu{},
		&models.Order{},
		&models.OrderItem{},
		&models.Payment{},
		&models.Notification{},
	)
	if err != nil {
		utils.ErrorLogger.Fatalf("Failed to AutoMigrate: %v", err)
	}
	utils.InfoLogger.Println("AutoMigrate completed.")
}
