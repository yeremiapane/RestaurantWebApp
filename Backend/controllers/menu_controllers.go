package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

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

	result := mc.DB.Preload("Category").Find(&menus)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  false,
			"message": result.Error.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  true,
		"message": "List of menus",
		"data":    menus,
	})
}

// CreateMenu
func (mc *MenuController) CreateMenu(c *gin.Context) {
	// Batasi ukuran upload ke 10MB
	c.Request.ParseMultipartForm(10 << 20)

	// Ambil data form
	categoryID, err := strconv.ParseUint(c.PostForm("category_id"), 10, 32)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, errors.New("invalid category_id"))
		return
	}

	price, err := strconv.ParseFloat(c.PostForm("price"), 64)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, errors.New("invalid price"))
		return
	}

	stock, err := strconv.Atoi(c.PostForm("stock"))
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, errors.New("invalid stock"))
		return
	}

	// Ambil file gambar
	form, err := c.MultipartForm()
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, errors.New("error processing form"))
		return
	}

	files := form.File["images"]
	if len(files) == 0 {
		utils.RespondError(c, http.StatusBadRequest, errors.New("at least one image is required"))
		return
	}

	// Buat direktori untuk menyimpan gambar jika belum ada
	uploadDir := "public/uploads/menu_images"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		utils.RespondError(c, http.StatusInternalServerError, errors.New("error creating upload directory"))
		return
	}

	// Simpan gambar dan kumpulkan URL-nya
	var imageUrls []string
	baseURL := "http://localhost:8080" // Sesuaikan dengan domain aplikasi

	for _, file := range files {
		// Generate nama file unik
		filename := fmt.Sprintf("%d-%s", time.Now().UnixNano(), file.Filename)
		filepath := fmt.Sprintf("%s/%s", uploadDir, filename)

		// Simpan file
		if err := c.SaveUploadedFile(file, filepath); err != nil {
			// Hapus file yang sudah diupload jika ada error
			for _, url := range imageUrls {
				// Hapus file berdasarkan URL
				localPath := strings.Replace(url, baseURL+"/uploads/menu_images/", uploadDir+"/", 1)
				os.Remove(localPath)
			}
			utils.RespondError(c, http.StatusInternalServerError, errors.New("error saving image"))
			return
		}

		// Simpan URL publik ke database
		imageUrl := fmt.Sprintf("%s/uploads/menu_images/%s", baseURL, filename)
		imageUrls = append(imageUrls, imageUrl)
	}

	// Buat menu baru
	menu := models.Menu{
		CategoryID:  uint(categoryID),
		Name:        c.PostForm("name"),
		Price:       price,
		Stock:       stock,
		Description: c.PostForm("description"),
	}

	if err := menu.SetImageUrls(imageUrls); err != nil {
		// Hapus gambar yang sudah diupload jika gagal
		for _, url := range imageUrls {
			localPath := strings.Replace(url, baseURL+"/uploads/menu_images/", uploadDir+"/", 1)
			os.Remove(localPath)
		}
		utils.RespondError(c, http.StatusInternalServerError, errors.New("error processing image urls"))
		return
	}

	if err := mc.DB.Create(&menu).Error; err != nil {
		// Hapus gambar yang sudah diupload jika gagal membuat menu
		for _, url := range imageUrls {
			// Hapus file berdasarkan URL
			localPath := strings.Replace(url, baseURL+"/uploads/menu_images/", uploadDir+"/", 1)
			os.Remove(localPath)
		}
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
// Endpoint: GET /menus/by-category?category=<id kategori>
func (mc *MenuController) GetMenuByCategory(c *gin.Context) {
	categoryIDStr := c.Query("category")
	if categoryIDStr == "" {
		utils.RespondError(c, http.StatusBadRequest, errors.New("query parameter 'category' is required"))
		return
	}

	categoryID, err := strconv.Atoi(categoryIDStr)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, errors.New("invalid category ID"))
		return
	}

	var menus []models.Menu
	if err := mc.DB.Preload("Category").
		Where("category_id = ?", categoryID).
		Find(&menus).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	utils.RespondJSON(c, http.StatusOK, fmt.Sprintf("List of menus for category ID: %d", categoryID), menus)
}

