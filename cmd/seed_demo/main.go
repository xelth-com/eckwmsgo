package main

import (
	"fmt"
	"log"
	"time"

	"github.com/xelth-com/eckwmsgo/internal/config"
	"github.com/xelth-com/eckwmsgo/internal/database"
	"github.com/xelth-com/eckwmsgo/internal/models"
)

func main() {
	fmt.Println("🌱 eckWMS Demo Data Seeder")
	fmt.Println("=" + string(make([]rune, 60)))

	// Load config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌ Failed to load config: %v", err)
	}

	// Connect to database
	db, err := database.Connect(cfg.Database)
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	defer db.Close()

	fmt.Println("✅ Connected to database")
	fmt.Println()

	// Run migrations first
	fmt.Println("🔨 Running database migrations...")
	err = db.AutoMigrate(
		&models.ProductProduct{},
		&models.StockLocation{},
		&models.StockLot{},
		&models.StockPackageType{},
		&models.StockQuantPackage{},
		&models.StockQuant{},
		&models.StockPicking{},
		&models.StockMoveLine{},
	)
	if err != nil {
		log.Fatalf("❌ Migration failed: %v", err)
	}
	fmt.Println("✅ Migrations complete")
	fmt.Println()

	// Check if data already exists
	var productCount int64
	db.Model(&models.ProductProduct{}).Count(&productCount)
	if productCount > 0 {
		fmt.Printf("⚠️  Database already has %d products. Clear it first? (y/N): ", productCount)
		var answer string
		fmt.Scanln(&answer)
		if answer != "y" && answer != "Y" {
			fmt.Println("❌ Aborted. Database not modified.")
			return
		}

		// Clear existing data
		fmt.Println("🗑️  Clearing existing data...")
		db.Exec("TRUNCATE TABLE stock_move_line CASCADE")
		db.Exec("TRUNCATE TABLE stock_picking CASCADE")
		db.Exec("TRUNCATE TABLE stock_quant CASCADE")
		db.Exec("TRUNCATE TABLE stock_quant_package CASCADE")
		db.Exec("TRUNCATE TABLE stock_lot CASCADE")
		db.Exec("TRUNCATE TABLE product_product CASCADE")
		db.Exec("TRUNCATE TABLE stock_location CASCADE")
		fmt.Println("✅ Data cleared")
	}

	fmt.Println()
	fmt.Println("📦 Creating demo data...")
	fmt.Println()

	// 1. Create Locations
	fmt.Println("📍 Creating locations...")
	locations := []models.StockLocation{
		{ID: 1, Name: "Physical Locations", CompleteName: "Physical Locations", Usage: "view", LocationID: nil},
		{ID: 8, Name: "WH", CompleteName: "Physical Locations / WH", Usage: "view", LocationID: intPtr(1)},
		{ID: 12, Name: "Stock", CompleteName: "Physical Locations / WH / Stock", Usage: "internal", LocationID: intPtr(8)},
		{ID: 13, Name: "Shelf A", CompleteName: "Physical Locations / WH / Stock / Shelf A", Usage: "internal", LocationID: intPtr(12)},
		{ID: 14, Name: "Shelf B", CompleteName: "Physical Locations / WH / Stock / Shelf B", Usage: "internal", LocationID: intPtr(12)},
		{ID: 15, Name: "Output", CompleteName: "Physical Locations / WH / Output", Usage: "internal", LocationID: intPtr(8)},
		{ID: 5, Name: "Partners", CompleteName: "Virtual Locations / Partners", Usage: "view", LocationID: nil},
		{ID: 9, Name: "Customers", CompleteName: "Virtual Locations / Partners / Customers", Usage: "customer", LocationID: intPtr(5)},
		{ID: 10, Name: "Vendors", CompleteName: "Virtual Locations / Partners / Vendors", Usage: "supplier", LocationID: intPtr(5)},
	}

	for _, loc := range locations {
		if err := db.Create(&loc).Error; err != nil {
			log.Printf("⚠️  Failed to create location %s: %v", loc.Name, err)
		} else {
			fmt.Printf("   ✓ Created location: %s\n", loc.CompleteName)
		}
	}
	fmt.Printf("✅ Created %d locations\n\n", len(locations))

	// 2. Create Products
	fmt.Println("📦 Creating products...")
	now := time.Now()
	products := []models.ProductProduct{
		{
			ID:          1,
			Name:        "InBody 270 Body Composition Analyzer",
			DefaultCode: "IB270",
			Barcode:     "8809509831101",
			Active:      true,
			Type:        "product",
			LastSyncedAt: now,
		},
		{
			ID:          2,
			Name:        "InBody 570 Professional Scanner",
			DefaultCode: "IB570",
			Barcode:     "8809509831202",
			Active:      true,
			Type:        "product",
			LastSyncedAt: now,
		},
		{
			ID:          3,
			Name:        "InBody 770 Clinical Model",
			DefaultCode: "IB770",
			Barcode:     "8809509831303",
			Active:      true,
			Type:        "product",
			LastSyncedAt: now,
		},
		{
			ID:          4,
			Name:        "InBody S10 Body Water Analyzer",
			DefaultCode: "IBS10",
			Barcode:     "8809509832001",
			Active:      true,
			Type:        "product",
			LastSyncedAt: now,
		},
		{
			ID:          5,
			Name:        "InBody Electrode Replacement Set",
			DefaultCode: "IB-ELEC-SET",
			Barcode:     "8809509850001",
			Active:      true,
			Type:        "product",
			LastSyncedAt: now,
		},
		{
			ID:          6,
			Name:        "InBody Thermal Paper Roll (10 pack)",
			DefaultCode: "IB-PAPER-10",
			Barcode:     "8809509850102",
			Active:      true,
			Type:        "product",
			LastSyncedAt: now,
		},
	}

	for _, p := range products {
		if err := db.Create(&p).Error; err != nil {
			log.Printf("⚠️  Failed to create product %s: %v", p.Name, err)
		} else {
			fmt.Printf("   ✓ Created product: [%s] %s\n", p.DefaultCode, p.Name)
		}
	}
	fmt.Printf("✅ Created %d products\n\n", len(products))

	// 3. Create Stock Lots
	fmt.Println("🏷️  Creating lots (serial numbers)...")
	lots := []models.StockLot{
		{ID: 1, Name: "IB270-SN-20240101", ProductID: 1, Ref: "SN-001"},
		{ID: 2, Name: "IB270-SN-20240102", ProductID: 1, Ref: "SN-002"},
		{ID: 3, Name: "IB570-SN-20240201", ProductID: 2, Ref: "SN-003"},
		{ID: 4, Name: "IB770-SN-20240301", ProductID: 3, Ref: "SN-004"},
		{ID: 5, Name: "IBS10-SN-20240401", ProductID: 4, Ref: "SN-005"},
	}

	for _, lot := range lots {
		if err := db.Create(&lot).Error; err != nil {
			log.Printf("⚠️  Failed to create lot %s: %v", lot.Name, err)
		} else {
			fmt.Printf("   ✓ Created lot: %s\n", lot.Name)
		}
	}
	fmt.Printf("✅ Created %d lots\n\n", len(lots))

	// 4. Create Stock Quants (inventory)
	fmt.Println("📊 Creating stock quants (inventory)...")
	quants := []models.StockQuant{
		{ID: 1, ProductID: 1, LocationID: 13, Quantity: 5.0, LotID: int64Ptr(1)},
		{ID: 2, ProductID: 1, LocationID: 13, Quantity: 3.0, LotID: int64Ptr(2)},
		{ID: 3, ProductID: 2, LocationID: 13, Quantity: 2.0, LotID: int64Ptr(3)},
		{ID: 4, ProductID: 3, LocationID: 14, Quantity: 1.0, LotID: int64Ptr(4)},
		{ID: 5, ProductID: 4, LocationID: 14, Quantity: 4.0, LotID: int64Ptr(5)},
		{ID: 6, ProductID: 5, LocationID: 13, Quantity: 50.0, LotID: nil}, // Consumable - no lot
		{ID: 7, ProductID: 6, LocationID: 13, Quantity: 100.0, LotID: nil}, // Consumable - no lot
	}

	for _, q := range quants {
		if err := db.Create(&q).Error; err != nil {
			log.Printf("⚠️  Failed to create quant: %v", err)
		} else {
			fmt.Printf("   ✓ Created quant: Product %d, Qty %.1f @ Location %d\n", q.ProductID, q.Quantity, q.LocationID)
		}
	}
	fmt.Printf("✅ Created %d quants\n\n", len(quants))

	// 5. Create Stock Pickings (orders)
	fmt.Println("📋 Creating pickings (transfer orders)...")
	scheduledDate1 := time.Now().Add(24 * time.Hour)
	scheduledDate2 := time.Now().Add(48 * time.Hour)
	scheduledDate3 := time.Now().Add(-24 * time.Hour) // Past order

	pickings := []models.StockPicking{
		{
			ID:             1,
			Name:           "WH/OUT/00001",
			State:          "assigned",
			LocationID:     12,  // Stock
			LocationDestID: 9,   // Customer
			ScheduledDate:  scheduledDate1,
			Origin:         "SO001",
			Priority:       "1",
			PartnerID:      int64Ptr(15),
		},
		{
			ID:             2,
			Name:           "WH/OUT/00002",
			State:          "assigned",
			LocationID:     12,  // Stock
			LocationDestID: 9,   // Customer
			ScheduledDate:  scheduledDate2,
			Origin:         "SO002",
			Priority:       "0",
			PartnerID:      int64Ptr(16),
		},
		{
			ID:             3,
			Name:           "WH/OUT/00003",
			State:          "done",
			LocationID:     12,  // Stock
			LocationDestID: 9,   // Customer
			ScheduledDate:  scheduledDate3,
			Origin:         "SO003",
			Priority:       "0",
			PartnerID:      int64Ptr(17),
			DateDone:       &scheduledDate3,
		},
	}

	for _, p := range pickings {
		if err := db.Create(&p).Error; err != nil {
			log.Printf("⚠️  Failed to create picking %s: %v", p.Name, err)
		} else {
			fmt.Printf("   ✓ Created picking: %s [%s] - %s\n", p.Name, p.State, p.Origin)
		}
	}
	fmt.Printf("✅ Created %d pickings\n\n", len(pickings))

	// 6. Create Move Lines
	fmt.Println("📝 Creating move lines...")
	moveLines := []models.StockMoveLine{
		{ID: 1, PickingID: 1, ProductID: 1, QtyDone: 2.0, LocationID: 13, LocationDestID: 9, LotID: int64Ptr(1), State: "assigned"},
		{ID: 2, PickingID: 1, ProductID: 5, QtyDone: 5.0, LocationID: 13, LocationDestID: 9, State: "assigned"},
		{ID: 3, PickingID: 2, ProductID: 2, QtyDone: 1.0, LocationID: 13, LocationDestID: 9, LotID: int64Ptr(3), State: "assigned"},
		{ID: 4, PickingID: 2, ProductID: 6, QtyDone: 10.0, LocationID: 13, LocationDestID: 9, State: "assigned"},
		{ID: 5, PickingID: 3, ProductID: 3, QtyDone: 1.0, LocationID: 14, LocationDestID: 9, LotID: int64Ptr(4), State: "done"},
	}

	for _, ml := range moveLines {
		if err := db.Create(&ml).Error; err != nil {
			log.Printf("⚠️  Failed to create move line: %v", err)
		} else {
			fmt.Printf("   ✓ Created move line: Picking %d, Product %d, Qty %.1f\n", ml.PickingID, ml.ProductID, ml.QtyDone)
		}
	}
	fmt.Printf("✅ Created %d move lines\n\n", len(moveLines))

	// Summary
	fmt.Println()
	fmt.Println("=" + string(make([]rune, 60)))
	fmt.Println("🎉 Demo data created successfully!")
	fmt.Println()
	fmt.Println("📊 Summary:")
	fmt.Printf("   • %d locations (including warehouse structure)\n", len(locations))
	fmt.Printf("   • %d products (InBody devices and accessories)\n", len(products))
	fmt.Printf("   • %d serial numbers\n", len(lots))
	fmt.Printf("   • %d stock records (quants)\n", len(quants))
	fmt.Printf("   • %d transfer orders (pickings)\n", len(pickings))
	fmt.Printf("   • %d move lines\n", len(moveLines))
	fmt.Println()
	fmt.Println("🚀 Run the utility to see the data:")
	fmt.Println("   go run ./cmd/test_sync/main.go")
	fmt.Println()
	fmt.Println("🌐 Or start the server:")
	fmt.Println("   go run ./cmd/api/main.go")
	fmt.Println("   Then visit: http://localhost:3210")
	fmt.Println("=" + string(make([]rune, 60)))
}

func intPtr(i int64) *int64 {
	return &i
}

func int64Ptr(i int64) *int64 {
	return &i
}
