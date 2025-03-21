package controllers

import (
	"net/http"
	"time"

	"errors"
	"fmt"
	"log"

	"database/sql"

	"encoding/csv"

	"bytes"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/go-pdf/fpdf"
	"github.com/wcharczuk/go-chart/v2"
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
		RevenueTrend []struct {
			Date   string  `json:"date"`
			Amount float64 `json:"amount"`
		} `json:"revenue_trend"`
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

	// Calculate total revenue from completed orders
	ac.DB.Model(&models.Order{}).
		Where("status = ?", "completed").
		Select("COALESCE(SUM(total_amount), 0)").
		Row().Scan(&stats.TotalRevenue)

	// Calculate today's revenue from completed orders
	ac.DB.Model(&models.Order{}).
		Where("status = ? AND DATE(created_at) = ?", "completed", today).
		Select("COALESCE(SUM(total_amount), 0)").
		Row().Scan(&stats.TodayRevenue)

	// Calculate average cooking time for completed orders
	var avgCookingTime sql.NullFloat64
	ac.DB.Model(&models.Order{}).
		Where("status = ? AND cooking_start_time IS NOT NULL AND cooking_end_time IS NOT NULL", "completed").
		Select("COALESCE(AVG(TIMESTAMPDIFF(MINUTE, cooking_start_time, cooking_end_time)), 0)").
		Row().Scan(&avgCookingTime)

	if avgCookingTime.Valid {
		stats.AvgCookingTime = avgCookingTime.Float64
	}

	// Calculate revenue trend for the last 7 days
	var revenueTrend []struct {
		Date   string  `json:"date"`
		Amount float64 `json:"amount"`
	}

	// Get last 7 days including today
	for i := 6; i >= 0; i-- {
		date := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		var amount float64

		ac.DB.Model(&models.Order{}).
			Where("status = ? AND DATE(created_at) = ?", "completed", date).
			Select("COALESCE(SUM(total_amount), 0)").
			Row().Scan(&amount)

		revenueTrend = append(revenueTrend, struct {
			Date   string  `json:"date"`
			Amount float64 `json:"amount"`
		}{
			Date:   date,
			Amount: amount,
		})
	}

	stats.RevenueTrend = revenueTrend

	// Log stats untuk debugging
	log.Printf("Sending dashboard stats: %+v", stats)

	utils.RespondJSON(c, http.StatusOK, "Dashboard stats retrieved successfully", gin.H{
		"data": gin.H{
			"table_stats": gin.H{
				"available": stats.TableStats.Available,
				"occupied":  stats.TableStats.Occupied,
				"dirty":     stats.TableStats.Dirty,
				"total":     stats.TableStats.Available + stats.TableStats.Occupied + stats.TableStats.Dirty,
			},
			"order_stats": gin.H{
				"pending_payment": stats.OrderStats.PendingPayment,
				"paid":            stats.OrderStats.Paid,
				"in_progress":     stats.OrderStats.InProgress,
				"ready":           stats.OrderStats.Ready,
				"completed":       stats.OrderStats.Completed,
			},
			"payment_stats": gin.H{
				"pending": stats.PaymentStats.Pending,
				"success": stats.PaymentStats.Success,
				"total":   stats.PaymentStats.Total,
				"today":   stats.PaymentStats.Today,
			},
			"total_orders":     stats.TotalOrders,
			"today_orders":     stats.TodayOrders,
			"total_revenue":    stats.TotalRevenue,
			"today_revenue":    stats.TodayRevenue,
			"avg_cooking_time": stats.AvgCookingTime,
			"revenue_trend":    stats.RevenueTrend,
		},
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
	// Ambil orders dengan relasi yang diperlukan
	var orders []models.Order
	if err := ac.DB.Preload("OrderItems").
		Preload("OrderItems.Menu").
		Preload("Table").
		Order("created_at DESC").
		Limit(10).
		Find(&orders).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	// Buat struktur untuk response
	type OrderItem struct {
		Name     string `json:"name"`
		Quantity int    `json:"quantity"`
	}

	type RecentOrder struct {
		OrderID     uint        `json:"order_id"`
		TableID     uint        `json:"table_id"`
		TableNumber string      `json:"table_number"`
		TotalAmount float64     `json:"total"`
		Status      string      `json:"status"`
		CreatedAt   time.Time   `json:"created_at"`
		Items       []OrderItem `json:"items"`
	}

	var recentOrders []RecentOrder

	// Transform data orders ke format yang diinginkan
	for _, order := range orders {
		var items []OrderItem
		for _, item := range order.OrderItems {
			items = append(items, OrderItem{
				Name:     item.Menu.Name,
				Quantity: item.Quantity,
			})
		}

		tableNumber := "0"
		if order.Table.TableNumber != "" {
			tableNumber = order.Table.TableNumber
		}

		recentOrders = append(recentOrders, RecentOrder{
			OrderID:     order.ID,
			TableID:     order.TableID,
			TableNumber: tableNumber,
			TotalAmount: order.TotalAmount,
			Status:      order.Status,
			CreatedAt:   order.CreatedAt,
			Items:       items,
		})
	}

	utils.RespondJSON(c, http.StatusOK, "Recent orders retrieved successfully", gin.H{
		"data": gin.H{
			"recent_orders": recentOrders,
		},
	})
}

// GetOrderStats mengambil statistik order untuk dashboard
func (ac *AdminController) GetOrderStats(c *gin.Context) {
	var stats struct {
		PendingPayment int64 `json:"pending_payment"`
		Paid           int64 `json:"paid"`
		InProgress     int64 `json:"in_progress"`
		Ready          int64 `json:"ready"`
		Completed      int64 `json:"completed"`
	}

	// Query order status counts
	ac.DB.Model(&models.Order{}).Where("status = ?", "pending_payment").Count(&stats.PendingPayment)
	ac.DB.Model(&models.Order{}).Where("status = ?", "paid").Count(&stats.Paid)
	ac.DB.Model(&models.Order{}).Where("status = ?", "in_progress").Count(&stats.InProgress)
	ac.DB.Model(&models.Order{}).Where("status = ?", "ready").Count(&stats.Ready)
	ac.DB.Model(&models.Order{}).Where("status = ?", "completed").Count(&stats.Completed)

	utils.RespondJSON(c, http.StatusOK, "Order stats retrieved successfully", gin.H{
		"data": stats,
	})
}

// GetAnalytics mengambil data analitik untuk laporan
func (ac *AdminController) GetAnalytics(c *gin.Context) {
	roleInterface, _ := c.Get("role")
	if roleInterface != "admin" {
		utils.RespondError(c, http.StatusForbidden, ErrNoPermission)
		return
	}

	// Get period parameter
	period := c.DefaultQuery("period", "week")
	var startDate time.Time
	var endDate time.Time

	now := time.Now()
	switch period {
	case "today":
		startDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		endDate = now
	case "week":
		startDate = now.AddDate(0, 0, -6)
		endDate = now
	case "month":
		startDate = now.AddDate(0, -1, 0)
		endDate = now
	case "year":
		startDate = now.AddDate(-1, 0, 0)
		endDate = now
	default:
		startDate = now.AddDate(0, 0, -6) // Default to last 7 days
		endDate = now
	}

	var analytics struct {
		TotalSales      float64 `json:"total_sales"`
		TotalOrders     int64   `json:"total_orders"`
		AverageOrder    float64 `json:"average_order"`
		PopularCategory struct {
			Name  string `json:"name"`
			Count int64  `json:"count"`
		} `json:"popular_category"`
		SalesTrend []struct {
			Date   string  `json:"date"`
			Amount float64 `json:"amount"`
		} `json:"sales_trend"`
		CategoryPerformance []struct {
			Name  string  `json:"name"`
			Total float64 `json:"total"`
		} `json:"category_performance"`
		PeakHours []struct {
			Hour  int   `json:"hour"`
			Count int64 `json:"count"`
		} `json:"peak_hours"`
		PopularItems []struct {
			MenuName string  `json:"menu_name"`
			Count    int     `json:"count"`
			Revenue  float64 `json:"revenue"`
			Trend    float64 `json:"trend"`
		} `json:"popular_items"`
		MenuPerformance []struct {
			Name    string  `json:"name"`
			Sold    int     `json:"sold"`
			Revenue float64 `json:"revenue"`
			Trend   float64 `json:"trend"`
		} `json:"menu_performance"`
	}

	// Query total sales dan orders with date range
	ac.DB.Model(&models.Order{}).
		Where("status = ? AND created_at BETWEEN ? AND ?", "completed", startDate, endDate).
		Count(&analytics.TotalOrders)

	ac.DB.Model(&models.Order{}).
		Where("status = ? AND created_at BETWEEN ? AND ?", "completed", startDate, endDate).
		Select("COALESCE(SUM(total_amount), 0)").
		Row().Scan(&analytics.TotalSales)

	if analytics.TotalOrders > 0 {
		analytics.AverageOrder = analytics.TotalSales / float64(analytics.TotalOrders)
	}

	// Query popular category with date range
	ac.DB.Raw(`
		SELECT c.name, COUNT(oi.id) as count
		FROM order_items oi
		JOIN menus m ON oi.menu_id = m.id
		JOIN menu_categories c ON m.category_id = c.id
		JOIN orders o ON oi.order_id = o.id
		WHERE o.status = 'completed'
		AND o.created_at BETWEEN ? AND ?
		GROUP BY c.id, c.name
		ORDER BY count DESC
		LIMIT 1
	`, startDate, endDate).Scan(&analytics.PopularCategory)

	// Query sales trend based on period
	var dateFormat string
	switch period {
	case "today":
		dateFormat = "%H:00"
		// Generate all hours for today
		for i := 0; i < 24; i++ {
			analytics.SalesTrend = append(analytics.SalesTrend, struct {
				Date   string  `json:"date"`
				Amount float64 `json:"amount"`
			}{
				Date:   fmt.Sprintf("%02d:00", i),
				Amount: 0,
			})
		}
	case "week":
		dateFormat = "%Y-%m-%d"
		// Generate last 7 days
		for i := 6; i >= 0; i-- {
			date := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
			analytics.SalesTrend = append(analytics.SalesTrend, struct {
				Date   string  `json:"date"`
				Amount float64 `json:"amount"`
			}{
				Date:   date,
				Amount: 0,
			})
		}
	case "month":
		dateFormat = "%Y-%m-%d"
		// Generate last 30 days
		for i := 29; i >= 0; i-- {
			date := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
			analytics.SalesTrend = append(analytics.SalesTrend, struct {
				Date   string  `json:"date"`
				Amount float64 `json:"amount"`
			}{
				Date:   date,
				Amount: 0,
			})
		}
	case "year":
		dateFormat = "%Y-%m"
		// Generate last 12 months
		for i := 11; i >= 0; i-- {
			date := time.Now().AddDate(0, -i, 0).Format("2006-01")
			analytics.SalesTrend = append(analytics.SalesTrend, struct {
				Date   string  `json:"date"`
				Amount float64 `json:"amount"`
			}{
				Date:   date,
				Amount: 0,
			})
		}
	default:
		dateFormat = "%Y-%m-%d"
		// Default to last 7 days
		for i := 6; i >= 0; i-- {
			date := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
			analytics.SalesTrend = append(analytics.SalesTrend, struct {
				Date   string  `json:"date"`
				Amount float64 `json:"amount"`
			}{
				Date:   date,
				Amount: 0,
			})
		}
	}

	// Query actual sales data
	var salesData []struct {
		Date   string  `json:"date"`
		Amount float64 `json:"amount"`
	}

	ac.DB.Raw(`
		WITH daily_sales AS (
			SELECT 
				DATE(created_at) as sale_date,
				COALESCE(SUM(total_amount), 0) as daily_amount
			FROM orders
			WHERE status = 'completed'
			AND created_at BETWEEN ? AND ?
			GROUP BY DATE(created_at)
		)
		SELECT 
			DATE_FORMAT(sale_date, ?) as date,
			daily_amount as amount
		FROM daily_sales
		ORDER BY sale_date ASC
	`, startDate, endDate, dateFormat).Scan(&salesData)

	// Log raw sales data
	log.Printf("Raw sales data for period %s: %+v", period, salesData)

	// Update sales trend with actual data
	for _, sale := range salesData {
		for i := range analytics.SalesTrend {
			if analytics.SalesTrend[i].Date == sale.Date {
				analytics.SalesTrend[i].Amount = sale.Amount
				log.Printf("Updated sales trend for date %s with amount %f", sale.Date, sale.Amount)
				break
			}
		}
	}

	// Log final sales trend data
	log.Printf("Final sales trend data for period %s: %+v", period, analytics.SalesTrend)

	// Query category performance with date range
	ac.DB.Raw(`
		SELECT c.name, COALESCE(SUM(oi.price * oi.quantity), 0) as total
		FROM order_items oi
		JOIN menus m ON oi.menu_id = m.id
		JOIN menu_categories c ON m.category_id = c.id
		JOIN orders o ON oi.order_id = o.id
		WHERE o.status = 'completed'
		AND o.created_at BETWEEN ? AND ?
		GROUP BY c.id, c.name
		ORDER BY total DESC
	`, startDate, endDate).Scan(&analytics.CategoryPerformance)

	// Query peak hours with date range
	ac.DB.Raw(`
		SELECT EXTRACT(HOUR FROM created_at) as hour, COUNT(*) as count
		FROM orders
		WHERE status = 'completed'
		AND created_at BETWEEN ? AND ?
		GROUP BY EXTRACT(HOUR FROM created_at)
		ORDER BY hour ASC
	`, startDate, endDate).Scan(&analytics.PeakHours)

	// Query popular items with date range
	ac.DB.Raw(`
		WITH recent_orders AS (
			SELECT 
				m.id as menu_id,
				m.name as menu_name,
				COALESCE(COUNT(oi.id), 0) as recent_count,
				COALESCE(SUM(oi.price * oi.quantity), 0) as recent_revenue
			FROM order_items oi
			JOIN menus m ON oi.menu_id = m.id
			JOIN orders o ON oi.order_id = o.id
			WHERE o.status = 'completed'
			AND o.created_at BETWEEN ? AND ?
			GROUP BY m.id, m.name
		),
		previous_orders AS (
			SELECT 
				m.id as menu_id,
				COALESCE(COUNT(oi.id), 0) as previous_count
			FROM order_items oi
			JOIN menus m ON oi.menu_id = m.id
			JOIN orders o ON oi.order_id = o.id
			WHERE o.status = 'completed'
			AND o.created_at BETWEEN ? AND ?
			GROUP BY m.id
		)
		SELECT 
			ro.menu_name,
			ro.recent_count as count,
			ro.recent_revenue as revenue,
			COALESCE(ro.recent_count - po.previous_count, ro.recent_count) as trend
		FROM recent_orders ro
		LEFT JOIN previous_orders po ON ro.menu_id = po.menu_id
		ORDER BY ro.recent_count DESC
		LIMIT 10
	`, startDate, endDate, startDate.Add(-time.Hour*24*7), startDate).Scan(&analytics.PopularItems)

	// Query menu performance with date range
	ac.DB.Raw(`
		WITH recent_orders AS (
			SELECT 
				m.id as menu_id,
				m.name,
				COALESCE(COUNT(oi.id), 0) as recent_sold,
				COALESCE(SUM(oi.price * oi.quantity), 0) as recent_revenue
			FROM order_items oi
			JOIN menus m ON oi.menu_id = m.id
			JOIN orders o ON oi.order_id = o.id
			WHERE o.status = 'completed'
			AND o.created_at BETWEEN ? AND ?
			GROUP BY m.id, m.name
		),
		previous_orders AS (
			SELECT 
				m.id as menu_id,
				COALESCE(COUNT(oi.id), 0) as previous_sold
			FROM order_items oi
			JOIN menus m ON oi.menu_id = m.id
			JOIN orders o ON oi.order_id = o.id
			WHERE o.status = 'completed'
			AND o.created_at BETWEEN ? AND ?
			GROUP BY m.id
		)
		SELECT 
			ro.name,
			ro.recent_sold as sold,
			ro.recent_revenue as revenue,
			COALESCE(ro.recent_sold - po.previous_sold, ro.recent_sold) as trend
		FROM recent_orders ro
		LEFT JOIN previous_orders po ON ro.menu_id = po.menu_id
		ORDER BY ro.recent_revenue DESC
	`, startDate, endDate, startDate.Add(-time.Hour*24*7), startDate).Scan(&analytics.MenuPerformance)

	// Initialize empty arrays if no data
	if analytics.CategoryPerformance == nil {
		analytics.CategoryPerformance = []struct {
			Name  string  `json:"name"`
			Total float64 `json:"total"`
		}{}
	}
	if analytics.PeakHours == nil {
		analytics.PeakHours = []struct {
			Hour  int   `json:"hour"`
			Count int64 `json:"count"`
		}{}
	}
	if analytics.PopularItems == nil {
		analytics.PopularItems = []struct {
			MenuName string  `json:"menu_name"`
			Count    int     `json:"count"`
			Revenue  float64 `json:"revenue"`
			Trend    float64 `json:"trend"`
		}{}
	}
	if analytics.MenuPerformance == nil {
		analytics.MenuPerformance = []struct {
			Name    string  `json:"name"`
			Sold    int     `json:"sold"`
			Revenue float64 `json:"revenue"`
			Trend   float64 `json:"trend"`
		}{}
	}

	// Log analytics data untuk debugging
	log.Printf("Sending analytics data for period %s: %+v", period, analytics)

	utils.RespondJSON(c, http.StatusOK, "Analytics data retrieved successfully", gin.H{
		"data": analytics,
	})
}

// ExportData mengekspor data dalam format CSV
func (ac *AdminController) ExportData(c *gin.Context) {
	roleInterface, _ := c.Get("role")
	if roleInterface != "admin" {
		utils.RespondError(c, http.StatusForbidden, ErrNoPermission)
		return
	}

	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	if startDate == "" || endDate == "" {
		utils.RespondError(c, http.StatusBadRequest, errors.New("start_date dan end_date harus diisi"))
		return
	}

	// Parse dates
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, errors.New("format start_date tidak valid"))
		return
	}

	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, errors.New("format end_date tidak valid"))
		return
	}

	// Set end date to end of day
	end = end.Add(24 * time.Hour).Add(-time.Second)

	// Query data
	var orders []models.Order
	if err := ac.DB.Preload("OrderItems").Preload("OrderItems.Menu").
		Where("created_at BETWEEN ? AND ?", start, end).
		Order("created_at DESC").
		Find(&orders).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	// Set response headers for CSV
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=orders_%s_%s.csv", startDate, endDate))

	// Create CSV writer
	writer := csv.NewWriter(c.Writer)

	// Write headers
	headers := []string{"Order ID", "Tanggal", "Meja", "Total", "Status", "Item", "Jumlah", "Harga"}
	if err := writer.Write(headers); err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	// Write data
	for _, order := range orders {
		for _, item := range order.OrderItems {
			row := []string{
				fmt.Sprintf("%d", order.ID),
				order.CreatedAt.Format("2006-01-02 15:04:05"),
				order.Table.TableNumber,
				fmt.Sprintf("%.2f", order.TotalAmount),
				order.Status,
				item.Menu.Name,
				fmt.Sprintf("%d", item.Quantity),
				fmt.Sprintf("%.2f", item.Price),
			}
			if err := writer.Write(row); err != nil {
				utils.RespondError(c, http.StatusInternalServerError, err)
				return
			}
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}
}

