package models

import "time"

type Receipt struct {
	ID           uint    `gorm:"primaryKey" json:"id"`
	OrderID      uint    `json:"order_id"`
	Order        Order   `gorm:"foreignKey:OrderID" json:"order"`
	PaymentID    uint    `json:"payment_id"`
	Payment      Payment `gorm:"foreignKey:PaymentID" json:"payment"`
	Total        float64 `gorm:"type:decimal(12,2);not null" json:"total"`
	RoundedTotal float64 `gorm:"type:decimal(12,2);not null" json:"rounded_total"`

	// Detail Pembayaran
	PaymentMethod    string  `gorm:"type:varchar(50);not null" json:"payment_method"`
	AmountPaid       float64 `gorm:"type:decimal(12,2);not null" json:"amount_paid"`
	Change           float64 `gorm:"type:decimal(12,2);not null" json:"change"`
	PaymentStatus    string  `gorm:"type:varchar(20);not null" json:"payment_status"`
	PaymentReference string  `gorm:"type:varchar(100)" json:"payment_reference"`

	// Items Detail akan disimpan dalam tabel terpisah
	ReceiptItems []ReceiptItem `gorm:"foreignKey:ReceiptID" json:"receipt_items"`

	ReceiptNumber string    `json:"receipt_number"`
	CreatedAt     time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt     time.Time `gorm:"not null" json:"updated_at"`
}

type ReceiptItem struct {
	ID        uint    `gorm:"primaryKey" json:"id"`
	ReceiptID uint    `gorm:"not null" json:"receipt_id"`
	Receipt   Receipt `gorm:"-" json:"-"`

	// Item Info
	MenuID    uint    `gorm:"not null" json:"menu_id"`
	MenuName  string  `gorm:"type:varchar(100);not null" json:"menu_name"`
	Quantity  int     `gorm:"not null" json:"quantity"`
	UnitPrice float64 `gorm:"type:decimal(12,2);not null" json:"unit_price"`
	Subtotal  float64 `gorm:"type:decimal(12,2);not null" json:"subtotal"`
	Notes     string  `gorm:"type:text" json:"notes"`

	// Add-on items akan disimpan dalam tabel terpisah
	AddOnItems []ReceiptAddOn `gorm:"foreignKey:ReceiptItemID" json:"add_on_items"`

	CreatedAt time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null" json:"updated_at"`
}

type ReceiptAddOn struct {
	ID            uint        `gorm:"primaryKey" json:"id"`
	ReceiptItemID uint        `gorm:"not null" json:"receipt_item_id"`
	ReceiptItem   ReceiptItem `gorm:"-" json:"-"`

	MenuID   uint    `gorm:"not null" json:"menu_id"`
	Name     string  `gorm:"type:varchar(100);not null" json:"name"`
	Quantity int     `gorm:"not null" json:"quantity"`
	Price    float64 `gorm:"type:decimal(12,2);not null" json:"price"`

	CreatedAt time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null" json:"updated_at"`
}
