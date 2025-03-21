package utils

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
)

type JSONResponse struct {
	Status  bool        `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func RespondJSON(c *gin.Context, code int, message string, data interface{}) {
	c.JSON(code, JSONResponse{
		Status:  code >= 200 && code < 300,
		Message: message,
		Data:    data,
	})
}

func RespondError(c *gin.Context, code int, err error) {
	c.JSON(code, JSONResponse{
		Status:  false,
		Message: err.Error(),
		Data:    nil,
	})
}

// FormatCurrency memformat angka ke format mata uang Rupiah
func FormatCurrency(amount float64) string {
	// Format dengan pemisah ribuan dan 2 desimal
	formatted := fmt.Sprintf("%.2f", amount)

	// Pisahkan bagian desimal
	parts := strings.Split(formatted, ".")
	integerPart := parts[0]
	decimalPart := parts[1]

	// Tambahkan pemisah ribuan
	var result []string
	for i := len(integerPart); i > 0; i -= 3 {
		start := i - 3
		if start < 0 {
			start = 0
		}
		result = append([]string{integerPart[start:i]}, result...)
	}

	// Gabungkan kembali dengan pemisah ribuan dan desimal
	return strings.Join(result, ".") + "," + decimalPart
}
