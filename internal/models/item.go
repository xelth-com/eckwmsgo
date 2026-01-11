package models

import (
	"time"

	"gorm.io/gorm"
)

// Item represents an item in the warehouse
type Item struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	SKU         string         `gorm:"unique;not null" json:"sku"`
	Name        string         `gorm:"not null" json:"name"`
	Description string         `json:"description"`
	Category    string         `json:"category"`
	Barcode     string         `gorm:"unique" json:"barcode"`
	PlaceID     *uint          `gorm:"index" json:"place_id,omitempty"`
	BoxID       *uint          `gorm:"index" json:"box_id,omitempty"`
	Quantity    int            `gorm:"default:0" json:"quantity"`
	MinStock    int            `gorm:"default:0" json:"min_stock"`
	MaxStock    int            `json:"max_stock"`
	UnitPrice   float64        `json:"unit_price"`
	Status      string         `gorm:"type:varchar(50);default:'available'" json:"status"`
	IsActive    bool           `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	// Sync metadata
	SyncVersion      int64  `gorm:"default:1" json:"sync_version"`
	VectorClock      JSONB  `gorm:"type:jsonb;default:'{}'" json:"vector_clock"`
	SourceInstance   string `gorm:"type:varchar(255)" json:"source_instance"`
	SourceDevice     *string `gorm:"type:varchar(255)" json:"source_device,omitempty"`
	SourcePriority   int    `gorm:"default:40" json:"source_priority"`
	ContentHash      string `gorm:"type:varchar(64)" json:"content_hash"`
	SyncedAt         *time.Time `json:"synced_at,omitempty"`

	// Relations
	Place       *Place         `gorm:"foreignKey:PlaceID" json:"place,omitempty"`
	Box         *Box           `gorm:"foreignKey:BoxID" json:"box,omitempty"`
}

// TableName specifies the table name for Item model
func (Item) TableName() string {
	return "items"
}

// Box represents a container for items
type Box struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	BoxNumber   string         `gorm:"unique;not null" json:"box_number"`
	Name        string         `json:"name"`
	Barcode     string         `gorm:"unique" json:"barcode"`
	PlaceID     *uint          `gorm:"index" json:"place_id,omitempty"`
	Capacity    int            `json:"capacity"`
	IsActive    bool           `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	// Sync metadata
	SyncVersion      int64  `gorm:"default:1" json:"sync_version"`
	VectorClock      JSONB  `gorm:"type:jsonb;default:'{}'" json:"vector_clock"`
	SourceInstance   string `gorm:"type:varchar(255)" json:"source_instance"`
	SourceDevice     *string `gorm:"type:varchar(255)" json:"source_device,omitempty"`
	SourcePriority   int    `gorm:"default:40" json:"source_priority"`
	ContentHash      string `gorm:"type:varchar(64)" json:"content_hash"`
	SyncedAt         *time.Time `json:"synced_at,omitempty"`

	// Relations
	Place       *Place         `gorm:"foreignKey:PlaceID" json:"place,omitempty"`
	Items       []Item         `gorm:"foreignKey:BoxID" json:"items,omitempty"`
}

// TableName specifies the table name for Box model
func (Box) TableName() string {
	return "boxes"
}

// ProductAlias represents an alias for a product
type ProductAlias struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	ProductSKU  string         `gorm:"not null;index" json:"product_sku"`
	Alias       string         `gorm:"not null" json:"alias"`
	AliasType   string         `json:"alias_type"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name for ProductAlias model
func (ProductAlias) TableName() string {
	return "product_aliases"
}
