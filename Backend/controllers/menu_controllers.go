package controllers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yeremiapane/restaurant-app/models"
	"github.com/yeremiapane/restaurant-app/utils"
	"gorm.io/gorm"
)

type MenuController struct {
	DB *gorm.DB
}

func NewMenuController(db *gorm.DB) *MenuController {
	return &MenuController{DB: db}
}

// GetAllMenus
func (mc *MenuController) GetAllMenus(c *gin.Context) {
	var menus []models.Menu
	// Preload Category agar data category langsung ikut
	if err := mc.DB.Preload("Category").Find(&menus).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}
	utils.RespondJSON(c, http.StatusOK, "List of menus", menus)
}

// CreateMenu
func (mc *MenuController) CreateMenu(c *gin.Context) {
	type reqBody struct {
		CategoryID  uint    `json:"category_id" binding:"required"`
		Name        string  `json:"name" binding:"required"`
		Price       float64 `json:"price" binding:"required"`
		Stock       int     `json:"stock" binding:"required"`
		Description string  `json:"description"`
		ImageURL    *string `json:"image_url"`
	}

	var body reqBody
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err)
		return
	}

	menu := models.Menu{
		CategoryID:  body.CategoryID,
		Name:        body.Name,
		Price:       body.Price,
		Stock:       body.Stock,
		Description: body.Description,
		ImageUrl:    body.ImageURL,
	}

	if err := mc.DB.Create(&menu).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	utils.RespondJSON(c, http.StatusCreated, "Menu created", menu)
}

// GetMenuByID
func (mc *MenuController) GetMenuByID(c *gin.Context) {
	idStr := c.Param("menu_id")
	id, _ := strconv.Atoi(idStr)

	var menu models.Menu
	if err := mc.DB.Preload("Category").First(&menu, id).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, err)
		return
	}

	utils.RespondJSON(c, http.StatusOK, "Menu detail", menu)
}


// GetMenuByCategory mengembalikan daftar menu berdasarkan kategori
// Endpoint: GET /menus/by-category?category=<nama kategori>
func (mc *MenuController) GetMenuByCategory(c *gin.Context) {
	categoryName := c.Query("category")
	if categoryName == "" {
		utils.RespondError(c, http.StatusBadRequest, errors.New("query parameter 'category' is required"))
		return
	}

	var menus []models.Menu
	// Gunakan join pada tabel menu_categories untuk mencari berdasarkan nama kategori (case-insensitive)
	if err := mc.DB.Preload("Category").
		Joins("JOIN menu_categories ON menu_categories.id = menus.category_id").
		Where("LOWER(menu_categories.name) = ?", strings.ToLower(categoryName)).
		Find(&menus).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	utils.RespondJSON(c, http.StatusOK, "List of menus for category: "+categoryName, menus)
}


// UpdateMenu
func (mc *MenuController) UpdateMenu(c *gin.Context) {
	idStr := c.Param("menu_id")
	id, _ := strconv.Atoi(idStr)

	var body struct {
		CategoryID  *uint    `json:"category_id"`
		Name        *string  `json:"name"`
		Price       *float64 `json:"price"`
		Stock       *int     `json:"stock"`
		Description *string  `json:"description"`
		ImageURL    *string  `json:"image_url"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err)
		return
	}

	var menu models.Menu
	if err := mc.DB.First(&menu, id).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, err)
		return
	}

	if body.CategoryID != nil {
		menu.CategoryID = *body.CategoryID
	}
	if body.Name != nil {
		menu.Name = *body.Name
	}
	if body.Price != nil {
		menu.Price = *body.Price
	}
	if body.Stock != nil {
		menu.Stock = *body.Stock
	}
	if body.Description != nil {
		menu.Description = *body.Description
	}
	if body.ImageURL != nil {
		menu.ImageUrl = body.ImageURL
	}

	if err := mc.DB.Save(&menu).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	utils.RespondJSON(c, http.StatusOK, "Menu updated", menu)
}

// DeleteMenu
func (mc *MenuController) DeleteMenu(c *gin.Context) {
	idStr := c.Param("menu_id")
	id, _ := strconv.Atoi(idStr)

	if err := mc.DB.Delete(&models.Menu{}, id).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}
	utils.RespondJSON(c, http.StatusOK, "Menu deleted", gin.H{"menu_id": id})
}
