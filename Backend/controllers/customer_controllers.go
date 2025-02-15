package controllers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yeremiapane/restaurant-app/models"
	"github.com/yeremiapane/restaurant-app/utils"
	"gorm.io/gorm"
)

type CustomerController struct {
	DB *gorm.DB
}

func NewCustomerController(db *gorm.DB) *CustomerController {
	return &CustomerController{DB: db}
}

// GetAllCustomers -> Mendapatkan semua customer (aktif/finished)
func (cc *CustomerController) GetAllCustomers(c *gin.Context) {
	var customers []models.Customer
	if err := cc.DB.Preload("Table").Find(&customers).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	utils.RespondJSON(c, http.StatusOK, "List of customers", customers)
}

// CreateCustomer -> Membuat record Customer baru (misal saat scan QR)
func (cc *CustomerController) CreateCustomer(c *gin.Context) {
	type reqBody struct {
		TableID    uint    `json:"table_id" binding:"required"`
		SessionKey *string `json:"session_key"`
	}

	var req reqBody
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err)
		return
	}

	// Cek apakah meja masih available
	var table models.Table
	if err := cc.DB.First(&table, req.TableID).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, err)
		return
	}

	if table.Status != "available" {
		utils.RespondError(c, http.StatusConflict,
			gin.Error{Err: ErrTableOccupied, Type: gin.ErrorTypePublic})
		return
	}

	// Buat customer
	customer := models.Customer{
		TableID:    req.TableID,
		SessionKey: req.SessionKey,
		Status:     "active",
	}

	if err := cc.DB.Create(&customer).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	// Ubah status table => occupied
	table.Status = "occupied"
	if err := cc.DB.Save(&table).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	utils.InfoLogger.Printf("New customer created (ID=%d) at TableID=%d", customer.ID, req.TableID)

	utils.RespondJSON(c, http.StatusCreated, "Customer created", customer)
}

// GetCustomerByID -> Menampilkan detail 1 customer
func (cc *CustomerController) GetCustomerByID(c *gin.Context) {
	idStr := c.Param("customer_id")
	id, _ := strconv.Atoi(idStr)

	var customer models.Customer
	if err := cc.DB.Preload("Table").First(&customer, id).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, err)
		return
	}

	utils.RespondJSON(c, http.StatusOK, "Customer detail", customer)
}

// UpdateCustomer -> Contoh update status 'finished' jika customer meninggalkan meja
func (cc *CustomerController) UpdateCustomer(c *gin.Context) {
	idStr := c.Param("customer_id")
	id, _ := strconv.Atoi(idStr)

	type reqBody struct {
		Status string `json:"status"`
	}

	var req reqBody
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err)
		return
	}

	var customer models.Customer
	if err := cc.DB.First(&customer, id).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, err)
		return
	}

	customer.Status = req.Status
	if err := cc.DB.Save(&customer).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	// Jika customer selesai => ubah meja => dirty
	if req.Status == "finished" {
		var table models.Table
		if err := cc.DB.First(&table, customer.TableID).Error; err == nil {
			table.Status = "dirty"
			cc.DB.Save(&table)
		}
	}

	utils.RespondJSON(c, http.StatusOK, "Customer updated", customer)
}

// DeleteCustomer -> Menghapus record customer (opsional)
func (cc *CustomerController) DeleteCustomer(c *gin.Context) {
	idStr := c.Param("customer_id")
	id, _ := strconv.Atoi(idStr)

	if err := cc.DB.Delete(&models.Customer{}, id).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	utils.RespondJSON(c, http.StatusOK, "Customer deleted", gin.H{"customer_id": id})
}

var ErrTableOccupied = &CustomError{"Table is not available"}