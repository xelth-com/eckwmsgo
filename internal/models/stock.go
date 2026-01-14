package models

import (
	"time"
)

// StockLocation mirrors 'stock.location'.
// ECK ID Prefix: 'p' (mapped to Barcode field)
type StockLocation struct {
	ID           int64          `gorm:"primaryKey;autoIncrement:false" json:"id" xmlrpc:"id"`
	Name         string         `json:"name" xmlrpc:"name"`
	CompleteName string         `gorm:"index" json:"complete_name" xmlrpc:"complete_name"` // "WH/Stock/Shelf 1"
	Barcode      string         `gorm:"uniqueIndex" json:"barcode" xmlrpc:"barcode"`       // ECK 'p' ID goes here
	Usage        string         `json:"usage" xmlrpc:"usage"`                              // internal, supplier, customer...
	LocationID   *int64         `json:"location_id" xmlrpc:"location_id"`                  // Parent Location
	Active       bool           `gorm:"default:true" json:"active" xmlrpc:"active"`

	// Sync Meta
	LastSyncedAt time.Time      `json:"last_synced_at"`

	// Relations
	Parent       *StockLocation  `gorm:"foreignKey:LocationID" json:"parent,omitempty"`
	Children     []StockLocation `gorm:"foreignKey:LocationID" json:"children,omitempty"`
}

func (StockLocation) TableName() string {
	return "stock_location"
}

// StockLot mirrors 'stock.lot' (Serial Numbers / Lots).
// ECK ID Prefix: 'i' (mapped to Name or Ref)
type StockLot struct {
	ID         int64     `gorm:"primaryKey;autoIncrement:false" json:"id" xmlrpc:"id"`
	Name       string    `gorm:"uniqueIndex" json:"name" xmlrpc:"name"` // ECK 'i' ID (Serial Number)
	ProductID  int64     `gorm:"index" json:"product_id" xmlrpc:"product_id"`
	Ref        string    `json:"ref" xmlrpc:"ref"` // Internal Reference
	CreateDate time.Time `json:"create_date" xmlrpc:"create_date"`

	// Relations
	Product    ProductProduct `gorm:"foreignKey:ProductID"`
}

func (StockLot) TableName() string {
	return "stock_lot"
}

// StockQuantPackage mirrors 'stock.quant.package' (Boxes / Pallets).
// ECK ID Prefix: 'b' (mapped to Name)
type StockQuantPackage struct {
	ID         int64     `gorm:"primaryKey;autoIncrement:false" json:"id" xmlrpc:"id"`
	Name       string    `gorm:"uniqueIndex" json:"name" xmlrpc:"name"` // ECK 'b' ID (LPN)
	PackDate   time.Time `json:"pack_date"`
	LocationID *int64    `json:"location_id" xmlrpc:"location_id"` // Current location of the box
}

func (StockQuantPackage) TableName() string {
	return "stock_quant_package"
}

// StockQuant mirrors 'stock.quant'.
// Represents physical inventory: "Product X is at Location Y, Lot Z, inside Package B, Qty N"
type StockQuant struct {
	ID               int64      `gorm:"primaryKey;autoIncrement:false" json:"id" xmlrpc:"id"`
	ProductID        int64      `gorm:"index" json:"product_id" xmlrpc:"product_id"`
	LocationID       int64      `gorm:"index" json:"location_id" xmlrpc:"location_id"`
	LotID            *int64     `gorm:"index" json:"lot_id" xmlrpc:"lot_id"`
	PackageID        *int64     `gorm:"index" json:"package_id" xmlrpc:"package_id"`
	Quantity         float64    `json:"quantity" xmlrpc:"quantity"`
	ReservedQuantity float64    `json:"reserved_quantity" xmlrpc:"reserved_quantity"`
	InventoryDate    *time.Time `json:"inventory_date" xmlrpc:"inventory_date"`

	// Relations
	Product  ProductProduct     `gorm:"foreignKey:ProductID"`
	Location StockLocation      `gorm:"foreignKey:LocationID"`
	Lot      StockLot           `gorm:"foreignKey:LotID"`
	Package  StockQuantPackage  `gorm:"foreignKey:PackageID"`
}

func (StockQuant) TableName() string {
	return "stock_quant"
}

// StockPicking mirrors 'stock.picking' (Transfer Orders)
type StockPicking struct {
	ID             int64     `gorm:"primaryKey;autoIncrement:false" json:"id" xmlrpc:"id"`
	Name           string    `gorm:"uniqueIndex" json:"name" xmlrpc:"name"` // WH/IN/0001
	State          string    `gorm:"index" json:"state" xmlrpc:"state"`     // draft, waiting, confirmed, assigned, done
	LocationID     int64     `json:"location_id" xmlrpc:"location_id"`      // Source
	LocationDestID int64     `json:"location_dest_id" xmlrpc:"location_dest_id"` // Dest
	Origin         string    `json:"origin" xmlrpc:"origin"`
	ScheduledDate  time.Time `json:"scheduled_date" xmlrpc:"scheduled_date"`
}

func (StockPicking) TableName() string {
	return "stock_picking"
}

// StockMoveLine mirrors 'stock.move.line' (Detailed Operations)
type StockMoveLine struct {
	ID              int64   `gorm:"primaryKey;autoIncrement:false" json:"id" xmlrpc:"id"`
	PickingID       int64   `gorm:"index" json:"picking_id" xmlrpc:"picking_id"`
	ProductID       int64   `gorm:"index" json:"product_id" xmlrpc:"product_id"`
	LocationID      int64   `json:"location_id" xmlrpc:"location_id"`
	LocationDestID  int64   `json:"location_dest_id" xmlrpc:"location_dest_id"`
	LotID           *int64  `json:"lot_id" xmlrpc:"lot_id"`
	PackageID       *int64  `json:"package_id" xmlrpc:"package_id"`
	ResultPackageID *int64  `json:"result_package_id" xmlrpc:"result_package_id"`
	QtyDone         float64 `json:"qty_done" xmlrpc:"qty_done"`
}

func (StockMoveLine) TableName() string {
	return "stock_move_line"
}
