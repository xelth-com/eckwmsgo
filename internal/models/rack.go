package models

import "time"

// WarehouseRack represents a storage rack on the visual warehouse blueprint
type WarehouseRack struct {
	ID           int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Name         string    `gorm:"type:varchar(100);not null" json:"name"`
	Prefix       string    `gorm:"type:varchar(10)" json:"prefix,omitempty"`
	Columns      int       `gorm:"not null;default:1" json:"columns"`
	Rows         int       `gorm:"not null;default:1" json:"rows"`
	StartIndex   int       `gorm:"not null" json:"start_index"`
	SortOrder    int       `gorm:"default:0" json:"sort_order"`
	WarehouseID  *int64    `gorm:"index" json:"warehouse_id,omitempty"`

	// Visual positioning for Blueprint editor
	PosX         int       `gorm:"default:0" json:"posX"`
	PosY         int       `gorm:"default:0" json:"posY"`
	Rotation     int       `gorm:"default:0" json:"rotation"`      // 0, 90, 180, 270
	VisualWidth  int       `gorm:"default:0" json:"visual_width"`  // Override width in px
	VisualHeight int       `gorm:"default:0" json:"visual_height"` // Override height in px

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relation to parent warehouse location
	Warehouse *StockLocation `gorm:"foreignKey:WarehouseID" json:"warehouse,omitempty"`
}

func (WarehouseRack) TableName() string { return "warehouse_racks" }