// ExportPDF mengekspor data dalam format PDF dengan grafik
func (ac *AdminController) ExportPDF(c *gin.Context) {
	roleInterface, _ := c.Get("role")
	if roleInterface != "admin" {
		utils.RespondError(c, http.StatusForbidden, ErrNoPermission)
		return
	}

	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	if startDate == "" || endDate == "" {
		utils.RespondError(c, http.StatusBadRequest, errors.New("start_date dan end_date harus diisi"))
		return
	}

	// Parse dates
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, errors.New("format start_date tidak valid"))
		return
	}

	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, errors.New("format end_date tidak valid"))
		return
	}

	// Set end date to end of day
	end = end.Add(24 * time.Hour).Add(-time.Second)

	// Get analytics data
	analytics, err := ac.getAnalyticsData(start, end)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	// Get sales trend data
	var salesTrend []struct {
		Date   string  `json:"date"`
		Amount float64 `json:"amount"`
	}
	ac.DB.Raw(`
		WITH daily_sales AS (
			SELECT 
				DATE(created_at) as sale_date,
				COALESCE(SUM(total_amount), 0) as daily_amount
			FROM orders
			WHERE status = 'completed'
			AND created_at BETWEEN ? AND ?
			GROUP BY DATE(created_at)
		)
		SELECT 
			DATE_FORMAT(sale_date, '%Y-%m-%d') as date,
			daily_amount as amount
		FROM daily_sales
		ORDER BY sale_date ASC
	`, start, end).Scan(&salesTrend)

	// Buat data dummy untuk setiap hari dalam rentang tanggal
	currentDate := start
	dummySalesTrend := make([]struct {
		Date   string  `json:"date"`
		Amount float64 `json:"amount"`
	}, 0)

	for currentDate.Before(end) || currentDate.Equal(end) {
		dateStr := currentDate.Format("2006-01-02")
		found := false

		// Cek apakah tanggal ini ada di data asli
		for _, sale := range salesTrend {
			if sale.Date == dateStr {
				dummySalesTrend = append(dummySalesTrend, sale)
				found = true
				break
			}
		}

		// Jika tidak ada data untuk tanggal ini, tambahkan data dummy
		if !found {
			dummySalesTrend = append(dummySalesTrend, struct {
				Date   string  `json:"date"`
				Amount float64 `json:"amount"`
			}{
				Date:   dateStr,
				Amount: 0,
			})
		}

		currentDate = currentDate.AddDate(0, 0, 1)
	}

	// Gunakan data dummy yang sudah dibuat
	salesTrend = dummySalesTrend

	// Create sales trend chart with updated data
	salesChart := chart.Chart{
		Title: "Tren Penjualan",
		TitleStyle: chart.Style{
			FontSize: 14,
		},
		Background: chart.Style{
			Padding: chart.Box{
				Top:    20,
				Left:   20,
				Right:  20,
				Bottom: 20,
			},
		},
		XAxis: chart.XAxis{
			Name:           "Tanggal",
			NameStyle:      chart.Style{},
			Style:          chart.Style{},
			ValueFormatter: chart.TimeValueFormatter,
			GridMajorStyle: chart.Style{
				StrokeColor: chart.ColorAlternateGray,
				StrokeWidth: 1.0,
			},
		},
		YAxis: chart.YAxis{
			Name:      "Total Penjualan (Rp)",
			NameStyle: chart.Style{},
			Style:     chart.Style{},
			GridMajorStyle: chart.Style{
				StrokeColor: chart.ColorAlternateGray,
				StrokeWidth: 1.0,
			},
		},
		Series: []chart.Series{
			chart.TimeSeries{
				Name: "Penjualan",
				Style: chart.Style{
					StrokeColor: chart.ColorBlue,
					FillColor:   chart.ColorBlue.WithAlpha(100),
				},
				XValues: func() []time.Time {
					times := make([]time.Time, len(salesTrend))
					for i, sale := range salesTrend {
						t, _ := time.Parse("2006-01-02", sale.Date)
						times[i] = t
					}
					return times
				}(),
				YValues: func() []float64 {
					values := make([]float64, len(salesTrend))
					for i, sale := range salesTrend {
						values[i] = sale.Amount
					}
					return values
				}(),
			},
		},
	}

	// Get category performance data
	var categoryPerformance []struct {
		Name  string  `json:"name"`
		Total float64 `json:"total"`
	}
	ac.DB.Raw(`
		SELECT c.name, COALESCE(SUM(oi.price * oi.quantity), 0) as total
		FROM order_items oi
		JOIN menus m ON oi.menu_id = m.id
		JOIN menu_categories c ON m.category_id = c.id
		JOIN orders o ON oi.order_id = o.id
		WHERE o.status = 'completed'
		AND o.created_at BETWEEN ? AND ?
		GROUP BY c.id, c.name
		ORDER BY total DESC
	`, start, end).Scan(&categoryPerformance)

	// Jika tidak ada data kategori, buat data dummy
	if len(categoryPerformance) == 0 {
		categoryPerformance = append(categoryPerformance, struct {
			Name  string  `json:"name"`
			Total float64 `json:"total"`
		}{
			Name:  "Tidak ada data",
			Total: 0,
		})
	}

	// Get peak hours data
	var peakHours []struct {
		Hour  int   `json:"hour"`
		Count int64 `json:"count"`
	}
	ac.DB.Raw(`
		SELECT EXTRACT(HOUR FROM created_at) as hour, COUNT(*) as count
		FROM orders
		WHERE status = 'completed'
		AND created_at BETWEEN ? AND ?
		GROUP BY EXTRACT(HOUR FROM created_at)
		ORDER BY hour ASC
	`, start, end).Scan(&peakHours)

	// Jika tidak ada data jam ramai, buat data dummy
	if len(peakHours) == 0 {
		peakHours = append(peakHours, struct {
			Hour  int   `json:"hour"`
			Count int64 `json:"count"`
		}{
			Hour:  0,
			Count: 0,
		})
		peakHours = append(peakHours, struct {
			Hour  int   `json:"hour"`
			Count int64 `json:"count"`
		}{
			Hour:  23,
			Count: 0,
		})
	}

	// Create category performance chart
	categoryChart := chart.Chart{
		Title: "Performa Kategori",
		TitleStyle: chart.Style{
			FontSize: 14,
		},
		Background: chart.Style{
			Padding: chart.Box{
				Top:    20,
				Left:   20,
				Right:  20,
				Bottom: 20,
			},
		},
		Series: []chart.Series{
			chart.ContinuousSeries{
				Name: "Kategori",
				Style: chart.Style{
					StrokeColor: chart.ColorBlue,
					FillColor:   chart.ColorBlue.WithAlpha(100),
				},
				XValues: func() []float64 {
					values := make([]float64, len(categoryPerformance))
					for i := range categoryPerformance {
						values[i] = float64(i)
					}
					return values
				}(),
				YValues: func() []float64 {
					values := make([]float64, len(categoryPerformance))
					for i, cat := range categoryPerformance {
						values[i] = cat.Total
					}
					return values
				}(),
			},
		},
		XAxis: chart.XAxis{
			Name:      "Kategori",
			NameStyle: chart.Style{},
			Style:     chart.Style{},
			ValueFormatter: func(v interface{}) string {
				if idx, ok := v.(float64); ok {
					if int(idx) < len(categoryPerformance) {
						return categoryPerformance[int(idx)].Name
					}
				}
				return ""
			},
			GridMajorStyle: chart.Style{
				StrokeColor: chart.ColorAlternateGray,
				StrokeWidth: 1.0,
			},
		},
		YAxis: chart.YAxis{
			Name:      "Total Penjualan (Rp)",
			NameStyle: chart.Style{},
			Style:     chart.Style{},
			GridMajorStyle: chart.Style{
				StrokeColor: chart.ColorAlternateGray,
				StrokeWidth: 1.0,
			},
		},
	}

	// Create peak hours chart
	peakHoursChart := chart.Chart{
		Title: "Jam Ramai",
		TitleStyle: chart.Style{
			FontSize: 14,
		},
		Background: chart.Style{
			Padding: chart.Box{
				Top:    20,
				Left:   20,
				Right:  20,
				Bottom: 20,
			},
		},
		XAxis: chart.XAxis{
			Name:      "Jam",
			NameStyle: chart.Style{},
			Style:     chart.Style{},
			ValueFormatter: func(v interface{}) string {
				if h, ok := v.(float64); ok {
					return fmt.Sprintf("%02d:00", int(h))
				}
				return ""
			},
			GridMajorStyle: chart.Style{
				StrokeColor: chart.ColorAlternateGray,
				StrokeWidth: 1.0,
			},
		},
		YAxis: chart.YAxis{
			Name:      "Jumlah Pesanan",
			NameStyle: chart.Style{},
			Style:     chart.Style{},
			GridMajorStyle: chart.Style{
				StrokeColor: chart.ColorAlternateGray,
				StrokeWidth: 1.0,
			},
		},
		Series: []chart.Series{
			chart.ContinuousSeries{
				Name: "Jumlah Pesanan",
				Style: chart.Style{
					StrokeColor: chart.ColorBlue,
					FillColor:   chart.ColorBlue.WithAlpha(100),
				},
				XValues: func() []float64 {
					values := make([]float64, len(peakHours))
					for i, ph := range peakHours {
						values[i] = float64(ph.Hour)
					}
					return values
				}(),
				YValues: func() []float64 {
					values := make([]float64, len(peakHours))
					for i, ph := range peakHours {
						values[i] = float64(ph.Count)
					}
					return values
				}(),
			},
		},
	}

	// Render charts to PNG
	var salesChartBuffer, categoryChartBuffer, peakHoursChartBuffer bytes.Buffer
	if err := salesChart.Render(chart.PNG, &salesChartBuffer); err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}
	if err := categoryChart.Render(chart.PNG, &categoryChartBuffer); err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}
	if err := peakHoursChart.Render(chart.PNG, &peakHoursChartBuffer); err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	// Create temporary files for charts
	salesTmpFile, err := os.CreateTemp("", "sales-chart-*.png")
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}
	defer os.Remove(salesTmpFile.Name())

	categoryTmpFile, err := os.CreateTemp("", "category-chart-*.png")
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}
	defer os.Remove(categoryTmpFile.Name())

	peakHoursTmpFile, err := os.CreateTemp("", "peak-hours-chart-*.png")
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}
	defer os.Remove(peakHoursTmpFile.Name())

	// Write charts to temporary files
	if _, err := salesTmpFile.Write(salesChartBuffer.Bytes()); err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}
	if _, err := categoryTmpFile.Write(categoryChartBuffer.Bytes()); err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}
	if _, err := peakHoursTmpFile.Write(peakHoursChartBuffer.Bytes()); err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	// Create PDF
	pdf := fpdf.New("P", "mm", "A4", "")

	// Set margins
	pdf.SetMargins(15, 15, 15)

	// Add first page
	pdf.AddPage()

	// Add header with background color
	pdf.SetFillColor(44, 62, 80)
	pdf.Rect(0, 0, 210, 40, "F")

	// Add title
	pdf.SetFont("Arial", "B", 24)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetY(15)
	pdf.Cell(0, 10, "LAPORAN PENJUALAN")

	// Add date range
	pdf.SetFont("Arial", "", 12)
	pdf.SetY(25)
	pdf.Cell(0, 10, fmt.Sprintf("Periode: %s s/d %s", startDate, endDate))

	// Reset text color
	pdf.SetTextColor(0, 0, 0)

	// Move to content area
	pdf.SetY(50)

	// Add summary section with box
	pdf.SetFillColor(245, 247, 250)
	pdf.SetDrawColor(200, 200, 200)
	pdf.Rect(15, pdf.GetY(), 180, 50, "FD")

	// Add summary title
	pdf.SetFont("Arial", "B", 14)
	pdf.SetTextColor(44, 62, 80)
	pdf.SetXY(20, pdf.GetY()+5)
	pdf.Cell(0, 10, "Ringkasan")

	// Add summary content
	pdf.SetFont("Arial", "", 12)
	pdf.SetTextColor(100, 100, 100)
	pdf.SetXY(25, pdf.GetY()+15)
	pdf.Cell(0, 10, fmt.Sprintf("Total Penjualan: Rp %s", utils.FormatCurrency(analytics.TotalSales)))
	pdf.SetXY(25, pdf.GetY()+10)
	pdf.Cell(0, 10, fmt.Sprintf("Total Pesanan: %d", analytics.TotalOrders))
	pdf.SetXY(120, pdf.GetY()-10)
	pdf.Cell(0, 10, fmt.Sprintf("Rata-rata Pesanan: Rp %s", utils.FormatCurrency(analytics.AverageOrder)))

	// Move to charts section
	pdf.SetY(pdf.GetY() + 30)

	// Add charts section title
	pdf.SetFont("Arial", "B", 14)
	pdf.SetTextColor(44, 62, 80)
	pdf.Cell(0, 10, "Grafik Analisis")
	pdf.Ln(15)

	// Add sales trend chart with box
	pdf.SetFillColor(255, 255, 255)
	pdf.SetDrawColor(200, 200, 200)
	pdf.Rect(15, pdf.GetY(), 180, 200, "FD")
	pdf.SetFont("Arial", "B", 12)
	pdf.SetXY(20, pdf.GetY()+5)
	pdf.Cell(0, 10, "Tren Penjualan")
	pdf.ImageOptions(salesTmpFile.Name(), 15, pdf.GetY()+15, 180, 180, false, fpdf.ImageOptions{ImageType: "PNG"}, 0, "")
	pdf.Ln(220)

	// Add new page for category performance chart
	pdf.AddPage()
	pdf.SetFillColor(255, 255, 255)
	pdf.Rect(15, pdf.GetY(), 180, 200, "FD")
	pdf.SetXY(20, pdf.GetY()+5)
	pdf.Cell(0, 10, "Performa Kategori")
	pdf.ImageOptions(categoryTmpFile.Name(), 15, pdf.GetY()+15, 180, 180, false, fpdf.ImageOptions{ImageType: "PNG"}, 0, "")
	pdf.Ln(220)

	// Add new page for peak hours chart
	pdf.AddPage()
	pdf.SetFillColor(255, 255, 255)
	pdf.Rect(15, pdf.GetY(), 180, 200, "FD")
	pdf.SetXY(20, pdf.GetY()+5)
	pdf.Cell(0, 10, "Jam Ramai")
	pdf.ImageOptions(peakHoursTmpFile.Name(), 15, pdf.GetY()+15, 180, 180, false, fpdf.ImageOptions{ImageType: "PNG"}, 0, "")
	pdf.Ln(220)

	// Add footer with more space
	pdf.SetY(-30)
	pdf.SetFont("Arial", "I", 8)
	pdf.SetTextColor(150, 150, 150)
	pdf.Cell(0, 10, fmt.Sprintf("Dicetak pada: %s", time.Now().Format("02/01/2006 15:04:05")))
	pdf.Ln(5)
	pdf.Cell(0, 10, "© 2024 Restaurant App. All rights reserved.")

	// Set response headers
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=report_%s_%s.pdf", startDate, endDate))

	// Output PDF
	if err := pdf.Output(c.Writer); err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}
}

// Helper function untuk mendapatkan data analitik
func (ac *AdminController) getAnalyticsData(start, end time.Time) (struct {
	TotalSales   float64
	TotalOrders  int64
	AverageOrder float64
}, error) {
	var result struct {
		TotalSales   float64
		TotalOrders  int64
		AverageOrder float64
	}

	// Query total orders
	if err := ac.DB.Model(&models.Order{}).
		Where("created_at BETWEEN ? AND ?", start, end).
		Count(&result.TotalOrders).Error; err != nil {
		return result, err
	}

	// Query total sales
	if err := ac.DB.Model(&models.Order{}).
		Where("created_at BETWEEN ? AND ?", start, end).
		Select("COALESCE(SUM(total_amount), 0)").
		Row().Scan(&result.TotalSales); err != nil {
		return result, err
	}

	// Calculate average order
	if result.TotalOrders > 0 {
		result.AverageOrder = result.TotalSales / float64(result.TotalOrders)
	}

	return result, nil
}
