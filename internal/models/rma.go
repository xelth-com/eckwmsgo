package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// RmaRequest represents a return merchandise authorization request
type RmaRequest struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	RMANumber         string         `gorm:"unique;not null" json:"rmaNumber"`
	CustomerName      string         `gorm:"not null" json:"customerName"`
	CustomerEmail     string         `json:"customerEmail"`
	CustomerPhone     string         `json:"customerPhone"`
	ProductSKU        string         `gorm:"not null;index" json:"productSku"`
	ProductName       string         `json:"productName"`
	SerialNumber      string         `json:"serialNumber"`
	PurchaseDate      *time.Time     `json:"purchaseDate,omitempty"`
	IssueDescription  string         `gorm:"type:text" json:"issueDescription"`
	Status            string         `gorm:"default:'pending'" json:"status"`
	Priority          string         `gorm:"default:'normal'" json:"priority"`
	AssignedTo        *uint          `gorm:"index" json:"assignedTo,omitempty"`
	Resolution        string         `gorm:"type:text" json:"resolution"`
	Notes             string         `gorm:"type:text" json:"notes"`
	Metadata          datatypes.JSON `json:"metadata"`
	CreatedAt         time.Time      `json:"createdAt"`
	UpdatedAt         time.Time      `json:"updatedAt"`
	ResolvedAt        *time.Time     `json:"resolvedAt,omitempty"`
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
	OrderNumber       string         `gorm:"unique;not null" json:"orderNumber"`
	RMARequestID      *uint          `gorm:"index" json:"rmaRequestId,omitempty"`
	ItemID            uint           `gorm:"not null;index" json:"itemId"`
	TechnicianID      *uint          `gorm:"index" json:"technicianId,omitempty"`
	Status            string         `gorm:"default:'pending'" json:"status"`
	DiagnosisNotes    string         `gorm:"type:text" json:"diagnosisNotes"`
	RepairNotes       string         `gorm:"type:text" json:"repairNotes"`
	PartsUsed         datatypes.JSON `json:"partsUsed"`
	LaborHours        float64        `json:"laborHours"`
	TotalCost         float64        `json:"totalCost"`
	StartedAt         *time.Time     `json:"startedAt,omitempty"`
	CompletedAt       *time.Time     `json:"completedAt,omitempty"`
	CreatedAt         time.Time      `json:"createdAt"`
	UpdatedAt         time.Time      `json:"updatedAt"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	RMARequest        *RmaRequest    `gorm:"foreignKey:RMARequestID" json:"rma_request,omitempty"`
	Item              ProductProduct           `gorm:"foreignKey:ItemID" json:"item,omitempty"`
	Technician        *UserAuth      `gorm:"foreignKey:TechnicianID" json:"technician,omitempty"`
}

// TableName specifies the table name for RepairOrder model
func (RepairOrder) TableName() string {
	return "repair_orders"
}
