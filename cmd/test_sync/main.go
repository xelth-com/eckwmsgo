package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/xelth-com/eckwmsgo/internal/config"
	"github.com/xelth-com/eckwmsgo/internal/database"
	"github.com/xelth-com/eckwmsgo/internal/models"
)

func main() {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to database
	db, err := database.Connect(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	fmt.Println("=" + string(make([]byte, 80)) + "=")
	fmt.Println("📊 eckWMS Database Status Report")
	fmt.Println("=" + string(make([]byte, 80)) + "=")
	fmt.Println()

	// Check Odoo Configuration
	fmt.Println("🔧 Odoo Configuration:")
	if cfg.Odoo.URL != "" {
		fmt.Printf("   URL: %s\n", cfg.Odoo.URL)
		fmt.Printf("   Database: %s\n", cfg.Odoo.Database)
		fmt.Printf("   Username: %s\n", cfg.Odoo.Username)
		fmt.Printf("   Sync Interval: %d minutes\n", cfg.Odoo.SyncInterval)
	} else {
		fmt.Println("   ⚠️  Odoo NOT configured (check .env file)")
		fmt.Println("   Add these variables to .env:")
		fmt.Println("   - ODOO_URL=https://your-odoo-instance.com")
		fmt.Println("   - ODOO_DB=your_database")
		fmt.Println("   - ODOO_USER=your_username")
		fmt.Println("   - ODOO_PASSWORD=your_password")
		fmt.Println("   - ODOO_SYNC_INTERVAL=15")
	}
	fmt.Println()

	// Count records
	var productCount, locationCount, lotCount, packageCount, quantCount int64
	var pickingCount, moveLineCount int64

	db.Model(&models.ProductProduct{}).Count(&productCount)
	db.Model(&models.StockLocation{}).Count(&locationCount)
	db.Model(&models.StockLot{}).Count(&lotCount)
	db.Model(&models.StockQuantPackage{}).Count(&packageCount)
	db.Model(&models.StockQuant{}).Count(&quantCount)
	db.Model(&models.StockPicking{}).Count(&pickingCount)
	db.Model(&models.StockMoveLine{}).Count(&moveLineCount)

	fmt.Println("📦 Database Statistics:")
	fmt.Printf("   Products:     %d\n", productCount)
	fmt.Printf("   Locations:    %d\n", locationCount)
	fmt.Printf("   Lots:         %d\n", lotCount)
	fmt.Printf("   Packages:     %d\n", packageCount)
	fmt.Printf("   Quants:       %d\n", quantCount)
	fmt.Printf("   Pickings:     %d\n", pickingCount)
	fmt.Printf("   Move Lines:   %d\n", moveLineCount)
	fmt.Println()

	// Show sample products
	if productCount > 0 {
		var products []models.ProductProduct
		db.Order("id ASC").Limit(5).Find(&products)

		fmt.Println("📋 Sample Products (first 5):")
		for _, p := range products {
			fmt.Printf("   [%d] %s\n", p.ID, p.Name)
			if p.DefaultCode != "" {
				fmt.Printf("       Reference: %s\n", p.DefaultCode)
			}
			if p.Barcode != "" {
				fmt.Printf("       Barcode: %s\n", p.Barcode)
			}
		}
		fmt.Println()
	}

	// Show sample locations
	if locationCount > 0 {
		var locations []models.StockLocation
		db.Order("id ASC").Limit(5).Find(&locations)

		fmt.Println("📍 Sample Locations (first 5):")
		for _, loc := range locations {
			fmt.Printf("   [%d] %s\n", loc.ID, loc.Name)
			if loc.CompleteName != "" {
				fmt.Printf("       Path: %s\n", loc.CompleteName)
			}
		}
		fmt.Println()
	}

	// Show pickings
	if pickingCount > 0 {
		var pickings []models.StockPicking
		db.Order("scheduled_date DESC").Limit(10).Find(&pickings)

		fmt.Println("📋 Recent Pickings (last 10):")
		for _, p := range pickings {
			fmt.Printf("   [%d] %s - State: %s\n", p.ID, p.Name, p.State)
			fmt.Printf("       Scheduled: %s\n", p.ScheduledDate.Format("2006-01-02 15:04"))
			if p.Origin != "" {
				fmt.Printf("       Origin: %s\n", p.Origin)
			}

			// Get move lines for this picking
			var moveLines []models.StockMoveLine
			db.Where("picking_id = ?", p.ID).Limit(3).Find(&moveLines)
			if len(moveLines) > 0 {
				fmt.Printf("       Move Lines: %d (showing first 3)\n", len(moveLines))
				for _, ml := range moveLines {
					fmt.Printf("         - Product ID: %d, Qty: %.2f, State: %s\n",
						ml.ProductID, ml.QtyDone, ml.State)
				}
			}
			fmt.Println()
		}
	}

	// Show summary
	totalRecords := productCount + locationCount + lotCount + packageCount + quantCount + pickingCount + moveLineCount
	fmt.Println("=" + string(make([]byte, 80)) + "=")
	fmt.Printf("📊 Total Records in Database: %d\n", totalRecords)

	if totalRecords == 0 {
		fmt.Println()
		fmt.Println("⚠️  Database is empty!")
		fmt.Println()
		fmt.Println("To sync data from Odoo:")
		fmt.Println("1. Configure Odoo settings in .env (see above)")
		fmt.Println("2. Start the server: ./eckwms")
		fmt.Println("3. Or trigger manual sync via API:")
		fmt.Println("   curl -X POST http://localhost:3210/api/odoo/sync/trigger \\")
		fmt.Println("     -H \"Authorization: Bearer YOUR_TOKEN\"")
	}

	fmt.Println("=" + string(make([]byte, 80)) + "=")

	// Export to JSON if requested
	if len(os.Args) > 1 && os.Args[1] == "--json" {
		data := map[string]interface{}{
			"stats": map[string]int64{
				"products":   productCount,
				"locations":  locationCount,
				"lots":       lotCount,
				"packages":   packageCount,
				"quants":     quantCount,
				"pickings":   pickingCount,
				"move_lines": moveLineCount,
				"total":      totalRecords,
			},
			"odoo_configured": cfg.Odoo.URL != "",
		}

		jsonData, _ := json.MarshalIndent(data, "", "  ")
		fmt.Println()
		fmt.Println("JSON Output:")
		fmt.Println(string(jsonData))
	}
}
