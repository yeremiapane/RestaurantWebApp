package utils

import (
	"errors"
	"os"
	"sync"
	"time"

	"github.com/dgrijalva/jwt-go"
)

type Claims struct {
	UserID uint   `json:"user_id"`
	Role   string `json:"role"`
	jwt.StandardClaims
}

var (
	blacklistedTokens = make(map[string]time.Time)
	blacklistMutex    sync.RWMutex
)

func BlacklistToken(token string) {
	blacklistMutex.Lock()
	defer blacklistMutex.Unlock()
	blacklistedTokens[token] = time.Now().Add(24 * time.Hour) // Simpan token di blacklist selama 24 jam
}

func IsTokenBlacklisted(token string) bool {
	blacklistMutex.RLock()
	defer blacklistMutex.RUnlock()

	if expiry, exists := blacklistedTokens[token]; exists {
		if time.Now().Before(expiry) {
			return true
		}
		// Hapus token kadaluarsa dari blacklist
		delete(blacklistedTokens, token)
	}
	return false
}

// Bersihkan token kadaluarsa secara periodik
func cleanupBlacklist() {
	for {
		time.Sleep(1 * time.Hour)
		blacklistMutex.Lock()
		now := time.Now()
		for token, expiry := range blacklistedTokens {
			if now.After(expiry) {
				delete(blacklistedTokens, token)
			}
		}
		blacklistMutex.Unlock()
	}
}

func ValidateToken(tokenString string) (*Claims, error) {
	InfoLogger.Printf("Validating token: %s", tokenString)

	if IsTokenBlacklisted(tokenString) {
		ErrorLogger.Printf("Token is blacklisted")
		return nil, errors.New("token telah di-blacklist")
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})

	if err != nil {
		ErrorLogger.Printf("Token parsing failed: %v", err)
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		InfoLogger.Printf("Token is valid for user %d with role %s", claims.UserID, claims.Role)
		return claims, nil
	}

	ErrorLogger.Printf("Token is invalid")
	return nil, errors.New("token tidak valid")
}

// GenerateToken membuat token JWT baru
func GenerateToken(userID uint, role string) (string, error) {
	InfoLogger.Printf("Generating token for user %d with role %s", userID, role)

	// Set waktu kadaluarsa token (24 jam)
	expirationTime := time.Now().Add(24 * time.Hour)

	// Buat claims
	claims := &Claims{
		UserID: userID,
		Role:   role,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
			IssuedAt:  time.Now().Unix(),
		},
	}

	// Buat token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Ambil secret key dari environment variable
	secretKey := os.Getenv("JWT_SECRET")
	if secretKey == "" {
		ErrorLogger.Printf("JWT_SECRET environment variable is not set")
		return "", errors.New("JWT secret key not configured")
	}

	// Sign token
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		ErrorLogger.Printf("Failed to generate token: %v", err)
		return "", err
	}

	InfoLogger.Printf("Token generated successfully for user %d", userID)
	return tokenString, nil
}
