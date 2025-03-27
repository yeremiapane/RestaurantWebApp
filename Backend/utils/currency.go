package utils

import (
	"fmt"
	"math"
)

// FormatCurrencyIDR formats a float64 value as a currency string in Indonesian Rupiah format
// Example: 15000.50 -> "Rp 15.000,50"
func FormatCurrencyIDR(amount float64) string {
	// Pisahkan bagian integer dan desimal
	integer := math.Floor(amount)
	decimal := amount - integer

	// Format bagian integer dengan separator ribuan
	integerStr := ""
	intTemp := integer

	// Jika angka 0, langsung kembalikan "0"
	if intTemp == 0 {
		integerStr = "0"
	}

	// Memformat bagian integer dengan pemisah ribuan
	for intTemp > 0 {
		remainder := int(math.Mod(intTemp, 1000))

		if intTemp >= 1000 {
			// Tambahkan leading zeros jika perlu
			if remainder < 10 {
				integerStr = fmt.Sprintf(".00%d%s", remainder, integerStr)
			} else if remainder < 100 {
				integerStr = fmt.Sprintf(".0%d%s", remainder, integerStr)
			} else {
				integerStr = fmt.Sprintf(".%d%s", remainder, integerStr)
			}
		} else {
			integerStr = fmt.Sprintf("%d%s", remainder, integerStr)
		}

		intTemp = math.Floor(intTemp / 1000)
	}

	// Format dengan 2 digit desimal
	if decimal > 0 {
		// Bulatkan ke 2 digit desimal
		decimal = math.Round(decimal*100) / 100

		// Format bagian desimal
		decimalStr := fmt.Sprintf("%02.0f", decimal*100)
		return fmt.Sprintf("Rp %s,%s", integerStr, decimalStr)
	}

	return fmt.Sprintf("Rp %s", integerStr)
}
