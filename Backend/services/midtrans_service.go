package services

import (
	"bytes"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/yeremiapane/restaurant-app/models"
)

// MidtransConfig holds Midtrans configuration
type MidtransConfig struct {
	ServerKey     string
	ClientKey     string
	IsProduction  bool
	MerchantID    string
	MerchantName  string
	MerchantEmail string
	MerchantPhone string
	WebhookURL    string
}

// MidtransService handles Midtrans API interactions
type MidtransService struct {
	config     *MidtransConfig
	httpClient *http.Client
}

var (
	midtransService *MidtransService
	midtransOnce    sync.Once
)

// GetMidtransService returns singleton instance of MidtransService
func GetMidtransService() *MidtransService {
	midtransOnce.Do(func() {
		// Get config dari environment variables
		serverKey := os.Getenv("MIDTRANS_SERVER_KEY")
		clientKey := os.Getenv("MIDTRANS_CLIENT_KEY")
		isProduction := os.Getenv("MIDTRANS_ENV") == "production"
		merchantID := os.Getenv("MIDTRANS_MERCHANT_ID")
		merchantName := os.Getenv("MIDTRANS_MERCHANT_NAME")
		merchantEmail := os.Getenv("MIDTRANS_MERCHANT_EMAIL")
		merchantPhone := os.Getenv("MIDTRANS_MERCHANT_PHONE")
		webhookURL := os.Getenv("MIDTRANS_WEBHOOK_URL")

		// Log konfigurasi untuk debugging
		fmt.Printf("Initializing Midtrans with config:\n")
		fmt.Printf("ServerKey: %s\n", serverKey)
		fmt.Printf("ClientKey: %s\n", clientKey)
		fmt.Printf("IsProduction: %v\n", isProduction)
		fmt.Printf("MerchantID: %s\n", merchantID)
		fmt.Printf("MerchantName: %s\n", merchantName)
		fmt.Printf("MerchantEmail: %s\n", merchantEmail)
		fmt.Printf("MerchantPhone: %s\n", merchantPhone)
		fmt.Printf("WebhookURL: %s\n", webhookURL)

		// Periksa apakah server key kosong, gunakan default nilai test jika kosong
		if serverKey == "" {
			fmt.Println("WARNING: MIDTRANS_SERVER_KEY is empty, using default sandbox key")
			serverKey = "SB-Mid-server-QTZL1aqTtYPBAv2VvQhP1ymF"
		}

		if clientKey == "" {
			fmt.Println("WARNING: MIDTRANS_CLIENT_KEY is empty, using default sandbox key")
			clientKey = "SB-Mid-client-N4d3OcqzGEcQ3Wxh"
		}

		if merchantID == "" {
			fmt.Println("WARNING: MIDTRANS_MERCHANT_ID is empty, using default value")
			merchantID = "G117629268"
		}

		// Atur nilai default untuk field lain jika kosong
		if merchantName == "" {
			merchantName = "Restaurant App"
		}

		if merchantEmail == "" {
			merchantEmail = "restaurant@example.com"
		}

		if merchantPhone == "" {
			merchantPhone = "08123456789"
		}

		if webhookURL == "" {
			webhookURL = "https://example.com/callback"
		}

		midtransService = &MidtransService{
			config: &MidtransConfig{
				ServerKey:     serverKey,
				ClientKey:     clientKey,
				IsProduction:  isProduction,
				MerchantID:    merchantID,
				MerchantName:  merchantName,
				MerchantEmail: merchantEmail,
				MerchantPhone: merchantPhone,
				WebhookURL:    webhookURL,
			},
			httpClient: &http.Client{
				Timeout: 30 * time.Second,
			},
		}
	})
	return midtransService
}

// ValidateConfig validates Midtrans configuration
func (ms *MidtransService) ValidateConfig() error {
	if ms.config.ServerKey == "" {
		return fmt.Errorf("MIDTRANS_SERVER_KEY is not set")
	}
	if ms.config.ClientKey == "" {
		return fmt.Errorf("MIDTRANS_CLIENT_KEY is not set")
	}
	if ms.config.MerchantID == "" {
		return fmt.Errorf("MIDTRANS_MERCHANT_ID is not set")
	}
	if ms.config.MerchantName == "" {
		return fmt.Errorf("MIDTRANS_MERCHANT_NAME is not set")
	}
	if ms.config.MerchantEmail == "" {
		return fmt.Errorf("MIDTRANS_MERCHANT_EMAIL is not set")
	}
	if ms.config.MerchantPhone == "" {
		return fmt.Errorf("MIDTRANS_MERCHANT_PHONE is not set")
	}
	if ms.config.WebhookURL == "" {
		return fmt.Errorf("MIDTRANS_WEBHOOK_URL is not set")
	}
	return nil
}

