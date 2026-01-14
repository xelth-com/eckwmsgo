package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// OrderType defines the type of order
type OrderType string

const (
	OrderTypeRMA    OrderType = "rma"     // Return Merchandise Authorization
	OrderTypeRepair OrderType = "repair"   // Internal repair order
)

// OrderStatus defines possible order statuses
type OrderStatus string

const (
	OrderStatusPending    OrderStatus = "pending"     // Awaiting action
	OrderStatusInProgress OrderStatus = "in_progress" // Currently being worked on
	OrderStatusCompleted  OrderStatus = "completed"   // Finished
	OrderStatusCancelled  OrderStatus = "cancelled"   // Cancelled
)

// Order represents a unified order/request table for RMA and repairs
type Order struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	OrderNumber       string         `gorm:"uniqueIndex;not null" json:"order_number"`

	// Order classification
	OrderType         OrderType      `gorm:"not null;index" json:"order_type"` // rma | repair

	// Customer information (for RMA)
	CustomerName      string         `gorm:"index" json:"customer_name"`
	CustomerEmail     string         `json:"customer_email"`
	CustomerPhone     string         `json:"customer_phone"`

	// Item information
	ItemID            *uint          `gorm:"index" json:"item_id,omitempty"`
	ProductSKU        string         `gorm:"index" json:"product_sku"`
	ProductName       string         `json:"product_name"`
	SerialNumber      string         `gorm:"index" json:"serial_number"`
	PurchaseDate      *time.Time     `json:"purchase_date,omitempty"`

	// Problem/Issue description
	IssueDescription  string         `gorm:"type:text" json:"issue_description"`
	DiagnosisNotes    string         `gorm:"type:text" json:"diagnosis_notes"`

	// Assignment
	AssignedTo        *uint          `gorm:"index" json:"assigned_to,omitempty"` // User who handles the order

	// Status and priority
	Status            OrderStatus    `gorm:"default:pending;index" json:"status"`
	Priority          string         `gorm:"default:normal" json:"priority"` // low | normal | high | urgent

	// Repair-specific fields
	RepairNotes       string         `gorm:"type:text" json:"repair_notes"`
	PartsUsed         datatypes.JSON `json:"parts_used"`
	LaborHours        float64        `json:"labor_hours"`
	TotalCost         float64        `json:"total_cost"`

	// Resolution
	Resolution        string         `gorm:"type:text" json:"resolution"`
	Notes             string         `gorm:"type:text" json:"notes"`
	Metadata          datatypes.JSON `json:"metadata"`

	// RMA-specific fields
	RMAReason         string         `gorm:"column:rma_reason" json:"rma_reason"` // return reason
	IsRefundRequested bool           `gorm:"default:false" json:"is_refund_requested"`

	// Timestamps
	StartedAt         *time.Time     `json:"started_at,omitempty"`
	CompletedAt       *time.Time     `json:"completed_at,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	AssignedUser      *UserAuth      `gorm:"foreignKey:AssignedTo" json:"assigned_user,omitempty"`
	// Item relation removed - use ItemID to link to ProductProduct/StockLot via sync logic
}

// TableName specifies the table name for Order model
func (Order) TableName() string {
	return "orders"
}

// BeforeCreate generates order number before creating
func (o *Order) BeforeCreate(tx *gorm.DB) error {
	if o.OrderNumber == "" {
		prefix := "ORD"
		if o.OrderType == OrderTypeRMA {
			prefix = "RMA"
		} else if o.OrderType == OrderTypeRepair {
			prefix = "REP"
		}
		o.OrderNumber = generateOrderNumber(prefix)
	}
	return nil
}

// generateOrderNumber creates a unique order number
func generateOrderNumber(prefix string) string {
	return prefix + time.Now().Format("20060102") + "-" + randomString(4)
}

// randomString generates a random string of given length
func randomString(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	now := time.Now().UnixNano()
	for i := 0; i < length; i++ {
		result[i] = charset[(now+int64(i))%int64(len(charset))]
	}
	return string(result)
}

// IsRMA returns true if this is an RMA order
func (o *Order) IsRMA() bool {
	return o.OrderType == OrderTypeRMA
}

// IsRepair returns true if this is a repair order
func (o *Order) IsRepair() bool {
	return o.OrderType == OrderTypeRepair
}
