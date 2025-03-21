package utils

import (
	"errors"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var JWTSecret []byte

func init() {
	// Tambahkan logging untuk debug
	log.Printf("Current working directory: %s", getCurrentDirectory())
	log.Printf("Loading JWT_SECRET from environment...")

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		// Gunakan default secret untuk development
		log.Printf("Warning: JWT_SECRET not found in environment, using default secret")
		secret = "TestSecretKeyAUTH1945" // Sama dengan yang di .env
	}

	JWTSecret = []byte(secret)
	log.Printf("JWT_SECRET loaded successfully")
}

// Helper function untuk mendapatkan current directory
func getCurrentDirectory() string {
	dir, err := os.Getwd()
	if err != nil {
		return "unknown"
	}
	return dir
}

type CustomClaims struct {
	UserID uint   `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func GenerateToken(userID uint, role string) (string, error) {
	// Log untuk debugging
	log.Printf("Generating token for userID: %d, role: %s", userID, role)

	claims := &CustomClaims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "RestaurantWebApp",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(JWTSecret)

	if err != nil {
		log.Printf("Error generating token: %v", err)
		return "", err
	}

	log.Printf("Token generated successfully")
	return tokenString, nil
}

func ParseToken(tokenString string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return JWTSecret, nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("invalid or expired token")
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}
