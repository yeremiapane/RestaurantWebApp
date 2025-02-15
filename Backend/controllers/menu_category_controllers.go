package controllers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yeremiapane/restaurant-app/models"
	"github.com/yeremiapane/restaurant-app/utils"
	"gorm.io/gorm"
)

type MenuCategoryController struct {
	DB *gorm.DB
}

func NewMenuCategoryController(db *gorm.DB) *MenuCategoryController {
	return &MenuCategoryController{DB: db}
}

// GetAllCategories
func (mcc *MenuCategoryController) GetAllCategories(c *gin.Context) {
	var categories []models.MenuCategory
	if err := mcc.DB.Find(&categories).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}
	utils.RespondJSON(c, http.StatusOK, "All menu categories", categories)
}

// CreateCategory
func (mcc *MenuCategoryController) CreateCategory(c *gin.Context) {
	var body struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err)
		return
	}

	category := models.MenuCategory{
		Name: body.Name,
	}
	if err := mcc.DB.Create(&category).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	utils.RespondJSON(c, http.StatusCreated, "Category created", category)
}

// GetCategoryByID
func (mcc *MenuCategoryController) GetCategoryByID(c *gin.Context) {
	idStr := c.Param("cat_id")
	id, _ := strconv.Atoi(idStr)

	var category models.MenuCategory
	if err := mcc.DB.First(&category, id).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, err)
		return
	}

	utils.RespondJSON(c, http.StatusOK, "Category detail", category)
}

// UpdateCategory
func (mcc *MenuCategoryController) UpdateCategory(c *gin.Context) {
	idStr := c.Param("cat_id")
	id, _ := strconv.Atoi(idStr)

	var body struct {
		Name string `json:"name"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err)
		return
	}

	var category models.MenuCategory
	if err := mcc.DB.First(&category, id).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, err)
		return
	}

	if body.Name != "" {
		category.Name = body.Name
	}

	if err := mcc.DB.Save(&category).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	utils.RespondJSON(c, http.StatusOK, "Category updated", category)
}

// DeleteCategory
func (mcc *MenuCategoryController) DeleteCategory(c *gin.Context) {
	idStr := c.Param("cat_id")
	id, _ := strconv.Atoi(idStr)

	if err := mcc.DB.Delete(&models.MenuCategory{}, id).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}
	utils.RespondJSON(c, http.StatusOK, "Category deleted", gin.H{"category_id": id})
}
