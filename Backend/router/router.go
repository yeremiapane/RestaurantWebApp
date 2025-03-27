package router

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yeremiapane/restaurant-app/controllers"
	"github.com/yeremiapane/restaurant-app/middlewares"
	"gorm.io/gorm"
)

func SetupRouter(db *gorm.DB) *gin.Engine {
	r := gin.Default()

	// Get current working directory
	workDir, _ := os.Getwd()

	// Debug the working directory
	fmt.Println("Current working directory:", workDir)

	// Handle static files with absolute paths
	frontendPath := filepath.Join(workDir, "..", "Frontend")

	// Check if Frontend directory exists
	if _, err := os.Stat(frontendPath); os.IsNotExist(err) {
		// Try with current directory if parent directory doesn't have Frontend
		frontendPath = filepath.Join(workDir, "Frontend")
		fmt.Println("Using local Frontend path:", frontendPath)
	} else {
		fmt.Println("Using parent Frontend path:", frontendPath)
	}

	// Check if path exists
	if _, err := os.Stat(frontendPath); os.IsNotExist(err) {
		fmt.Println("WARNING: Frontend path not found:", frontendPath)
	} else {
		fmt.Println("Frontend path exists:", frontendPath)
		// List files in the frontend directory to debug
		if files, err := os.ReadDir(frontendPath); err == nil {
			fmt.Println("Files in Frontend directory:")
			for _, f := range files {
				fmt.Println(" -", f.Name())
			}
		}
	}

	// Serve static files
	r.Static("/Frontend", frontendPath)

	// Middleware untuk membatasi akses ke direktori uploads
	uploadsPath := filepath.Join(workDir, "public", "uploads")
	r.Static("/uploads", uploadsPath)

	// Root path handler - redirect to login page
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/Frontend/auth/login/index.html")
	})

	r.Use(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/uploads/") {
			// Hanya izinkan akses ke file gambar
			if !strings.HasSuffix(strings.ToLower(c.Request.URL.Path), ".jpg") &&
				!strings.HasSuffix(strings.ToLower(c.Request.URL.Path), ".jpeg") &&
				!strings.HasSuffix(strings.ToLower(c.Request.URL.Path), ".png") &&
				!strings.HasSuffix(strings.ToLower(c.Request.URL.Path), ".gif") &&
				!strings.HasSuffix(strings.ToLower(c.Request.URL.Path), ".webp") {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
		}
		c.Next()
	})

	// Apply security middlewares
	r.Use(middlewares.SecurityHeaders())
	r.Use(middlewares.CORSMiddlewares())
	r.Use(middlewares.LoggerMiddleware())

	// Inisialisasi controller
	userCtrl := controllers.NewUserController(db)
	tableCtrl := controllers.NewTableController(db)
	customerCtrl := controllers.NewCustomerController(db)
	categoryCtrl := controllers.NewMenuCategoryController(db)
	menuCtrl := controllers.NewMenuController(db)
	orderCtrl := controllers.NewOrderController(db)
	cleanLogCtrl := controllers.NewCleaningLogController(db)
	notificationCtrl := controllers.NewNotificationController(db)
	adminCtrl := controllers.NewAdminController(db)
	receiptCtrl := controllers.NewReceiptController(db)

	// Melayani File Statis

	// ----------------------------------------------------------------
	//                      PUBLIC ROUTES
	// ----------------------------------------------------------------
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	// Rate limiter untuk login/register
	public := r.Group("/")
	public.Use(middlewares.NewStrictRateLimiter())
	{
		public.POST("/register", userCtrl.Register)
		public.POST("/login", userCtrl.Login)
	}

	// Endpoint KDS WebSocket (opsional, jika Chef perlu real-time)
	r.GET("/kds/ws", controllers.KDSHandler)

	// -- CUSTOMER (Tanpa Auth) --
	// Lihat kategori
	r.GET("/categories", categoryCtrl.GetAllCategories)

	// Lihat menu
	r.GET("/menus", menuCtrl.GetAllMenus)
	r.GET("/menus/by-category", menuCtrl.GetMenuByCategory)

	// Membuat order (Customer tidak perlu login)
	r.POST("/orders", orderCtrl.CreateOrder)
	// Opsional: Melihat detail order
	r.GET("/orders/:order_id", orderCtrl.GetOrderByID)

	// Membayar (mis. cash/QRIS) tanpa login (sesuai kebutuhan)
	r.POST("/payments", controllers.CreatePayment)
	r.POST("/payments/callback", controllers.HandlePaymentCallback)

	// Public routes untuk customer
	r.GET("/tables/:table_id/scan", customerCtrl.ScanTable)           // Scan QR
	r.GET("/tables/:table_id/session", customerCtrl.GetActiveSession) // Cek sesi aktif
	r.GET("/tables", tableCtrl.GetAllTables)                          // Get all tables
	r.GET("/customers", customerCtrl.GetAllCustomers)                 // Get all customers

	// ----------------------------------------------------------------
	//                      AUTHENTICATED ROUTES
	// ----------------------------------------------------------------
	auth := r.Group("/admin")
	auth.Use(middlewares.EnhancedAuthMiddleware())

	// Contoh: Profil user (Admin/Staff/Chef)
	auth.GET("/profile", userCtrl.GetProfile)
	auth.GET("/users", userCtrl.GetAllUsers)

	// TABLE
	auth.GET("/tables", tableCtrl.GetAllTables)
	auth.PATCH("/tables/:table_id", tableCtrl.UpdateTableStatus)

	// CUSTOMERS (staff/admin)
	auth.GET("/customers", customerCtrl.GetAllCustomers)
	auth.POST("/customers", customerCtrl.CreateCustomer) // mgkn staff menambahkan customer manual
	auth.GET("/customers/:customer_id", customerCtrl.GetCustomerByID)
	auth.PATCH("/customers/:customer_id", customerCtrl.UpdateCustomer)
	auth.DELETE("/customers/:customer_id", customerCtrl.DeleteCustomer)

	// MENU CATEGORIES (staff/admin only)
	auth.POST("/categories", categoryCtrl.CreateCategory)
	auth.PATCH("/categories/:cat_id", categoryCtrl.UpdateCategory)
	auth.DELETE("/categories/:cat_id", categoryCtrl.DeleteCategory)

	// MENUS (staff/admin)
	auth.GET("/menus", menuCtrl.GetAllMenus) // Get all menus
	auth.POST("/menus", menuCtrl.CreateMenu)
	auth.GET("/menus/:menu_id", menuCtrl.GetMenuByID) // detail 1 menu
	auth.PATCH("/menus/:menu_id", menuCtrl.UpdateMenu)
	auth.DELETE("/menus/:menu_id", menuCtrl.DeleteMenu)

	// ORDERS (staff/admin)
	auth.GET("/orders", orderCtrl.GetAllOrders)            // melihat semua orders
	auth.GET("/orders/:order_id", orderCtrl.GetOrderByID)  // melihat detail order
	auth.PATCH("/orders/:order_id", orderCtrl.UpdateOrder) // staff menambahkan item, dsb.
	auth.PUT("/orders/:order_id", orderCtrl.UpdateOrder)   // untuk update status
	auth.DELETE("/orders/:order_id", orderCtrl.DeleteOrder)

	// PAYMENTS (staff/admin)
	auth.GET("/payments", controllers.GetPayments)
	auth.POST("/payments", controllers.CreatePayment)
	auth.GET("/payments/:payment_id", controllers.GetPayment)
	auth.DELETE("/payments/:payment_id", controllers.DeletePayment)
	auth.POST("/payments/:payment_id/verify", controllers.VerifyPayment)
	auth.GET("/payments/:payment_id/check", controllers.CheckPaymentStatus)
	auth.GET("/orders/:order_id/check-payment", controllers.CheckOrderPaymentStatus)
	auth.GET("/payments/config", controllers.GetMidtransConfig)

	// Routes untuk receipt dengan middleware logger
	receiptGroup := auth.Group("/payments")
	receiptGroup.Use(middlewares.ReceiptLoggerMiddleware())
	{
		receiptGroup.POST("/:payment_id/receipt", receiptCtrl.GenerateReceipt)
	}
	auth.GET("/receipts/:receipt_id", receiptCtrl.GetReceiptByID)

	// CLEANING LOGS (Cleaner, staff, admin)
	auth.GET("/cleaning-logs", cleanLogCtrl.GetAllCleaningLogs)
	auth.POST("/cleaning-logs", cleanLogCtrl.CreateCleaningLog)
	auth.GET("/cleaning-logs/:clean_id", cleanLogCtrl.GetCleaningLogByID)
	auth.PATCH("/cleaning-logs/:clean_id", cleanLogCtrl.UpdateCleaningLog)
	auth.DELETE("/cleaning-logs/:clean_id", cleanLogCtrl.DeleteCleaningLog)

	// NOTIFICATIONS (staff/admin)
	auth.GET("/notifications", notificationCtrl.GetAllNotifications)
	auth.POST("/notifications", notificationCtrl.CreateNotification)
	auth.GET("/notifications/:notif_id", notificationCtrl.GetNotificationByID)
	auth.DELETE("/notifications/:notif_id", notificationCtrl.DeleteNotification)

	// KDS item-level (Chef)
	auth.POST("/order-items/:item_id/start", orderCtrl.StartCookingItem)
	auth.POST("/order-items/:item_id/finish", orderCtrl.FinishCookingItem)

	// KDS order-level (opsional)
	auth.POST("/orders/:id/start-cooking", orderCtrl.StartCooking)
	auth.POST("/orders/:id/finish-cooking", orderCtrl.FinishCooking)
	auth.POST("/orders/:id/complete", orderCtrl.CompleteOrder) // staff mark completed

	// Routes untuk Chef
	auth.GET("/kitchen/pending-items", orderCtrl.GetPendingItems)
	auth.GET("/kitchen/display", orderCtrl.GetKitchenDisplay)

	// Routes untuk Staff/Cleaner
	auth.PATCH("/tables/:table_id/clean", tableCtrl.MarkTableClean)

	// Routes untuk Admin
	auth.GET("/dashboard/stats", adminCtrl.GetDashboardStats)
	auth.GET("/orders/flow", adminCtrl.MonitorOrderFlow)
	auth.GET("/orders/analytics", adminCtrl.GetAnalytics)
	auth.GET("/orders/getflow", adminCtrl.GetOrderFlow)
	auth.GET("/orders/stats", adminCtrl.GetOrderStats)
	auth.GET("/reports/export", adminCtrl.ExportData)
	auth.GET("/reports/export-pdf", adminCtrl.ExportPDF)

	// WebSocket endpoint dengan middleware khusus
	wsGroup := r.Group("/ws")
	wsGroup.Use(middlewares.WebSocketAuthMiddleware())
	{
		wsGroup.GET("/:role", controllers.KDSHandler)
	}

	return r
}
