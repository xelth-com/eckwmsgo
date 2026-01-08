package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// RmaRequest represents a return merchandise authorization request
type RmaRequest struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	RMANumber         string         `gorm:"unique;not null" json:"rma_number"`
	CustomerName      string         `gorm:"not null" json:"customer_name"`
	CustomerEmail     string         `json:"customer_email"`
	CustomerPhone     string         `json:"customer_phone"`
	ProductSKU        string         `gorm:"not null;index" json:"product_sku"`
	ProductName       string         `json:"product_name"`
	SerialNumber      string         `json:"serial_number"`
	PurchaseDate      *time.Time     `json:"purchase_date,omitempty"`
	IssueDescription  string         `gorm:"type:text" json:"issue_description"`
	Status            string         `gorm:"default:'pending'" json:"status"`
	Priority          string         `gorm:"default:'normal'" json:"priority"`
	AssignedTo        *uint          `gorm:"index" json:"assigned_to,omitempty"`
	Resolution        string         `gorm:"type:text" json:"resolution"`
	Notes             string         `gorm:"type:text" json:"notes"`
	Metadata          datatypes.JSON `json:"metadata"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	ResolvedAt        *time.Time     `json:"resolved_at,omitempty"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	AssignedUser      *UserAuth      `gorm:"foreignKey:AssignedTo" json:"assigned_user,omitempty"`
}

// TableName specifies the table name for RmaRequest model
func (RmaRequest) TableName() string {
	return "rma_requests"
}

// RepairOrder represents a repair order for an item
type RepairOrder struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	OrderNumber       string         `gorm:"unique;not null" json:"order_number"`
	RMARequestID      *uint          `gorm:"index" json:"rma_request_id,omitempty"`
	ItemID            uint           `gorm:"not null;index" json:"item_id"`
	TechnicianID      *uint          `gorm:"index" json:"technician_id,omitempty"`
	Status            string         `gorm:"default:'pending'" json:"status"`
	DiagnosisNotes    string         `gorm:"type:text" json:"diagnosis_notes"`
	RepairNotes       string         `gorm:"type:text" json:"repair_notes"`
	PartsUsed         datatypes.JSON `json:"parts_used"`
	LaborHours        float64        `json:"labor_hours"`
	TotalCost         float64        `json:"total_cost"`
	StartedAt         *time.Time     `json:"started_at,omitempty"`
	CompletedAt       *time.Time     `json:"completed_at,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	RMARequest        *RmaRequest    `gorm:"foreignKey:RMARequestID" json:"rma_request,omitempty"`
	Item              Item           `gorm:"foreignKey:ItemID" json:"item,omitempty"`
	Technician        *UserAuth      `gorm:"foreignKey:TechnicianID" json:"technician,omitempty"`
}

// TableName specifies the table name for RepairOrder model
func (RepairOrder) TableName() string {
	return "repair_orders"
}
