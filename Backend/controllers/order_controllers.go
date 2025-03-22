package controllers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/yeremiapane/restaurant-app/models"
	"github.com/yeremiapane/restaurant-app/utils"

	// import kds untuk broadcast (jika masih ingin menyiarkan event ke websocket)
	"github.com/yeremiapane/restaurant-app/kds"
)

type OrderController struct {
	DB *gorm.DB
}

func NewOrderController(db *gorm.DB) *OrderController {
	return &OrderController{DB: db}
}

// GetAllOrders mengembalikan semua orders
func (oc *OrderController) GetAllOrders(c *gin.Context) {
	var orders []models.Order

	result := oc.DB.
		Preload("Customer").
		Preload("Chef").
		Preload("Table").
		Preload("OrderItems").
		Preload("OrderItems.Menu").
		Find(&orders)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  false,
			"message": result.Error.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  true,
		"message": "Orders retrieved successfully",
		"data":    orders,
	})
}

// CreateOrder -> buat order baru
func (oc *OrderController) CreateOrder(c *gin.Context) {
	var req struct {
		TableID     uint    `json:"table_id" binding:"required"`
		CustomerID  uint    `json:"customer_id" binding:"required"`
		SessionKey  string  `json:"session_key" binding:"required"`
		Status      string  `json:"status"`
		TotalAmount float64 `json:"total_amount"`
		Items       []struct {
			MenuID   uint    `json:"menu_id" binding:"required"`
			Quantity int     `json:"quantity" binding:"required,min=1"`
			Price    float64 `json:"price"`
			Notes    string  `json:"notes"`
			Status   string  `json:"status"`
		} `json:"Items" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  false,
			"message": err.Error(),
		})
		return
	}

	// Cek customer
	var customer models.Customer
	if err := oc.DB.First(&customer, req.CustomerID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  false,
			"message": "Customer not found",
		})
		return
	}

	// Validasi session key
	if customer.SessionKey == nil || *customer.SessionKey != req.SessionKey {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  false,
			"message": "Invalid session key",
		})
		return
	}

	// Cek table
	var table models.Table
	if err := oc.DB.First(&table, req.TableID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  false,
			"message": "Table not found",
		})
		return
	}

	// Buat order baru
	order := models.Order{
		TableID:     req.TableID,
		CustomerID:  req.CustomerID,
		Status:      "pending_payment",
		TotalAmount: req.TotalAmount,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	tx := oc.DB.Begin()

	if err := tx.Create(&order).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  false,
			"message": err.Error(),
		})
		return
	}

	// Proses items
	for _, item := range req.Items {
		var menu models.Menu
		if err := tx.First(&menu, item.MenuID).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusNotFound, gin.H{
				"status":  false,
				"message": fmt.Sprintf("Menu ID %d not found", item.MenuID),
			})
			return
		}

		orderItem := models.OrderItem{
			OrderID:   order.ID,
			MenuID:    menu.ID,
			Quantity:  item.Quantity,
			Price:     item.Price,
			Notes:     item.Notes,
			Status:    "pending",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if err := tx.Create(&orderItem).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  false,
				"message": err.Error(),
			})
			return
		}
	}

	tx.Commit()

	// Broadcast update
	kds.BroadcastOrderUpdate(order)

	c.JSON(http.StatusCreated, gin.H{
		"status":  true,
		"message": "Order created successfully",
		"data":    order,
	})
}

// GetOrderByID -> detail 1 order
func (oc *OrderController) GetOrderByID(c *gin.Context) {
	idStr := c.Param("order_id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  false,
			"message": "Invalid order ID",
		})
		return
	}

	var order models.Order
	result := oc.DB.Preload("Customer").
		Preload("Chef").
		Preload("Table").
		Preload("OrderItems").
		Preload("OrderItems.Menu").
		First(&order, id)

	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  false,
			"message": "Order not found",
		})
		return
	}

	// Log untuk debugging
	utils.InfoLogger.Printf("Retrieved order #%d with %d items", order.ID, len(order.OrderItems))

	c.JSON(http.StatusOK, gin.H{
		"status":  true,
		"message": "Order detail retrieved successfully",
		"data":    order,
	})
}

// UpdateOrder untuk admin/staff mengupdate order
func (oc *OrderController) UpdateOrder(c *gin.Context) {
	roleInterface, _ := c.Get("role")
	if roleInterface != "admin" && roleInterface != "staff" {
		utils.RespondError(c, http.StatusForbidden, ErrNoPermission)
		return
	}

	orderID := c.Param("order_id")

	var order models.Order
	if err := oc.DB.Preload("OrderItems").First(&order, orderID).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, err)
		return
	}

	type UpdateReq struct {
		Status *string `json:"status"`
		Items  []struct {
			ID       uint    `json:"id"`
			Status   string  `json:"status"`
			Quantity *int    `json:"quantity"`
			Notes    *string `json:"notes"`
		} `json:"items"`
	}

	var req UpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err)
		return
	}

	tx := oc.DB.Begin()

	// Update order status if provided
	if req.Status != nil {
		order.Status = *req.Status
		if err := tx.Save(&order).Error; err != nil {
			tx.Rollback()
			utils.RespondError(c, http.StatusInternalServerError, err)
			return
		}
	}

	// Update items if provided
	for _, itemUpdate := range req.Items {
		var item models.OrderItem
		if err := tx.First(&item, itemUpdate.ID).Error; err != nil {
			continue
		}

		item.Status = itemUpdate.Status
		if itemUpdate.Quantity != nil {
			item.Quantity = *itemUpdate.Quantity
		}
		if itemUpdate.Notes != nil {
			item.Notes = *itemUpdate.Notes
		}

		if err := tx.Save(&item).Error; err != nil {
			tx.Rollback()
			utils.RespondError(c, http.StatusInternalServerError, err)
			return
		}
	}

	tx.Commit()

	// Dapatkan data dashboard terbaru
	tableCtrl := &TableController{DB: oc.DB}
	dashboardData := tableCtrl.getDashboardData()

	// Broadcast update
	kds.BroadcastMessage(kds.Message{
		Event: kds.EventOrderUpdate,
		Data:  dashboardData,
	})

	// Broadcast updates
	kds.BroadcastOrderUpdate(order)
	kds.BroadcastStaffNotification(fmt.Sprintf("Order #%d updated by %s", order.ID, roleInterface))

	utils.RespondJSON(c, http.StatusOK, "Order updated", order)
}

// DeleteOrder
func (oc *OrderController) DeleteOrder(c *gin.Context) {
	idStr := c.Param("order_id")
	id, _ := strconv.Atoi(idStr)

	if err := oc.DB.Delete(&models.Order{}, id).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}
	utils.RespondJSON(c, http.StatusOK, "Order deleted", gin.H{"order_id": id})
}

/*
========================================
 ITEM-LEVEL COOKING
========================================
*/

// /////////////////////////////////////////////////////////////////
// StartCookingItem -> Chef menandai 1 item dari "pending" => "in_progress"
func (oc *OrderController) StartCookingItem(c *gin.Context) {
	itemID := c.Param("item_id")

	// Dapatkan ID chef yang sedang login
	userID, exists := c.Get("user_id")
	if !exists {
		utils.RespondError(c, http.StatusUnauthorized, fmt.Errorf("unauthorized"))
		return
	}

	var item models.OrderItem
	if err := oc.DB.Preload("Order").First(&item, itemID).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, err)
		return
	}

	if item.Status != "pending" {
		utils.RespondError(c, http.StatusBadRequest, fmt.Errorf("item not in pending status"))
		return
	}

	// Cek apakah order sudah ditangani chef lain
	if item.Order.ChefID != nil && *item.Order.ChefID != userID.(uint) {
		chefName := "another chef"
		var chef models.User
		if err := oc.DB.First(&chef, *item.Order.ChefID); err == nil {
			chefName = chef.Name
		}
		utils.RespondError(c, http.StatusBadRequest, fmt.Errorf("order is already being handled by %s", chefName))
		return
	}

	tx := oc.DB.Begin()

	// Update item status
	item.Status = "in_progress"
	item.UpdatedAt = time.Now()
	if err := tx.Save(&item).Error; err != nil {
		tx.Rollback()
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	// Jika order dalam status "paid", update ke "in_progress" tanpa mengubah status item lain
	if item.Order.Status == "paid" {
		var order models.Order
		if err := tx.First(&order, item.OrderID).Error; err != nil {
			tx.Rollback()
			utils.RespondError(c, http.StatusInternalServerError, err)
			return
		}

		now := time.Now()
		order.Status = "in_progress"
		order.StartCookingTime = &now
		order.UpdatedAt = now

		// Tetapkan chef_id jika belum diatur
		userIDUint := userID.(uint)
		if order.ChefID == nil {
			order.ChefID = &userIDUint
		}

		if err := tx.Save(&order).Error; err != nil {
			tx.Rollback()
			utils.RespondError(c, http.StatusInternalServerError, err)
			return
		}

		// Broadcast update status order
		kds.BroadcastOrderUpdate(order)
	}

	tx.Commit()

	// Reload item dengan relasi
	var updatedItem models.OrderItem
	oc.DB.Preload("Order").Preload("Menu").First(&updatedItem, itemID)

	utils.RespondJSON(c, http.StatusOK, "Item in_progress", updatedItem)
}

// FinishCookingItem -> Chef menandai 1 item => "ready".
// Jika semua item di order => "ready", order => "ready".
func (oc *OrderController) FinishCookingItem(c *gin.Context) {
	itemID := c.Param("item_id")

	var item models.OrderItem
	if err := oc.DB.Preload("Order").First(&item, itemID).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, err)
		return
	}

	if item.Status != "in_progress" {
		utils.RespondError(c, http.StatusBadRequest, fmt.Errorf("item not in in_progress status"))
		return
	}

	tx := oc.DB.Begin()

	// Update item status
	item.Status = "ready"
	item.UpdatedAt = time.Now()
	if err := tx.Save(&item).Error; err != nil {
		tx.Rollback()
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	// Cek apakah semua item di order ini => "ready"
	var countNotReady int64
	if err := tx.Model(&models.OrderItem{}).
		Where("order_id = ? AND status != ?", item.OrderID, "ready").
		Count(&countNotReady).Error; err != nil {
		tx.Rollback()
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	if countNotReady == 0 {
		var order models.Order
		if err := tx.First(&order, item.OrderID).Error; err != nil {
			tx.Rollback()
			utils.RespondError(c, http.StatusInternalServerError, err)
			return
		}

		now := time.Now()
		order.Status = "ready"
		order.FinishCookingTime = &now
		order.UpdatedAt = now

		if err := tx.Save(&order).Error; err != nil {
			tx.Rollback()
			utils.RespondError(c, http.StatusInternalServerError, err)
			return
		}

		// Broadcast ke semua
		kds.BroadcastOrderUpdate(order)
		// Notifikasi khusus staff
		kds.BroadcastStaffNotification(fmt.Sprintf("Order #%d siap disajikan", order.ID))
	}

	tx.Commit()

	// Reload item dengan relasi
	var updatedItem models.OrderItem
	oc.DB.Preload("Order").Preload("Menu").First(&updatedItem, itemID)

	utils.RespondJSON(c, http.StatusOK, "Item finished", updatedItem)
}

// StartCooking -> Chef menandai entire order => "in_progress" (opsional)
func (oc *OrderController) StartCooking(c *gin.Context) {
	orderID := c.Param("order_id")

	// Dapatkan ID chef yang sedang login
	userID, exists := c.Get("user_id")
	if !exists {
		utils.RespondError(c, http.StatusUnauthorized, fmt.Errorf("unauthorized"))
		return
	}

	var order models.Order
	if err := oc.DB.Preload("OrderItems").First(&order, orderID).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, err)
		return
	}

	// Pastikan status='paid' => boleh "in_progress"
	if order.Status != "paid" {
		utils.RespondError(c, http.StatusBadRequest, fmt.Errorf("order not in paid status"))
		return
	}

	// Cek apakah order sudah ditangani chef lain
	if order.ChefID != nil && *order.ChefID != userID.(uint) {
		chefName := "another chef"
		var chef models.User
		if err := oc.DB.First(&chef, *order.ChefID); err == nil {
			chefName = chef.Name
		}
		utils.RespondError(c, http.StatusBadRequest, fmt.Errorf("order is already being handled by %s", chefName))
		return
	}

	tx := oc.DB.Begin()

	// Update order status
	now := time.Now()
	order.Status = "in_progress"
	order.StartCookingTime = &now
	order.UpdatedAt = now

	// Tetapkan chef_id
	userIDUint := userID.(uint)
	order.ChefID = &userIDUint

	if err := tx.Save(&order).Error; err != nil {
		tx.Rollback()
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	// Update semua item yang masih pending menjadi in_progress
	for _, item := range order.OrderItems {
		if item.Status == "pending" {
			item.Status = "in_progress"
			item.UpdatedAt = now
			if err := tx.Save(&item).Error; err != nil {
				tx.Rollback()
				utils.RespondError(c, http.StatusInternalServerError, err)
				return
			}
		}
	}

	tx.Commit()

	// Reload order with relationships
	oc.DB.Preload("OrderItems").Preload("OrderItems.Menu").Preload("Customer").Preload("Table").Preload("Chef").First(&order, orderID)

	// Broadcast
	kds.BroadcastOrderUpdate(order)

	utils.RespondJSON(c, http.StatusOK, "Order in progress", order)
}

// FinishCooking -> Chef menandai entire order => "ready" (opsional)
func (oc *OrderController) FinishCooking(c *gin.Context) {
	orderID := c.Param("order_id")

	var order models.Order
	if err := oc.DB.First(&order, orderID).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, err)
		return
	}

	if order.Status != "in_progress" {
		utils.RespondError(c, http.StatusBadRequest, fmt.Errorf("order not in in_progress status"))
		return
	}

	now := time.Now()
	order.Status = "ready"
	order.FinishCookingTime = &now
	oc.DB.Save(&order)

	kds.BroadcastOrderUpdate(order)

	utils.RespondJSON(c, http.StatusOK, "Order is ready", order)
}

// CompleteOrder -> staff menandai order "completed"
func (oc *OrderController) CompleteOrder(c *gin.Context) {
	orderID := c.Param("order_id")

	var order models.Order
	if err := oc.DB.First(&order, orderID).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, err)
		return
	}

	// Pastikan status='ready'
	if order.Status != "ready" {
		utils.RespondError(c, http.StatusBadRequest, fmt.Errorf("order not in ready status"))
		return
	}

	order.Status = "completed"
	order.UpdatedAt = time.Now()
	oc.DB.Save(&order)

	utils.RespondJSON(c, http.StatusOK, "Order completed", order)
}

// GetPendingItems khusus untuk Chef - menampilkan item yang perlu dimasak
func (oc *OrderController) GetPendingItems(c *gin.Context) {
	// Cek role
	roleInterface, _ := c.Get("role")
	if roleInterface != "chef" {
		utils.RespondError(c, http.StatusForbidden, ErrNoPermission)
		return
	}

	var items []models.OrderItem
	if err := oc.DB.Preload("Menu").
		Preload("Order").
		Where("status = ?", "pending").
		Order("created_at asc").
		Find(&items).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	utils.RespondJSON(c, http.StatusOK, "Pending items", items)
}

// GetKitchenDisplay khusus untuk Chef & Staff - overview dapur
func (oc *OrderController) GetKitchenDisplay(c *gin.Context) {
	role, _ := c.Get("role")
	if role != "chef" && role != "staff" {
		utils.RespondError(c, http.StatusForbidden, ErrNoPermission)
		return
	}

	var orders []models.Order
	if err := oc.DB.Preload("OrderItems").
		Preload("OrderItems.Menu").
		Preload("Customer").
		Preload("Table").
		Preload("Chef").
		Where("status IN ?", []string{"paid", "in_progress", "ready"}).
		Order("created_at asc").
		Find(&orders).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	// Hanya sinkronisasi item status dengan order status jika diperlukan untuk pesanan 'ready'
	for i := range orders {
		// Update status item untuk order dengan status 'ready' tapi item masih 'pending'
		if orders[i].Status == "ready" {
			for j := range orders[i].OrderItems {
				if orders[i].OrderItems[j].Status != "ready" {
					// Update item status ke ready
					orders[i].OrderItems[j].Status = "ready"
					oc.DB.Save(&orders[i].OrderItems[j])
				}
			}
		}
	}

	utils.RespondJSON(c, http.StatusOK, "Kitchen display orders", orders)
}

// GetOrderAnalytics untuk admin melihat analisis order
func (oc *OrderController) GetOrderAnalytics(c *gin.Context) {
	roleInterface, _ := c.Get("role")
	if roleInterface != "admin" {
		utils.RespondError(c, http.StatusForbidden, ErrNoPermission)
		return
	}

	var analytics struct {
		PopularItems []struct {
			MenuID   uint    `json:"menu_id"`
			MenuName string  `json:"menu_name"`
			Count    int     `json:"count"`
			Revenue  float64 `json:"revenue"`
		} `json:"popular_items"`
		AveragePrepTime float64 `json:"average_prep_time"`
		PeakHours       []struct {
			Hour  int   `json:"hour"`
			Count int64 `json:"count"`
		} `json:"peak_hours"`
	}

	// Query popular items
	oc.DB.Raw(`
		SELECT m.id as menu_id, m.name as menu_name, 
		COUNT(oi.id) as count, SUM(oi.price * oi.quantity) as revenue
		FROM order_items oi
		JOIN menus m ON oi.menu_id = m.id
		GROUP BY m.id, m.name
		ORDER BY count DESC
		LIMIT 10
	`).Scan(&analytics.PopularItems)

	// Calculate average prep time
	oc.DB.Model(&models.Order{}).
		Where("finish_cooking_time IS NOT NULL").
		Select("AVG(EXTRACT(EPOCH FROM (finish_cooking_time - start_cooking_time)))").
		Row().Scan(&analytics.AveragePrepTime)

	// Get peak hours
	oc.DB.Raw(`
		SELECT EXTRACT(HOUR FROM created_at) as hour, COUNT(*) as count
		FROM orders
		GROUP BY EXTRACT(HOUR FROM created_at)
		ORDER BY count DESC
	`).Scan(&analytics.PeakHours)

	utils.RespondJSON(c, http.StatusOK, "Order analytics", analytics)
}
