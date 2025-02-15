package controllers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yeremiapane/restaurant-app/models"
	"github.com/yeremiapane/restaurant-app/utils"
	"gorm.io/gorm"
)

type OrderController struct {
	DB *gorm.DB
}

func NewOrderController(db *gorm.DB) *OrderController {
	return &OrderController{DB: db}
}

// GetAllOrders -> dapat mencantumkan order beserta order_items
func (oc *OrderController) GetAllOrders(c *gin.Context) {
	var orders []models.Order
	if err := oc.DB.Preload("OrderItems").Find(&orders).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	utils.RespondJSON(c, http.StatusOK, "List of orders", orders)
}

// CreateOrder -> contoh membuat order status "draft"
func (oc *OrderController) CreateOrder(c *gin.Context) {
	type itemReq struct {
		MenuID   uint `json:"menu_id" binding:"required"`
		Quantity int  `json:"quantity" binding:"required"`
	}
	type reqBody struct {
		CustomerID uint      `json:"customer_id" binding:"required"`
		Items      []itemReq `json:"items" binding:"required"`
	}

	var body reqBody
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err)
		return
	}

	// Buat order
	order := models.Order{
		CustomerID:  body.CustomerID,
		Status:      "draft",
		TotalAmount: 0,
	}

	if err := oc.DB.Create(&order).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	var total float64
	for _, it := range body.Items {
		var menu models.Menu
		if err := oc.DB.First(&menu, it.MenuID).Error; err != nil {
			continue // skip jika tidak ditemukan
		}
		subTotal := float64(it.Quantity) * menu.Price
		total += subTotal

		orderItem := models.OrderItem{
			OrderID:  order.ID,
			MenuID:   menu.ID,
			Quantity: it.Quantity,
			Price:    menu.Price,
		}
		oc.DB.Create(&orderItem)
	}

	// Update totalAmount
	order.TotalAmount = total
	if err := oc.DB.Save(&order).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	utils.RespondJSON(c, http.StatusCreated, "Order created (draft)", order)
}

// GetOrderByID
func (oc *OrderController) GetOrderByID(c *gin.Context) {
	idStr := c.Param("order_id")
	id, _ := strconv.Atoi(idStr)

	var order models.Order
	if err := oc.DB.Preload("OrderItems").First(&order, id).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, err)
		return
	}

	utils.RespondJSON(c, http.StatusOK, "Order detail", order)
}

// UpdateOrder -> contoh menambahkan item baru / mengubah status
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

	// Update status jika diisi
	if body.Status != "" {
		order.Status = body.Status
	}

	// Jika ada item baru, tambahkan
	if len(body.Items) > 0 {
		var total = order.TotalAmount
		for _, it := range body.Items {
			var menu models.Menu
			if err := oc.DB.First(&menu, it.MenuID).Error; err != nil {
				continue
			}
			subTotal := float64(it.Quantity) * menu.Price
			total += subTotal

			orderItem := models.OrderItem{
				OrderID:  order.ID,
				MenuID:   menu.ID,
				Quantity: it.Quantity,
				Price:    menu.Price,
			}
			oc.DB.Create(&orderItem)
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
