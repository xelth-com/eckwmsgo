package models

import (
	"fmt"
	"time"
)

// DeliveryCarrier represents a delivery service provider configuration
// This is analogous to Odoo's delivery.carrier model
type DeliveryCarrier struct {
	ID           int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Name         string    `gorm:"not null" json:"name"`                // e.g., "OPAL Express"
	ProviderCode string    `gorm:"uniqueIndex;not null" json:"providerCode"` // e.g., "opal", "dhl", "ups"
	Active       bool      `gorm:"default:true" json:"active"`
	ConfigJSON   string    `gorm:"type:text" json:"configJson"` // JSON-encoded provider-specific config
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

func (DeliveryCarrier) TableName() string { return "delivery_carrier" }

// StockPickingDelivery links a StockPicking to delivery information
// This extends the stock_picking model with delivery-specific data
type StockPickingDelivery struct {
	ID             int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	PickingID      *int64     `gorm:"uniqueIndex" json:"pickingId"`                 // Pointer to allow null (orphaned shipments from OPAL import)
	CarrierID      *int64     `gorm:"index" json:"carrierId"`                       // Foreign key to delivery_carrier
	TrackingNumber string     `gorm:"index" json:"trackingNumber"`
	CarrierPrice   float64    `json:"carrierPrice"`
	Currency       string     `gorm:"default:EUR" json:"currency"`
	Status         string     `gorm:"index;default:draft" json:"status"` // draft, pending, shipped, delivered, error
	ErrorMessage   string     `gorm:"type:text" json:"errorMessage"`    // Error details if status = error
	LabelURL       string     `json:"labelUrl"`                         // URL to shipping label
	LabelData      []byte     `gorm:"type:bytea" json:"-"`               // Binary label data (PDF)
	RawResponse    string     `gorm:"type:text" json:"rawResponse"`     // JSON response from provider
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
	ShippedAt      *time.Time `json:"shippedAt"`      // When shipment was created
	DeliveredAt    *time.Time `json:"deliveredAt"`    // When shipment was delivered
	LastActivityAt *time.Time `gorm:"index" json:"lastActivityAt"` // Latest known activity date for sorting

	// Relations
	Picking *StockPicking    `gorm:"foreignKey:PickingID" json:"picking,omitempty"`
	Carrier *DeliveryCarrier `gorm:"foreignKey:CarrierID" json:"carrier,omitempty"`
}

func (StockPickingDelivery) TableName() string { return "stock_picking_delivery" }

// GetEntityID implements SyncableEntity interface
func (s StockPickingDelivery) GetEntityID() string {
	return fmt.Sprintf("%d", s.ID)
}

// GetEntityType implements SyncableEntity interface
func (s StockPickingDelivery) GetEntityType() string {
	return "shipment"
}

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
	PickingDeliveryID    int64     `gorm:"index;not null" json:"pickingDeliveryId"`
	Timestamp            time.Time `gorm:"not null" json:"timestamp"`
	Status               string    `gorm:"not null" json:"status"`
	StatusCode           string    `json:"statusCode"`    // Provider-specific status code
	Location             string    `json:"location"`       // Current location
	Description          string    `gorm:"type:text" json:"description"` // Event description
	CreatedAt            time.Time `json:"createdAt"`

	// Relations
	PickingDelivery *StockPickingDelivery `gorm:"foreignKey:PickingDeliveryID" json:"picking_delivery,omitempty"`
}

func (DeliveryTracking) TableName() string { return "delivery_tracking" }

// GetEntityID implements SyncableEntity interface
func (t DeliveryTracking) GetEntityID() string {
	return fmt.Sprintf("%d", t.ID)
}

// GetEntityType implements SyncableEntity interface
func (t DeliveryTracking) GetEntityType() string {
	return "tracking"
}