// CreateTransaction creates a new transaction in Midtrans using Order model
func (ms *MidtransService) CreateTransaction(orderID string, amount float64, order models.Order) (*MidtransResponse, error) {
	baseURL := ms.getBaseURL()
	url := fmt.Sprintf("%s/v2/charge", baseURL)

	// Log konfigurasi untuk debugging
	fmt.Printf("Creating Midtrans transaction with config:\n")
	fmt.Printf("Server Key: %s\n", ms.config.ServerKey)
	fmt.Printf("Base URL: %s\n", baseURL)
	fmt.Printf("Is Production: %v\n", ms.config.IsProduction)

	// Get customer name and email from order
	customerName := order.GetCustomerName()
	customerEmail := order.GetCustomerEmail()

	// Cetak payload yang dikirim ke Midtrans
	payload := map[string]interface{}{
		"payment_type": "qris",
		"transaction_details": map[string]interface{}{
			"order_id":     orderID,
			"gross_amount": int64(amount),
		},
		"customer_details": map[string]interface{}{
			"first_name": customerName,
			"email":      customerEmail,
		},
		"item_details": []map[string]interface{}{
			{
				"id":       orderID,
				"price":    int64(amount),
				"quantity": 1,
				"name":     "Order Payment",
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	fmt.Printf("Sending payload to Midtrans: %s\n", string(jsonData))

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	// Buat basic auth dengan encoding base64 yang benar
	authString := "Basic " + base64.StdEncoding.EncodeToString([]byte(ms.config.ServerKey+":"))
	fmt.Printf("Using Authorization header: %s\n", authString)

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authString)

	// Log request yang akan dikirim
	fmt.Printf("Sending request to Midtrans API: %s %s\n", req.Method, req.URL.String())
	fmt.Printf("Headers: %v\n", req.Header)

	resp, err := ms.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	// Log response dari Midtrans
	fmt.Printf("Response from Midtrans (status %d):\n%s\n", resp.StatusCode, string(body))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Midtrans API error: %s", string(body))
	}

	var midtransResp MidtransResponse
	if err := json.Unmarshal(body, &midtransResp); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	fmt.Printf("Successfully created transaction with ID: %s\n", orderID)

	return &midtransResp, nil
}

// CreateTransactionWithCustomer creates a new transaction in Midtrans with customer details
func (ms *MidtransService) CreateTransactionWithCustomer(orderID string, amount float64, customerName string, customerEmail string) (*MidtransResponse, error) {
	baseURL := ms.getBaseURL()
	url := fmt.Sprintf("%s/v2/charge", baseURL)

	// Log konfigurasi untuk debugging
	fmt.Printf("Creating Midtrans transaction with config:\n")
	fmt.Printf("Server Key: %s\n", ms.config.ServerKey)
	fmt.Printf("Base URL: %s\n", baseURL)
	fmt.Printf("Is Production: %v\n", ms.config.IsProduction)

	// Cetak payload yang dikirim ke Midtrans
	payload := map[string]interface{}{
		"payment_type": "qris",
		"transaction_details": map[string]interface{}{
			"order_id":     orderID,
			"gross_amount": int64(amount),
		},
		"customer_details": map[string]interface{}{
			"first_name": customerName,
			"email":      customerEmail,
		},
		"item_details": []map[string]interface{}{
			{
				"id":       orderID,
				"price":    int64(amount),
				"quantity": 1,
				"name":     "Order Payment",
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	fmt.Printf("Sending payload to Midtrans: %s\n", string(jsonData))

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	// Buat basic auth dengan encoding base64 yang benar
	authString := "Basic " + base64.StdEncoding.EncodeToString([]byte(ms.config.ServerKey+":"))
	fmt.Printf("Using Authorization header: %s\n", authString)

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authString)

	// Log request yang akan dikirim
	fmt.Printf("Sending request to Midtrans API: %s %s\n", req.Method, req.URL.String())
	fmt.Printf("Headers: %v\n", req.Header)

	resp, err := ms.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	// Log response dari Midtrans
	fmt.Printf("Response from Midtrans (status %d):\n%s\n", resp.StatusCode, string(body))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Midtrans API error: %s", string(body))
	}

	var midtransResp MidtransResponse
	if err := json.Unmarshal(body, &midtransResp); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	fmt.Printf("Successfully created transaction with ID: %s\n", orderID)

	return &midtransResp, nil
}

// CheckTransactionStatus checks transaction status from Midtrans
func (ms *MidtransService) CheckTransactionStatus(orderID string) (string, error) {
	baseURL := ms.getBaseURL()
	url := fmt.Sprintf("%s/v2/%s/status", baseURL, orderID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	// Buat basic auth dengan encoding base64 yang benar
	authString := "Basic " + base64.StdEncoding.EncodeToString([]byte(ms.config.ServerKey+":"))
	fmt.Printf("Using Authorization header for status check: %s\n", authString)

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", authString)

	// Log request yang akan dikirim
	fmt.Printf("Sending request to check transaction status: %s %s\n", req.Method, req.URL.String())
	fmt.Printf("Headers: %v\n", req.Header)

	resp, err := ms.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %v", err)
	}

	// Log response dari Midtrans
	fmt.Printf("Response from Midtrans status check (status %d):\n%s\n", resp.StatusCode, string(body))

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Midtrans API error: %s", string(body))
	}

	var statusResp struct {
		TransactionStatus string `json:"transaction_status"`
	}
	if err := json.Unmarshal(body, &statusResp); err != nil {
		return "", fmt.Errorf("error unmarshaling response: %v", err)
	}

	return ms.mapTransactionStatus(statusResp.TransactionStatus), nil
}

// ValidateSignature validates Midtrans signature
func (ms *MidtransService) ValidateSignature(orderID, statusCode, grossAmount, signature string) bool {
	signatureString := fmt.Sprintf("%s%s%s%s", orderID, statusCode, grossAmount, ms.config.ServerKey)
	hash := sha512.New()
	hash.Write([]byte(signatureString))
	calculatedSignature := hex.EncodeToString(hash.Sum(nil))
	return calculatedSignature == signature
}

// mapTransactionStatus maps Midtrans transaction status to internal status
func (ms *MidtransService) mapTransactionStatus(status string) string {
	switch status {
	case "capture", "settlement":
		return "success"
	case "pending", "authorize":
		return "pending"
	case "deny", "cancel", "expire", "failure":
		return "failed"
	default:
		return "unknown"
	}
}

// getBaseURL returns the appropriate Midtrans API base URL
func (ms *MidtransService) getBaseURL() string {
	if ms.config.IsProduction {
		return "https://api.midtrans.com"
	}
	return "https://api.sandbox.midtrans.com"
}

// MidtransResponse represents Midtrans API response
type MidtransResponse struct {
	Token           string `json:"token"`
	RedirectURL     string `json:"redirect_url"`
	QRCode          string `json:"qr_code"`
	QRCodeURL       string `json:"qr_string"` // Alternatif field untuk QR QRIS
	Status          string `json:"status"`
	StatusCode      string `json:"status_code"`
	TransactionID   string `json:"transaction_id"`
	OrderID         string `json:"order_id"`
	GrossAmount     string `json:"gross_amount"`
	PaymentType     string `json:"payment_type"`
	TransactionTime string `json:"transaction_time"`
	Message         string `json:"message"`
	ExpiryTime      string `json:"expiry_time"` // Waktu kadaluarsa pembayaran
	Actions         []struct {
		Name   string `json:"name"`
		Method string `json:"method"`
		URL    string `json:"url"`
	} `json:"actions"`
}

// GenerateQRImageURL menghasilkan URL gambar QR code dari data QRIS
func (ms *MidtransService) GenerateQRImageURL(qrisData string) string {
	// Format yang sesuai dengan dokumentasi Midtrans
	// Jika qrisData adalah ID transaksi, gunakan format URL yang benar
	if len(qrisData) < 100 { // ID transaksi biasanya lebih pendek dari data QR code
		// Gunakan format sesuai dokumentasi: https://api.midtrans.com/v2/qris/{transaction_id}/qr-code
		if ms.config.IsProduction {
			return fmt.Sprintf("https://api.midtrans.com/v2/qris/%s/qr-code", qrisData)
		} else {
			return fmt.Sprintf("https://api.sandbox.midtrans.com/v2/qris/%s/qr-code", qrisData)
		}
	}

	// Jika qrisData adalah data QR yang panjang, gunakan format lama (fallback)
	encoded := base64.URLEncoding.EncodeToString([]byte(qrisData))
	if ms.config.IsProduction {
		return fmt.Sprintf("https://api.midtrans.com/v2/qris/qr-code?data=%s", encoded)
	} else {
		return fmt.Sprintf("https://api.sandbox.midtrans.com/v2/qris/qr-code?data=%s", encoded)
	}
}

// NewMidtransService creates a new instance of MidtransService
func NewMidtransService(config *MidtransConfig) *MidtransService {
	return &MidtransService{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}
