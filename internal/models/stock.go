package models

import "time"

// StockLocation (p-code)
type StockLocation struct {
	ID           int64          `gorm:"primaryKey;autoIncrement:false" json:"id" xmlrpc:"id"`
	Name         string         `json:"name" xmlrpc:"name"`
	CompleteName string         `gorm:"index" json:"complete_name" xmlrpc:"complete_name"`
	Barcode      string         `gorm:"uniqueIndex" json:"barcode" xmlrpc:"barcode"` // 'p' code
	Usage        string         `json:"usage" xmlrpc:"usage"`
	LocationID   *int64         `json:"location_id" xmlrpc:"location_id"`
	Active       bool           `gorm:"default:true" json:"active" xmlrpc:"active"`

	// Sync Meta
	LastSyncedAt time.Time      `json:"last_synced_at"`

	Parent       *StockLocation  `gorm:"foreignKey:LocationID" json:"parent,omitempty"`
	Children     []StockLocation `gorm:"foreignKey:LocationID" json:"children,omitempty"`
}

func (StockLocation) TableName() string { return "stock_location" }

// StockLot (i-code serial part)
type StockLot struct {
	ID         int64     `gorm:"primaryKey;autoIncrement:false" json:"id" xmlrpc:"id"`
	Name       string    `gorm:"uniqueIndex" json:"name" xmlrpc:"name"` // 'i' code serial part
	ProductID  int64     `gorm:"index" json:"product_id" xmlrpc:"product_id"`
	Ref        string    `json:"ref" xmlrpc:"ref"` // Internal reference
	CreateDate time.Time `json:"create_date" xmlrpc:"create_date"`
}

func (StockLot) TableName() string { return "stock_lot" }

// StockPackageType (Definition for b-code types)
type StockPackageType struct {
	ID        int64   `gorm:"primaryKey;autoIncrement:false" json:"id" xmlrpc:"id"`
	Name      string  `json:"name" xmlrpc:"name"`
	Barcode   string  `json:"barcode" xmlrpc:"barcode"` // The 'T' char in b-code
	MaxWeight float64 `json:"max_weight" xmlrpc:"max_weight"`
	Length    int     `json:"packaging_length" xmlrpc:"packaging_length"`
	Width     int     `json:"width" xmlrpc:"width"`
	Height    int     `json:"height" xmlrpc:"height"`
}

func (StockPackageType) TableName() string { return "stock_package_type" }

// StockQuantPackage (b-code instance)
type StockQuantPackage struct {
	ID            int64     `gorm:"primaryKey;autoIncrement:false" json:"id" xmlrpc:"id"`
	Name          string    `gorm:"uniqueIndex" json:"name" xmlrpc:"name"` // 'b' code
	PackageTypeID *int64    `gorm:"index" json:"package_type_id" xmlrpc:"package_type_id"`
	LocationID    *int64    `json:"location_id" xmlrpc:"location_id"`
	PackDate      time.Time `json:"pack_date"`

	PackageType   *StockPackageType `gorm:"foreignKey:PackageTypeID"`
}

func (StockQuantPackage) TableName() string { return "stock_quant_package" }

// StockQuant (Inventory)
type StockQuant struct {
	ID          int64   `gorm:"primaryKey;autoIncrement:false" json:"id" xmlrpc:"id"`
	ProductID   int64   `gorm:"index" json:"product_id" xmlrpc:"product_id"`
	LocationID  int64   `gorm:"index" json:"location_id" xmlrpc:"location_id"`
	LotID       *int64  `gorm:"index" json:"lot_id" xmlrpc:"lot_id"`
	PackageID   *int64  `gorm:"index" json:"package_id" xmlrpc:"package_id"`
	Quantity    float64 `json:"quantity" xmlrpc:"quantity"`
	ReservedQty float64 `json:"reserved_quantity" xmlrpc:"reserved_quantity"`
}

func (StockQuant) TableName() string { return "stock_quant" }

// StockPicking (Move Order)
type StockPicking struct {
	ID             int64     `gorm:"primaryKey;autoIncrement:false" json:"id" xmlrpc:"id"`
	Name           string    `gorm:"uniqueIndex" json:"name" xmlrpc:"name"`
	State          string    `gorm:"index" json:"state" xmlrpc:"state"`
	LocationID     int64     `json:"location_id" xmlrpc:"location_id"`
	LocationDestID int64     `json:"location_dest_id" xmlrpc:"location_dest_id"`
	ScheduledDate  time.Time `json:"scheduled_date" xmlrpc:"scheduled_date"`
}

func (StockPicking) TableName() string { return "stock_picking" }

// StockMoveLine (Move Detail)
type StockMoveLine struct {
	ID              int64   `gorm:"primaryKey;autoIncrement:false" json:"id" xmlrpc:"id"`
	PickingID       int64   `gorm:"index" json:"picking_id" xmlrpc:"picking_id"`
	ProductID       int64   `gorm:"index" json:"product_id" xmlrpc:"product_id"`
	QtyDone         float64 `json:"qty_done" xmlrpc:"qty_done"`
	LocationID      int64   `json:"location_id" xmlrpc:"location_id"`
	LocationDestID  int64   `json:"location_dest_id" xmlrpc:"location_dest_id"`
	PackageID       *int64  `json:"package_id" xmlrpc:"package_id"`
	ResultPackageID *int64  `json:"result_package_id" xmlrpc:"result_package_id"`
	LotID           *int64  `json:"lot_id" xmlrpc:"lot_id"`
}

func (StockMoveLine) TableName() string { return "stock_move_line" }
