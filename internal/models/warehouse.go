package models

import (
	"time"

	"gorm.io/gorm"
)

// Warehouse represents a warehouse in the system
type Warehouse struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"not null;unique" json:"name"`
	Location    string         `json:"location"`
	Description string         `json:"description"`
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
	HierarchyHash    string `gorm:"type:varchar(64)" json:"hierarchy_hash"`
	SyncedAt         *time.Time `json:"synced_at,omitempty"`

	// Relations
	Racks       []WarehouseRack `gorm:"foreignKey:WarehouseID" json:"racks,omitempty"`
}

// TableName specifies the table name for Warehouse model
func (Warehouse) TableName() string {
	return "warehouses"
}

// WarehouseRack represents a rack in a warehouse
type WarehouseRack struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	WarehouseID uint           `gorm:"not null;index" json:"warehouse_id"`
	Name        string         `gorm:"not null" json:"name"`
	Section     string         `json:"section"`
	Level       int            `json:"level"`
	Position    int            `json:"position"`
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
	HierarchyHash    string `gorm:"type:varchar(64)" json:"hierarchy_hash"`
	SyncedAt         *time.Time `json:"synced_at,omitempty"`

	// Relations
	Warehouse   Warehouse      `gorm:"foreignKey:WarehouseID" json:"warehouse,omitempty"`
	Places      []Place        `gorm:"foreignKey:RackID" json:"places,omitempty"`
}

// TableName specifies the table name for WarehouseRack model
func (WarehouseRack) TableName() string {
	return "warehouse_racks"
}

// Place represents a specific location in a rack
type Place struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	RackID      uint           `gorm:"not null;index" json:"rack_id"`
	Name        string         `gorm:"not null" json:"name"`
	Row         int            `json:"row"`
	Column      int            `json:"column"`
	Barcode     string         `gorm:"unique" json:"barcode"`
	Position    string         `json:"position"`
	IsOccupied  bool           `gorm:"default:false" json:"is_occupied"`
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
	HierarchyHash    string `gorm:"type:varchar(64)" json:"hierarchy_hash"`
	SyncedAt         *time.Time `json:"synced_at,omitempty"`

	// Relations
	Rack        WarehouseRack  `gorm:"foreignKey:RackID" json:"rack,omitempty"`
}

// TableName specifies the table name for Place model
func (Place) TableName() string {
	return "places"
}
