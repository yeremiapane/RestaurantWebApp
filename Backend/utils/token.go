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
	if IsTokenBlacklisted(tokenString) {
		return nil, errors.New("token telah di-blacklist")
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("token tidak valid")
}
