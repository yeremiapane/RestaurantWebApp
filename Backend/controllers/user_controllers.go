package controllers

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yeremiapane/restaurant-app/models"
	"github.com/yeremiapane/restaurant-app/utils"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserController struct {
	DB *gorm.DB
}

func NewUserController(db *gorm.DB) *UserController {
	return &UserController{DB: db}
}

// Register user baru
func (uc *UserController) Register(c *gin.Context) {
	type request struct {
		Name     string `json:"name" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
		Role     string `json:"role" binding:"required"` // admin, staff, chef, cleaner
	}
	var req request
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err)
		return
	}

	// Hash password
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	user := models.User{
		Name:     req.Name,
		Email:    req.Email,
		Password: string(hashed),
		Role:     req.Role,
	}

	if err := uc.DB.Create(&user).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	utils.InfoLogger.Printf("New user registered: %s (role=%s)", user.Email, user.Role)

	utils.RespondJSON(c, http.StatusCreated, "User registered", gin.H{
		"user_id": user.ID,
	})
}

// Login user -> return JWT
func (uc *UserController) Login(c *gin.Context) {
	fmt.Println("InfoLogger is nil:", utils.InfoLogger == nil)
	fmt.Println("ErrorLogger is nil:", utils.ErrorLogger == nil)

	var input struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		utils.InfoLogger.Errorf("Login: invalid input: %v", err)
		utils.RespondError(c, http.StatusBadRequest, err)
		return
	}

	if utils.InfoLogger == nil {
		log.Println("WARNING: Logger is nil! Initializing logger.")
		utils.InitLogger()
	}

	var user models.User
	if err := uc.DB.Where("email = ?", input.Email).First(&user).Error; err != nil {
		utils.InfoLogger.Errorf("Login failed: User not found: %s", input.Email)
		utils.RespondError(c, http.StatusUnauthorized, errors.New("Invalid credentials"))
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		utils.InfoLogger.Errorf("Login failed: Password mismatch for user %s", input.Email)
		utils.RespondError(c, http.StatusUnauthorized, errors.New("Invalid credentials"))
		return
	}

	token, err := utils.GenerateToken(user.ID, user.Role)
	if err != nil {
		utils.InfoLogger.Errorf("Token generation failed for user %s: %v", input.Email, err)
		utils.RespondError(c, http.StatusInternalServerError, errors.New("Failed to generate token"))
		return
	}

	utils.InfoLogger.Infof("User login success: %s (role=%s)", user.Email, user.Role)

	utils.RespondJSON(c, http.StatusOK, "Login success", gin.H{
		"token": token,
	})
}

// GetProfile -> memeriksa user dari JWT
func (uc *UserController) GetProfile(c *gin.Context) {
	// Data userID & role disimpan di context oleh AuthMiddleware
	userIDInterface, _ := c.Get("userID")
	userID := userIDInterface.(uint)

	var user models.User
	if err := uc.DB.First(&user, userID).Error; err != nil {
		utils.RespondError(c, http.StatusNotFound, err)
		return
	}

	utils.RespondJSON(c, http.StatusOK, "Profile data", gin.H{
		"id":    user.ID,
		"name":  user.Name,
		"email": user.Email,
		"role":  user.Role,
	})
}

// GetAllUsers -> contoh endpoint khusus Admin
func (uc *UserController) GetAllUsers(c *gin.Context) {
	// cek role
	roleInterface, _ := c.Get("role")
	if roleInterface != "admin" {
		utils.RespondError(c, http.StatusForbidden,
			gin.Error{Err: ErrNoPermission, Type: gin.ErrorTypePublic})
		return
	}

	var users []models.User
	if err := uc.DB.Find(&users).Error; err != nil {
		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	utils.RespondJSON(c, http.StatusOK, "All users", users)
}

// ErrNoPermission adalah contoh error custom
var ErrNoPermission = &CustomError{"You do not have permission"}

type CustomError struct {
	Message string
}

func (e *CustomError) Error() string {
	return e.Message
}
