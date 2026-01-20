package models

import (
	"time"
)

// DeliveryCarrier represents a delivery service provider configuration
// This is analogous to Odoo's delivery.carrier model
type DeliveryCarrier struct {
	ID           int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Name         string    `gorm:"not null" json:"name"`                // e.g., "OPAL Express"
	ProviderCode string    `gorm:"uniqueIndex;not null" json:"provider_code"` // e.g., "opal", "dhl", "ups"
	Active       bool      `gorm:"default:true" json:"active"`
	ConfigJSON   string    `gorm:"type:text" json:"config_json"` // JSON-encoded provider-specific config
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (DeliveryCarrier) TableName() string { return "delivery_carrier" }

// StockPickingDelivery links a StockPicking to delivery information
// This extends the stock_picking model with delivery-specific data
type StockPickingDelivery struct {
	ID             int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	PickingID      *int64     `gorm:"uniqueIndex" json:"picking_id"`                 // Pointer to allow null (orphaned shipments from OPAL import)
	CarrierID      *int64     `gorm:"index" json:"carrier_id"`                       // Foreign key to delivery_carrier
	TrackingNumber string     `gorm:"index" json:"tracking_number"`
	CarrierPrice   float64    `json:"carrier_price"`
	Currency       string     `gorm:"default:EUR" json:"currency"`
	Status         string     `gorm:"index;default:draft" json:"status"` // draft, pending, shipped, delivered, error
	ErrorMessage   string     `gorm:"type:text" json:"error_message"`    // Error details if status = error
	LabelURL       string     `json:"label_url"`                         // URL to shipping label
	LabelData      []byte     `gorm:"type:bytea" json:"-"`               // Binary label data (PDF)
	RawResponse    string     `gorm:"type:text" json:"raw_response"`     // JSON response from provider
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	ShippedAt      *time.Time `json:"shipped_at"`  // When shipment was created
	DeliveredAt    *time.Time `json:"delivered_at"` // When shipment was delivered

	// Relations
	Picking *StockPicking    `gorm:"foreignKey:PickingID" json:"picking,omitempty"`
	Carrier *DeliveryCarrier `gorm:"foreignKey:CarrierID" json:"carrier,omitempty"`
}

func (StockPickingDelivery) TableName() string { return "stock_picking_delivery" }

// Delivery status constants
const (
	DeliveryStatusDraft     = "draft"     // Not yet processed
	DeliveryStatusPending   = "pending"   // Queued for shipment creation
	DeliveryStatusShipped   = "shipped"   // Shipment created successfully
	DeliveryStatusDelivered = "delivered" // Delivered to customer
	DeliveryStatusError     = "error"     // Error occurred during processing
	DeliveryStatusCancelled = "cancelled" // Shipment cancelled
)

// DeliveryTracking stores tracking history for a shipment
type DeliveryTracking struct {
	ID                   int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	PickingDeliveryID    int64     `gorm:"index;not null" json:"picking_delivery_id"`
	Timestamp            time.Time `gorm:"not null" json:"timestamp"`
	Status               string    `gorm:"not null" json:"status"`
	StatusCode           string    `json:"status_code"`    // Provider-specific status code
	Location             string    `json:"location"`       // Current location
	Description          string    `gorm:"type:text" json:"description"` // Event description
	CreatedAt            time.Time `json:"created_at"`

	// Relations
	PickingDelivery *StockPickingDelivery `gorm:"foreignKey:PickingDeliveryID" json:"picking_delivery,omitempty"`
}

func (DeliveryTracking) TableName() string { return "delivery_tracking" }
