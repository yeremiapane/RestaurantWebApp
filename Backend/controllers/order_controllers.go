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

// CreateOrder -> buat order (status='draft'), item => 'pending'
func (oc *OrderController) CreateOrder(c *gin.Context) {
	type ItemReq struct {
		MenuID       uint   `json:"menu_id"`
		Quantity     int    `json:"quantity"`
		Notes        string `json:"notes"`
		ParentItemID *uint  `json:"parent_item_id,omitempty"`
	}
	type ReqBody struct {
		CustomerID uint      `json:"customer_id" binding:"required"`
		Items      []ItemReq `json:"items" binding:"required"`
	}

	var body ReqBody
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err)
		return
	}

	// Buat order => 'draft'
	order := models.Order{
		CustomerID:  body.CustomerID,
		Status:      "draft",
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
			ParentItemID: item.ParentItemID,
			Status:       "pending", // item-level status
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		oc.DB.Create(&orderItem)
	}

	// Update total
	order.TotalAmount = total
	oc.DB.Save(&order)

	utils.RespondJSON(c, http.StatusCreated, "Order created (draft)", order)
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

// UpdateOrder -> menambahkan item baru / ubah status
func (oc *OrderController) UpdateOrder(c *gin.Context) {
	idStr := c.Param("order_id")
	id, _ := strconv.Atoi(idStr)

	type itemReq struct {
		MenuID   uint `json:"menu_id"`
		Quantity int  `json:"quantity"`
	}
	type reqBody struct {
		Status string    `json:"status"`
		Items  []itemReq `json:"items"`
	}

	var body reqBody
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err)
		return
	}

	var order models.Order
	if err := oc.DB.Preload("OrderItems").First(&order, id).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, err)
		return
	}

	// Update status
	if body.Status != "" {
		order.Status = body.Status
	}

	// Tambah item
	if len(body.Items) > 0 {
		var total = order.TotalAmount
		for _, it := range body.Items {
			var menu models.Menu
			if err := oc.DB.First(&menu, it.MenuID).Error; err == nil {
				subTotal := float64(it.Quantity) * menu.Price
				total += subTotal

				orderItem := models.OrderItem{
					OrderID:   order.ID,
					MenuID:    menu.ID,
					Quantity:  it.Quantity,
					Price:     menu.Price,
					Status:    "pending",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				oc.DB.Create(&orderItem)
			}
		}
		order.TotalAmount = total
	}

	if err := oc.DB.Save(&order).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}

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
		// semua item => ready => set orders.status='ready'
		var order models.Order
		if err := oc.DB.First(&order, item.OrderID).Error; err == nil {
			order.Status = "ready"
			now := time.Now()
			order.FinishCookingTime = &now // jika mau catat finish masak
			oc.DB.Save(&order)

			// broadcast ke KDS?
			kds.BroadcastOrderUpdate(order)
		}
	}

	utils.RespondJSON(c, http.StatusOK, "Item finished, set ready", item)
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
