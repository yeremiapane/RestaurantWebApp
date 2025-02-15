package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yeremiapane/restaurant-app/controllers"
	"github.com/yeremiapane/restaurant-app/middlewares"
	"gorm.io/gorm"
	"os"
)

func SetupRouter(db *gorm.DB) *gin.Engine {
	r := gin.Default()
	r.Use(middlewares.LoggerMiddleware())

	// Controller
	userCtrl := controllers.NewUserController(db)
	tableCtrl := controllers.NewTableController(db)
	customerCtrl := controllers.NewCustomerController(db)
	categoryCtrl := controllers.NewMenuCategoryController(db)
	menuCtrl := controllers.NewMenuController(db)
	orderCtrl := controllers.NewOrderController(db)
	paymentCtrl := controllers.NewPaymentController(db)
	cleanLogCtrl := controllers.NewCleaningLogController(db)
	notificationCtrl := controllers.NewNotificationController(db)

	// Public routes
	r.GET("/ping", func(c *gin.Context) { c.JSON(200, gin.H{"message": "pong"}) })
	r.POST("/register", userCtrl.Register)
	r.POST("/login", userCtrl.Login)
	// Route WebSocket KDS
	r.GET("/kds/ws", controllers.KDSHandler)

	// Auth group
	auth := r.Group("/")
	if os.Getenv("ENV") != "TEST" {
		auth.Use(middlewares.AuthMiddleware())
	}

	// USER
	auth.GET("/profile", userCtrl.GetProfile)
	auth.GET("/users", userCtrl.GetAllUsers)

	// TABLE
	auth.GET("/tables", tableCtrl.GetAllTables)
	auth.PATCH("/tables/:table_id", tableCtrl.UpdateTableStatus)

	// CUSTOMERS
	auth.GET("/customers", customerCtrl.GetAllCustomers)
	auth.POST("/customers", customerCtrl.CreateCustomer)
	auth.GET("/customers/:customer_id", customerCtrl.GetCustomerByID)
	auth.PATCH("/customers/:customer_id", customerCtrl.UpdateCustomer)
	auth.DELETE("/customers/:customer_id", customerCtrl.DeleteCustomer)

	// MENU CATEGORIES
	auth.GET("/categories", categoryCtrl.GetAllCategories)
	auth.POST("/categories", categoryCtrl.CreateCategory)
	auth.GET("/categories/:cat_id", categoryCtrl.GetCategoryByID)
	auth.PATCH("/categories/:cat_id", categoryCtrl.UpdateCategory)
	auth.DELETE("/categories/:cat_id", categoryCtrl.DeleteCategory)

	// MENUS
	auth.GET("/menus", menuCtrl.GetAllMenus)
	auth.POST("/menus", menuCtrl.CreateMenu)
	auth.GET("/menus/:menu_id", menuCtrl.GetMenuByID)
	auth.PATCH("/menus/:menu_id", menuCtrl.UpdateMenu)
	auth.DELETE("/menus/:menu_id", menuCtrl.DeleteMenu)

	// ORDERS
	auth.GET("/orders", orderCtrl.GetAllOrders)
	auth.POST("/orders", orderCtrl.CreateOrder)
	auth.GET("/orders/:order_id", orderCtrl.GetOrderByID)
	auth.PATCH("/orders/:order_id", orderCtrl.UpdateOrder)
	auth.DELETE("/orders/:order_id", orderCtrl.DeleteOrder)

	// PAYMENTS
	auth.GET("/payments", paymentCtrl.GetAllPayments)
	auth.POST("/payments", paymentCtrl.CreatePayment)
	auth.GET("/payments/:payment_id", paymentCtrl.GetPaymentByID)
	auth.DELETE("/payments/:payment_id", paymentCtrl.DeletePayment)

	// CLEANING LOGS
	auth.GET("/cleaning-logs", cleanLogCtrl.GetAllCleaningLogs)
	auth.POST("/cleaning-logs", cleanLogCtrl.CreateCleaningLog)
	auth.GET("/cleaning-logs/:clean_id", cleanLogCtrl.GetCleaningLogByID)
	auth.PATCH("/cleaning-logs/:clean_id", cleanLogCtrl.UpdateCleaningLog)
	auth.DELETE("/cleaning-logs/:clean_id", cleanLogCtrl.DeleteCleaningLog)

	// NOTIFICATIONS
	auth.GET("/notifications", notificationCtrl.GetAllNotifications)
	auth.POST("/notifications", notificationCtrl.CreateNotification)
	auth.GET("/notifications/:notif_id", notificationCtrl.GetNotificationByID)
	auth.DELETE("/notifications/:notif_id", notificationCtrl.DeleteNotification)

	// KDS item-level
	auth.POST("/order-items/:item_id/start", orderCtrl.StartCookingItem)
	auth.POST("/order-items/:item_id/finish", orderCtrl.FinishCookingItem)

	// KDS order-level (opsional)
	auth.POST("/orders/:id/start-cooking", orderCtrl.StartCooking)
	auth.POST("/orders/:id/finish-cooking", orderCtrl.FinishCooking)
	auth.POST("/orders/:id/complete", orderCtrl.CompleteOrder) // staff mark completed

	return r
}
