package controllers

import (
	"net/http"
	"time"

	"errors"
	"log"

	"database/sql"

	"github.com/gin-gonic/gin"
	"github.com/yeremiapane/restaurant-app/kds"
	"github.com/yeremiapane/restaurant-app/models"
	"github.com/yeremiapane/restaurant-app/utils"
	"gorm.io/gorm"
)

type AdminController struct {
	DB *gorm.DB
}

func NewAdminController(db *gorm.DB) *AdminController {
	return &AdminController{DB: db}
}

// GetDashboardStats mengambil statistik untuk dashboard
func (ac *AdminController) GetDashboardStats(c *gin.Context) {
	// Debug log
	log.Printf("Role from context: %v", c.GetString("role"))

	roleInterface, exists := c.Get("role")
	if !exists {
		log.Printf("No role found in context")
		utils.RespondError(c, http.StatusUnauthorized, errors.New("no role found"))
		return
	}

	role, ok := roleInterface.(string)
	if !ok {
		log.Printf("Role is not a string: %T", roleInterface)
		utils.RespondError(c, http.StatusUnauthorized, errors.New("invalid role format"))
		return
	}

	if role != "admin" {
		log.Printf("Unauthorized role access attempt: %s", role)
		utils.RespondError(c, http.StatusUnauthorized, errors.New("unauthorized access"))
		return
	}

	today := time.Now().Format("2006-01-02")

	var stats struct {
		TotalOrders    int64   `json:"total_orders"`
		TodayOrders    int64   `json:"today_orders"`
		TotalRevenue   float64 `json:"total_revenue"`
		TodayRevenue   float64 `json:"today_revenue"`
		AvgCookingTime float64 `json:"avg_cooking_time"`
		OrderStats     struct {
			PendingPayment int64 `json:"pending_payment"`
			Paid           int64 `json:"paid"`
			InProgress     int64 `json:"in_progress"`
			Ready          int64 `json:"ready"`
			Completed      int64 `json:"completed"`
		} `json:"order_stats"`
		PaymentStats struct {
			Pending int64   `json:"pending"`
			Success int64   `json:"success"`
			Total   float64 `json:"total"`
			Today   float64 `json:"today"`
		} `json:"payment_stats"`
		TableStats struct {
			Available int64 `json:"available"`
			Occupied  int64 `json:"occupied"`
			Dirty     int64 `json:"dirty"`
		} `json:"table_stats"`
	}

	// Query total dan today orders
	ac.DB.Model(&models.Order{}).Count(&stats.TotalOrders)
	ac.DB.Model(&models.Order{}).Where("DATE(created_at) = ?", today).Count(&stats.TodayOrders)

	// Query order status counts
	ac.DB.Model(&models.Order{}).Where("status = ?", "pending_payment").Count(&stats.OrderStats.PendingPayment)
	ac.DB.Model(&models.Order{}).Where("status = ?", "paid").Count(&stats.OrderStats.Paid)
	ac.DB.Model(&models.Order{}).Where("status = ?", "in_progress").Count(&stats.OrderStats.InProgress)
	ac.DB.Model(&models.Order{}).Where("status = ?", "ready").Count(&stats.OrderStats.Ready)
	ac.DB.Model(&models.Order{}).Where("status = ?", "completed").Count(&stats.OrderStats.Completed)

	// Query payment stats
	ac.DB.Model(&models.Payment{}).Where("status = ?", "pending").Count(&stats.PaymentStats.Pending)
	ac.DB.Model(&models.Payment{}).Where("status = ?", "success").Count(&stats.PaymentStats.Success)

	// Total revenue (all time)
	ac.DB.Model(&models.Payment{}).Where("status = ?", "success").
		Select("COALESCE(SUM(amount), 0)").Row().Scan(&stats.PaymentStats.Total)

	// Today's revenue
	ac.DB.Model(&models.Payment{}).
		Where("status = ? AND DATE(created_at) = ?", "success", today).
		Select("COALESCE(SUM(amount), 0)").Row().Scan(&stats.PaymentStats.Today)

	// Table stats
	ac.DB.Model(&models.Table{}).Where("status = ?", "available").Count(&stats.TableStats.Available)
	ac.DB.Model(&models.Table{}).Where("status = ?", "occupied").Count(&stats.TableStats.Occupied)
	ac.DB.Model(&models.Table{}).Where("status = ?", "dirty").Count(&stats.TableStats.Dirty)

	// Calculate total revenue
	stats.TotalRevenue = stats.PaymentStats.Total
	stats.TodayRevenue = stats.PaymentStats.Today

	// Calculate average cooking time (in minutes)
	var avgCookingTime sql.NullFloat64
	ac.DB.Model(&models.Order{}).
		Where("status = ? AND cooking_start_time IS NOT NULL AND cooking_end_time IS NOT NULL", "completed").
		Select("AVG(TIMESTAMPDIFF(MINUTE, cooking_start_time, cooking_end_time))").
		Row().Scan(&avgCookingTime)

	if avgCookingTime.Valid {
		stats.AvgCookingTime = avgCookingTime.Float64
	}

	// Broadcast stats update
	kds.BroadcastMessage(kds.Message{
		Event: "dashboard_stats",
		Data:  stats,
	})

	utils.RespondJSON(c, http.StatusOK, "Dashboard stats retrieved successfully", gin.H{
		"data": stats,
	})
}

