package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yeremiapane/restaurant-app/controllers"
	"github.com/yeremiapane/restaurant-app/middlewares"
	"gorm.io/gorm"
)

func SetupRouter(db *gorm.DB) *gin.Engine {
	r := gin.Default()

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
	paymentCtrl := controllers.NewPaymentController(db)
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
	r.POST("/payments", paymentCtrl.CreatePayment)

	// Public routes untuk customer
	r.GET("/tables/:table_id/scan", customerCtrl.ScanTable)           // Scan QR
	r.GET("/tables/:table_id/session", customerCtrl.GetActiveSession) // Cek sesi aktif
	// r.POST("/tables/:table_id/orders", orderCtrl.CreateOrderFromTable) // Buat order dari meja

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
	auth.POST("/menus", menuCtrl.CreateMenu)
	auth.GET("/menus/:menu_id", menuCtrl.GetMenuByID) // detail 1 menu
	auth.PATCH("/menus/:menu_id", menuCtrl.UpdateMenu)
	auth.DELETE("/menus/:menu_id", menuCtrl.DeleteMenu)

	// ORDERS (staff/admin)
	auth.GET("/orders", orderCtrl.GetAllOrders)            // melihat semua orders
	auth.PATCH("/orders/:order_id", orderCtrl.UpdateOrder) // staff menambahkan item, dsb.
	auth.DELETE("/orders/:order_id", orderCtrl.DeleteOrder)

	// PAYMENTS (staff/admin)
	auth.GET("/payments", paymentCtrl.GetAllPayments)
	auth.GET("/payments/:payment_id", paymentCtrl.GetPaymentByID)
	auth.DELETE("/payments/:payment_id", paymentCtrl.DeletePayment)
	auth.POST("/payments/:payment_id/verify", paymentCtrl.VerifyPayment)

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
	auth.GET("/orders/analytics", orderCtrl.GetOrderAnalytics)
	auth.GET("/orders/getflow", adminCtrl.GetOrderFlow)

	// WebSocket endpoint dengan middleware khusus
	wsGroup := r.Group("/ws")
	wsGroup.Use(middlewares.WebSocketAuthMiddleware())
	{
		wsGroup.GET("/:role", controllers.KDSHandler)
	}

	return r
}
