package models

import (
	"time"

	"gorm.io/datatypes"
)

// ProductProduct mirrors Odoo 'product.product'.
// Represents a specific variant of a product.
type ProductProduct struct {
	ID            int64          `gorm:"primaryKey;autoIncrement:false" json:"id" xmlrpc:"id"` // Odoo ID is the PK
	DefaultCode   string         `gorm:"index" json:"default_code" xmlrpc:"default_code"`      // SKU / Internal Reference
	Barcode       string         `gorm:"index" json:"barcode" xmlrpc:"barcode"`                // EAN13
	Name          string         `json:"name" xmlrpc:"name"`
	Active        bool           `gorm:"default:true" json:"active" xmlrpc:"active"`
	Type          string         `json:"type" xmlrpc:"type"`                         // product, consu, service
	ListPrice     float64        `json:"list_price" xmlrpc:"list_price"`             // Sales Price
	StandardPrice float64        `json:"standard_price" xmlrpc:"standard_price"`     // Cost
	Weight        float64        `json:"weight" xmlrpc:"weight"`
	Volume        float64        `json:"volume" xmlrpc:"volume"`
	WriteDate     time.Time      `json:"write_date" xmlrpc:"write_date"`

	// Sync Meta
	LastSyncedAt  time.Time      `json:"last_synced_at"`
	RawData       datatypes.JSON `gorm:"type:jsonb" json:"raw_data"`

	// Relations
	StockQuants   []StockQuant   `gorm:"foreignKey:ProductID" json:"quants,omitempty"`
	StockLots     []StockLot     `gorm:"foreignKey:ProductID" json:"lots,omitempty"`
}

// TableName overrides standard GORM naming to match Odoo convention (optional but cleaner)
func (ProductProduct) TableName() string {
	return "product_product"
}
