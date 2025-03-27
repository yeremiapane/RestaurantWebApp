package main

import (
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
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
	} else {
		log.Printf("Successfully loaded .env file")
		// Debug: Print current working directory
		if dir, err := os.Getwd(); err == nil {
			log.Printf("Current working directory: %s", dir)
		}
	}

	// Debug: Print all Midtrans environment variables
	log.Printf("MIDTRANS_SERVER_KEY: %s", os.Getenv("MIDTRANS_SERVER_KEY"))
	log.Printf("MIDTRANS_CLIENT_KEY: %s", os.Getenv("MIDTRANS_CLIENT_KEY"))
	log.Printf("MIDTRANS_ENV: %s", os.Getenv("MIDTRANS_ENV"))
	log.Printf("MIDTRANS_MERCHANT_ID: %s", os.Getenv("MIDTRANS_MERCHANT_ID"))
	log.Printf("MIDTRANS_MERCHANT_NAME: %s", os.Getenv("MIDTRANS_MERCHANT_NAME"))
	log.Printf("MIDTRANS_MERCHANT_EMAIL: %s", os.Getenv("MIDTRANS_MERCHANT_EMAIL"))
	log.Printf("MIDTRANS_MERCHANT_PHONE: %s", os.Getenv("MIDTRANS_MERCHANT_PHONE"))

	// Validate required environment variables
	requiredEnvVars := []string{
		"MIDTRANS_SERVER_KEY",
		"MIDTRANS_CLIENT_KEY",
		"MIDTRANS_MERCHANT_ID",
		"MIDTRANS_MERCHANT_NAME",
		"MIDTRANS_MERCHANT_EMAIL",
		"MIDTRANS_MERCHANT_PHONE",
	}

	for _, envVar := range requiredEnvVars {
		if os.Getenv(envVar) == "" {
			log.Printf("Warning: Required environment variable %s is not set", envVar)
		} else {
			log.Printf("Environment variable %s is set", envVar)
		}
	}

	// Initialize logger
	utils.InfoLogger = logrus.New()
	utils.ErrorLogger = logrus.New()

	// Set output to stdout
	utils.InfoLogger.SetOutput(os.Stdout)
	utils.ErrorLogger.SetOutput(os.Stderr)

	// Set formatters
	utils.InfoLogger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
		ForceColors:   true,
	})
	utils.ErrorLogger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
		ForceColors:   true,
	})
}

func main() {
	// Load .env
	if err := godotenv.Load(); err != nil {
		utils.InfoLogger.Println("Warning: .env file not found")
	}

	// Initialize DB
	db, err := config.InitDB()
	if err != nil {
		utils.ErrorLogger.Fatalf("Failed to connect to database: %v", err)
	}

	// Simpan koneksi database ke utils untuk digunakan di controller
	utils.InitDB(db)

	// Set gin mode
	ginMode := os.Getenv("GIN_MODE")
	if ginMode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	autoMigrate(db)

	// Setup rate limiter (10 requests per second per IP)
	rateLimiter := middlewares.NewRateLimiter(50, 1)

	// Inisialisasi change monitor dengan interval yang lebih pendek
	monitor := services.NewChangeMonitor(db)
	monitor.Interval = 500 * time.Millisecond // 500ms interval untuk polling lebih cepat
	monitor.Start()
	defer monitor.Stop()

	// Initialize payment monitor untuk menangani retry dan metrics
	paymentMonitor := services.NewPaymentMonitor(db)
	paymentMonitor.Start()

	// Initialize payment service dan start timeout checker
	paymentService := services.NewPaymentService(db)
	paymentService.StartTimeoutChecker()

	// Setup router
	r := router.SetupRouter(db)
	r.Use(rateLimiter.RateLimit())

	// Add CSP middleware
	r.Use(func(c *gin.Context) {
		// Konfigurasi CSP yang lebih permisif untuk development
		c.Header("Content-Security-Policy", "default-src 'self' 'unsafe-inline' 'unsafe-eval' https://*.ngrok-free.app; img-src 'self' data: https:; connect-src 'self' https://*.ngrok-free.app wss://*.ngrok-free.app; frame-ancestors 'self' https://*.ngrok-free.app; script-src 'self' 'unsafe-inline' 'unsafe-eval';")
		c.Next()
	})

	// Debug middleware
	r.Use(func(c *gin.Context) {
		utils.InfoLogger.Printf("Incoming request: %s %s from %s", c.Request.Method, c.Request.URL.Path, c.ClientIP())
		c.Next()
	})

	// Set trusted proxies
	r.SetTrustedProxies([]string{"127.0.0.1", "localhost", "ngrok-free.app"})

	// Run server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	utils.InfoLogger.Printf("Listening on port %s", port)
	if err := r.Run(":" + port); err != nil {
		utils.ErrorLogger.Fatal(err)
	}
}

func autoMigrate(db *gorm.DB) {
	// Hapus kolom lama terlebih dahulu jika ada
	if db.Migrator().HasColumn(&models.Menu{}, "image_url") {
		if err := db.Migrator().DropColumn(&models.Menu{}, "image_url"); err != nil {
			utils.ErrorLogger.Printf("Error dropping image_url column: %v", err)
		}
	}

	// Kemudian lakukan AutoMigrate
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

	// Update existing records yang memiliki image_urls NULL
	if err := db.Exec("UPDATE menus SET image_urls = '[]' WHERE image_urls IS NULL OR image_urls = ''").Error; err != nil {
		utils.ErrorLogger.Printf("Error updating null image_urls: %v", err)
	}
}
