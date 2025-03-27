package services

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMidtransService_ValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *MidtransConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &MidtransConfig{
				ServerKey:     "test-server-key",
				ClientKey:     "test-client-key",
				MerchantID:    "test-merchant-id",
				MerchantName:  "test-merchant",
				MerchantEmail: "test@merchant.com",
				MerchantPhone: "1234567890",
				WebhookURL:    "https://test.com/webhook",
			},
			wantErr: false,
		},
		{
			name: "missing server key",
			config: &MidtransConfig{
				ClientKey:     "test-client-key",
				MerchantID:    "test-merchant-id",
				MerchantName:  "test-merchant",
				MerchantEmail: "test@merchant.com",
				MerchantPhone: "1234567890",
				WebhookURL:    "https://test.com/webhook",
			},
			wantErr: true,
		},
		{
			name: "missing merchant id",
			config: &MidtransConfig{
				ServerKey:     "test-server-key",
				ClientKey:     "test-client-key",
				MerchantName:  "test-merchant",
				MerchantEmail: "test@merchant.com",
				MerchantPhone: "1234567890",
				WebhookURL:    "https://test.com/webhook",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := &MidtransService{
				config: tt.config,
			}
			err := ms.ValidateConfig()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMidtransService_CheckTransactionStatus(t *testing.T) {
	tests := []struct {
		name           string
		orderID        string
		mockResponse   string
		mockStatusCode int
		wantStatus     string
		wantErr        bool
	}{
		{
			name:           "success status",
			orderID:        "test-order-1",
			mockResponse:   `{"transaction_status": "settlement"}`,
			mockStatusCode: http.StatusOK,
			wantStatus:     "success",
			wantErr:        false,
		},
		{
			name:           "pending status",
			orderID:        "test-order-2",
			mockResponse:   `{"transaction_status": "pending"}`,
			mockStatusCode: http.StatusOK,
			wantStatus:     "pending",
			wantErr:        false,
		},
		{
			name:           "failed status",
			orderID:        "test-order-3",
			mockResponse:   `{"transaction_status": "failure"}`,
			mockStatusCode: http.StatusOK,
			wantStatus:     "failed",
			wantErr:        false,
		},
		{
			name:           "api error",
			orderID:        "test-order-4",
			mockResponse:   `{"error": "Invalid order ID"}`,
			mockStatusCode: http.StatusBadRequest,
			wantStatus:     "",
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			ms := &MidtransService{
				config: &MidtransConfig{
					ServerKey: "test-server-key",
				},
				httpClient: server.Client(),
			}

			status, err := ms.CheckTransactionStatus(tt.orderID)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckTransactionStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
			if status != tt.wantStatus {
				t.Errorf("CheckTransactionStatus() status = %v, want %v", status, tt.wantStatus)
			}
		})
	}
}

func TestMidtransService_ValidateSignature(t *testing.T) {
	tests := []struct {
		name        string
		orderID     string
		statusCode  string
		grossAmount string
		signature   string
		serverKey   string
		wantValid   bool
	}{
		{
			name:        "valid signature",
			orderID:     "test-order-1",
			statusCode:  "200",
			grossAmount: "10000",
			signature:   "valid-signature", // This should be replaced with actual calculated signature
			serverKey:   "test-server-key",
			wantValid:   true,
		},
		{
			name:        "invalid signature",
			orderID:     "test-order-2",
			statusCode:  "200",
			grossAmount: "10000",
			signature:   "invalid-signature",
			serverKey:   "test-server-key",
			wantValid:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := &MidtransService{
				config: &MidtransConfig{
					ServerKey: tt.serverKey,
				},
			}

			valid := ms.ValidateSignature(tt.orderID, tt.statusCode, tt.grossAmount, tt.signature)
			if valid != tt.wantValid {
				t.Errorf("ValidateSignature() valid = %v, want %v", valid, tt.wantValid)
			}
		})
	}
}
