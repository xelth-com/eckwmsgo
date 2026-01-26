package models

import (
	"strconv"
	"time"

	"gorm.io/datatypes"
)

// ProductProduct mirrors Odoo 'product.product'
type ProductProduct struct {
	ID            int64      `gorm:"primaryKey;autoIncrement:false" json:"id" xmlrpc:"id"`
	DefaultCode   OdooString `gorm:"index" json:"default_code" xmlrpc:"default_code"` // SKU (Fixed type)
	Barcode       OdooString `gorm:"index" json:"barcode" xmlrpc:"barcode"`           // EAN13 (Fixed type)
	Name          string     `json:"name" xmlrpc:"name"`
	Active        bool       `gorm:"default:true" json:"active" xmlrpc:"active"`
	Type          string     `json:"type" xmlrpc:"type"`
	ListPrice     float64    `json:"list_price" xmlrpc:"list_price"`
	StandardPrice float64    `json:"standard_price" xmlrpc:"standard_price"`
	Weight        float64    `json:"weight" xmlrpc:"weight"`
	Volume        float64    `json:"volume" xmlrpc:"volume"`
	WriteDate     time.Time  `json:"write_date" xmlrpc:"write_date"`

	LastSyncedAt time.Time      `json:"last_synced_at"`
	RawData      datatypes.JSON `gorm:"type:jsonb" json:"raw_data"`

	StockLots []StockLot `gorm:"foreignKey:ProductID" json:"lots,omitempty"`
}

func (ProductProduct) TableName() string { return "product_product" }

// GetEntityID implements SyncableEntity interface
func (p ProductProduct) GetEntityID() string { return strconv.FormatInt(p.ID, 10) }

// GetEntityType implements SyncableEntity interface
func (p ProductProduct) GetEntityType() string { return "product" }
