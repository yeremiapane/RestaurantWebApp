package main

import (
	_ "fmt"
	"log"
	"os"
	_ "strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/yeremiapane/restaurant-app/config"
	"github.com/yeremiapane/restaurant-app/database"
	"github.com/yeremiapane/restaurant-app/middlewares"
	"github.com/yeremiapane/restaurant-app/models"
	"github.com/yeremiapane/restaurant-app/router"
	"github.com/yeremiapane/restaurant-app/services"
	"github.com/yeremiapane/restaurant-app/utils"
	"gorm.io/gorm"
)

func init() {
	// Load .env file di awal sebelum apapun
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found or error loading: %v", err)
	}
}

func main() {
	// Init logger
	utils.InitLogger()
	utils.InfoLogger.Println("Starting Restaurant App...")

	db, err := config.InitDB()
	if err != nil {
		utils.ErrorLogger.Fatalf("Cannot connect DB: %v", err)
	}

	autoMigrate(db)

	// Setup rate limiter (10 requests per second per IP)
	rateLimiter := middlewares.NewRateLimiter(50, 1)

	// Inisialisasi change monitor dengan interval yang lebih pendek
	monitor := services.NewChangeMonitor(db)
	monitor.Interval = 500 * time.Millisecond // 500ms interval untuk polling lebih cepat
	monitor.Start()
	defer monitor.Stop()

	r := router.SetupRouter(db)
	r.Use(rateLimiter.RateLimit())

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
		&models.Receipt{},
		&models.ReceiptItem{},
		&models.ReceiptAddOn{},
		&models.DBChange{},
	)
	if err != nil {
		utils.ErrorLogger.Fatalf("Failed to AutoMigrate: %v", err)
	}
	utils.InfoLogger.Println("AutoMigrate completed.")

	// Execute triggers
	if err := database.ExecuteTriggers(db); err != nil {
		utils.ErrorLogger.Printf("Error setting up triggers: %v", err)
	}
}