// MonitorOrderFlow memantau alur order secara real-time
func (ac *AdminController) MonitorOrderFlow(c *gin.Context) {
	roleInterface, _ := c.Get("role")
	if roleInterface != "admin" {
		utils.RespondError(c, http.StatusForbidden, ErrNoPermission)
		return
	}

	var orderFlow struct {
		PendingOrders []models.Order   `json:"pending_orders"`
		ActiveOrders  []models.Order   `json:"active_orders"`
		Payments      []models.Payment `json:"pending_payments"`
	}

	// Get pending orders with items
	ac.DB.Preload("OrderItems").Preload("OrderItems.Menu").
		Where("status = ?", "pending_payment").
		Find(&orderFlow.PendingOrders)

	// Get active orders (paid, in_progress, ready)
	ac.DB.Preload("OrderItems").Preload("OrderItems.Menu").
		Where("status IN ?", []string{"paid", "in_progress", "ready"}).
		Find(&orderFlow.ActiveOrders)

	// Get pending payments
	ac.DB.Preload("Order").
		Where("status = ?", "pending").
		Find(&orderFlow.Payments)

	utils.RespondJSON(c, http.StatusOK, "Order flow status", gin.H{
		"data": gin.H{
			"order_flow": orderFlow,
		},
	})
}

// GetSalesReport mengambil laporan penjualan
func (ac *AdminController) GetSalesReport(c *gin.Context) {
	var sales struct {
		TotalSales     float64 `json:"total_sales"`
		TotalOrders    int64   `json:"total_orders"`
		AverageOrder   float64 `json:"average_order"`
		TopSellingMenu []struct {
			MenuID   uint    `json:"menu_id"`
			Name     string  `json:"name"`
			Quantity int     `json:"quantity"`
			Revenue  float64 `json:"revenue"`
		} `json:"top_selling_menu"`
	}

	// Query data penjualan
	ac.DB.Model(&models.Payment{}).Where("status = ?", "success").Select("COALESCE(SUM(amount), 0)").Row().Scan(&sales.TotalSales)
	ac.DB.Model(&models.Order{}).Where("status = ?", "completed").Count(&sales.TotalOrders)

	if sales.TotalOrders > 0 {
		sales.AverageOrder = sales.TotalSales / float64(sales.TotalOrders)
	}

	utils.RespondJSON(c, http.StatusOK, "Sales report", gin.H{
		"data": gin.H{
			"sales": sales,
		},
	})
}

func (ac *AdminController) GetOrderFlow(c *gin.Context) {
	// Ambil orders terlebih dahulu
	var orders []models.Order
	if err := ac.DB.Preload("OrderItems.Menu").
		Order("created_at DESC").
		Limit(10).
		Find(&orders).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	// Buat struktur untuk response
	var recentOrders []struct {
		OrderID     uint      `json:"order_id"`
		TableID     uint      `json:"table_id"`
		TotalAmount float64   `json:"total"`
		Status      string    `json:"status"`
		CreatedAt   time.Time `json:"created_at"`
		Items       []struct {
			Name     string `json:"name"`
			Quantity int    `json:"quantity"`
		} `json:"items"`
	}

	// Transform data orders ke format yang diinginkan
	for _, order := range orders {
		var orderItems []struct {
			Name     string `json:"name"`
			Quantity int    `json:"quantity"`
		}

		for _, item := range order.OrderItems {
			orderItems = append(orderItems, struct {
				Name     string `json:"name"`
				Quantity int    `json:"quantity"`
			}{
				Name:     item.Menu.Name,
				Quantity: item.Quantity,
			})
		}

		recentOrders = append(recentOrders, struct {
			OrderID     uint      `json:"order_id"`
			TableID     uint      `json:"table_id"`
			TotalAmount float64   `json:"total"`
			Status      string    `json:"status"`
			CreatedAt   time.Time `json:"created_at"`
			Items       []struct {
				Name     string `json:"name"`
				Quantity int    `json:"quantity"`
			} `json:"items"`
		}{
			OrderID:     order.ID,
			TableID:     order.TableID,
			TotalAmount: order.TotalAmount,
			Status:      order.Status,
			CreatedAt:   order.CreatedAt,
			Items:       orderItems,
		})
	}

	utils.RespondJSON(c, http.StatusOK, "Recent orders retrieved successfully", gin.H{
		"data": gin.H{
			"recent_orders": recentOrders,
		},
	})
}
