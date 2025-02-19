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

// GetAllOrders -> list orders beserta items
func (oc *OrderController) GetAllOrders(c *gin.Context) {
	var orders []models.Order
	if err := oc.DB.Preload("OrderItems").Find(&orders).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}
	utils.RespondJSON(c, http.StatusOK, "List of orders", orders)
}

// CreateOrder -> buat order (status='pending_payment')
func (oc *OrderController) CreateOrder(c *gin.Context) {
	tableID := c.Param("table_id")

	// Cek sesi customer Aktif
	var customer models.Customer
	if err := oc.DB.Where("table_id = ? AND status = ?", tableID, "active").First(&customer).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, fmt.Errorf("tidak ada sesi aktif di meja ini"))
		return
	}

	type ItemReq struct {
		MenuID       uint   `json:"menu_id"`
		Quantity     int    `json:"quantity"`
		Notes        string `json:"notes"`
		ParentItemID *uint  `json:"parent_item_id,omitempty"` // untuk add-on
	}

	type ReqBody struct {
		Items []ItemReq `json:"items" binding:"required"`
	}

	var body ReqBody
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err)
		return
	}

	// Buat order dengan status pending_payment
	order := models.Order{
		CustomerID:  customer.ID,
		Status:      "pending_payment",
		TotalAmount: 0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := oc.DB.Create(&order).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	var total float64
	// Loop item
	for _, item := range body.Items {
		// Ambil menu untuk harga
		var menu models.Menu
		if err := oc.DB.First(&menu, item.MenuID).Error; err != nil {
			// skip jika tak ketemu
			continue
		}
		subTotal := float64(item.Quantity) * menu.Price
		total += subTotal

		orderItem := models.OrderItem{
			OrderID:      order.ID,
			MenuID:       menu.ID,
			Quantity:     item.Quantity,
			Price:        menu.Price,
			Notes:        item.Notes,
			ParentItemID: item.ParentItemID, // untuk add-on
			Status:       "pending",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		oc.DB.Create(&orderItem)
	}

	// Update total
	order.TotalAmount = total
	oc.DB.Save(&order)

	utils.RespondJSON(c, http.StatusCreated, "Order created", order)
}

// GetOrderByID -> detail 1 order
func (oc *OrderController) GetOrderByID(c *gin.Context) {
	idStr := c.Param("order_id")
	id, _ := strconv.Atoi(idStr)

	var order models.Order
	fmt.Printf("Order %d has %d items\n", order.ID, len(order.OrderItems))
	// preload items
	if err := oc.DB.Preload("OrderItems").First(&order, id).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, err)
		return
	}

	utils.RespondJSON(c, http.StatusOK, "Order detail", order)
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

	var item models.OrderItem
	if err := oc.DB.First(&item, itemID).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, err)
		return
	}

	if item.Status != "pending" {
		utils.RespondError(c, http.StatusBadRequest, fmt.Errorf("Item not in pending status"))
		return
	}

	item.Status = "in_progress"
	item.UpdatedAt = time.Now()
	oc.DB.Save(&item)

	utils.RespondJSON(c, http.StatusOK, "Item in_progress", item)
}

// FinishCookingItem -> Chef menandai 1 item => "ready".
// Jika semua item di order => "ready", order => "ready".
func (oc *OrderController) FinishCookingItem(c *gin.Context) {
	itemID := c.Param("item_id")

	var item models.OrderItem
	if err := oc.DB.First(&item, itemID).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, err)
		return
	}

	if item.Status != "in_progress" {
		utils.RespondError(c, http.StatusBadRequest, fmt.Errorf("Item not in in_progress status"))
		return
	}

	item.Status = "ready"
	item.UpdatedAt = time.Now()
	oc.DB.Save(&item)

	// Cek apakah semua item di order ini => "ready"
	var countNotReady int64
	oc.DB.Model(&models.OrderItem{}).
		Where("order_id = ? AND status != ?", item.OrderID, "ready").
		Count(&countNotReady)

	if countNotReady == 0 {
		var order models.Order
		if err := oc.DB.First(&order, item.OrderID).Error; err == nil {
			order.Status = "ready"
			now := time.Now()
			order.FinishCookingTime = &now
			oc.DB.Save(&order)

			// Broadcast ke semua
			kds.BroadcastOrderUpdate(order)
			// Notifikasi khusus staff
			kds.BroadcastStaffNotification(fmt.Sprintf("Order #%d siap disajikan", order.ID))
		}
	}

	utils.RespondJSON(c, http.StatusOK, "Item finished", item)
}

// StartCooking -> Chef menandai entire order => "in_progress" (opsional)
func (oc *OrderController) StartCooking(c *gin.Context) {
	orderID := c.Param("order_id")

	var order models.Order
	if err := oc.DB.First(&order, orderID).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, err)
		return
	}

	// Pastikan status='paid' => boleh "in_progress"
	if order.Status != "paid" {
		utils.RespondError(c, http.StatusBadRequest, fmt.Errorf("order not in paid status"))
		return
	}

	now := time.Now()
	order.Status = "in_progress"
	order.StartCookingTime = &now

	oc.DB.Save(&order)

	// broadcast
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
		utils.RespondError(c, http.StatusBadRequest, fmt.Errorf("Order not in ready status"))
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
		Where("status IN ?", []string{"paid", "in_progress", "ready"}).
		Order("created_at asc").
		Find(&orders).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
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