// UpdateMenu dengan dukungan update gambar
func (mc *MenuController) UpdateMenu(c *gin.Context) {
	idStr := c.Param("menu_id")
	id, _ := strconv.Atoi(idStr)

	// Parse multipart form terlebih dahulu
	if err := c.Request.ParseMultipartForm(10 << 20); err != nil {
		utils.RespondError(c, http.StatusBadRequest, errors.New("error parsing form data"))
		return
	}

	// Ambil data dari form
	name := c.PostForm("name")
	categoryIDStr := c.PostForm("category_id")
	priceStr := c.PostForm("price")
	stockStr := c.PostForm("stock")
	description := c.PostForm("description")
	removedImagesStr := c.PostForm("removed_images") // String JSON dari URL gambar yang akan dihapus

	// Validasi dan konversi data
	categoryID, err := strconv.ParseUint(categoryIDStr, 10, 32)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, errors.New("invalid category_id"))
		return
	}

	// Parse price (hapus karakter non-numerik jika ada)
	priceStr = strings.ReplaceAll(priceStr, ".", "")
	priceStr = strings.ReplaceAll(priceStr, ",", "")
	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, errors.New("invalid price format"))
		return
	}

	// Parse stock
	stock, err := strconv.Atoi(stockStr)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, errors.New("invalid stock format"))
		return
	}

	// Ambil menu yang akan diupdate
	var menu models.Menu
	if err := mc.DB.First(&menu, id).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, errors.New("menu not found"))
		return
	}

	// Update data menu
	menu.Name = name
	menu.CategoryID = uint(categoryID)
	menu.Price = price
	menu.Stock = stock
	menu.Description = description

	// Proses penghapusan gambar yang dipilih
	var removedImages []string
	if removedImagesStr != "" {
		if err := json.Unmarshal([]byte(removedImagesStr), &removedImages); err != nil {
			utils.RespondError(c, http.StatusBadRequest, errors.New("invalid removed_images format"))
			return
		}
	}

	// Ambil URL gambar yang ada
	currentImages := menu.GetImageUrls()
	var newImageList []string

	// Filter gambar yang tidak dihapus
	for _, img := range currentImages {
		isRemoved := false
		for _, removedImg := range removedImages {
			if img == removedImg {
				isRemoved = true
				// Hapus file gambar yang dihapus
				localPath := strings.Replace(img, "http://localhost:8080/uploads/menu_images/", "public/uploads/menu_images/", 1)
				os.Remove(localPath)
				break
			}
		}
		if !isRemoved {
			newImageList = append(newImageList, img)
		}
	}

	// Handle file gambar baru jika ada
	form, _ := c.MultipartForm()
	if form != nil && form.File != nil {
		if files := form.File["images"]; len(files) > 0 {
			baseURL := "http://localhost:8080"
			uploadDir := "public/uploads/menu_images"

			// Pastikan direktori upload ada
			if err := os.MkdirAll(uploadDir, 0755); err != nil {
				utils.RespondError(c, http.StatusInternalServerError, errors.New("error creating upload directory"))
				return
			}

			// Upload gambar baru
			for _, file := range files {
				filename := fmt.Sprintf("%d-%s", time.Now().UnixNano(), file.Filename)
				filepath := fmt.Sprintf("%s/%s", uploadDir, filename)

				if err := c.SaveUploadedFile(file, filepath); err != nil {
					utils.RespondError(c, http.StatusInternalServerError, errors.New("error saving image"))
					return
				}

				imageUrl := fmt.Sprintf("%s/uploads/menu_images/%s", baseURL, filename)
				newImageList = append(newImageList, imageUrl)
			}
		}
	}

	// Update image URLs di menu dengan gabungan gambar lama (yang tidak dihapus) dan gambar baru
	if err := menu.SetImageUrls(newImageList); err != nil {
		utils.RespondError(c, http.StatusInternalServerError, errors.New("error processing image urls"))
		return
	}

	// Simpan perubahan ke database
	if err := mc.DB.Save(&menu).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	utils.RespondJSON(c, http.StatusOK, "Menu updated successfully", menu)
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
