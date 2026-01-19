package main

import (
	"encoding/json"
	"fmt"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"github.com/xelth-com/eckwmsgo/internal/models"
)

func main() {
	// Connect directly to running embedded postgres
	// Try eckwmsgo_local database first (where data was seeded)
	dsn := "host=localhost user=postgres password=postgres dbname=eckwmsgo_local port=5433 sslmode=disable client_encoding=UTF8"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		fmt.Printf("❌ Failed to connect: %v\n", err)
		fmt.Println("\n💡 Try starting the server first:")
		fmt.Println("   go run ./cmd/api/main.go")
		os.Exit(1)
	}

	fmt.Println("╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║          📊 eckWMS Standalone Data Report                ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Count stats
	var productCount, locationCount, lotCount, quantCount, pickingCount, moveLineCount int64
	db.Model(&models.ProductProduct{}).Count(&productCount)
	db.Model(&models.StockLocation{}).Count(&locationCount)
	db.Model(&models.StockLot{}).Count(&lotCount)
	db.Model(&models.StockQuant{}).Count(&quantCount)
	db.Model(&models.StockPicking{}).Count(&pickingCount)
	db.Model(&models.StockMoveLine{}).Count(&moveLineCount)

	fmt.Println("📈 DATABASE STATISTICS")
	fmt.Println("──────────────────────────────────────────────────────────")
	fmt.Printf("  Products:      %3d\n", productCount)
	fmt.Printf("  Locations:     %3d\n", locationCount)
	fmt.Printf("  Lots:          %3d\n", lotCount)
	fmt.Printf("  Quants:        %3d\n", quantCount)
	fmt.Printf("  Pickings:      %3d\n", pickingCount)
	fmt.Printf("  Move Lines:    %3d\n", moveLineCount)
	fmt.Println()

	if productCount > 0 {
		// Show products
		var products []models.ProductProduct
		db.Order("id").Find(&products)

		fmt.Println("📦 PRODUCTS")
		fmt.Println("──────────────────────────────────────────────────────────")
		for _, p := range products {
			fmt.Printf("  [%d] %s\n", p.ID, p.Name)
			if p.DefaultCode != "" {
				fmt.Printf("      └─ Code: %s", p.DefaultCode)
				if p.Barcode != "" {
					fmt.Printf(" | Barcode: %s", p.Barcode)
				}
				fmt.Println()
			}
		}
		fmt.Println()

		// Show locations (hierarchical)
		var locations []models.StockLocation
		db.Order("id").Find(&locations)

		fmt.Println("📍 LOCATIONS (Warehouse Structure)")
		fmt.Println("──────────────────────────────────────────────────────────")
		for _, loc := range locations {
			if loc.LocationID == nil {
				printLocationTree(db, loc, 0)
			}
		}
		fmt.Println()

		// Show inventory (quants)
		var quants []models.StockQuant
		db.Order("location_id, product_id").Find(&quants)

		fmt.Println("📊 INVENTORY (Stock Quants)")
		fmt.Println("──────────────────────────────────────────────────────────")
		for _, q := range quants {
			var product models.ProductProduct
			var location models.StockLocation
			db.First(&product, q.ProductID)
			db.First(&location, q.LocationID)

			lotInfo := ""
			if q.LotID != nil {
				var lot models.StockLot
				db.First(&lot, *q.LotID)
				lotInfo = fmt.Sprintf(" [Lot: %s]", lot.Name)
			}

			fmt.Printf("  Location: %s\n", location.Name)
			fmt.Printf("    └─ %s x %.1f%s\n", product.DefaultCode, q.Quantity, lotInfo)
		}
		fmt.Println()

		// Show pickings with details
		var pickings []models.StockPicking
		db.Order("scheduled_date DESC").Find(&pickings)

		if len(pickings) > 0 {
			fmt.Println("📋 TRANSFER ORDERS (Pickings)")
			fmt.Println("──────────────────────────────────────────────────────────")
			for _, p := range pickings {
				statusIcon := "📦"
				if p.State == "done" {
					statusIcon = "✅"
				} else if p.State == "assigned" {
					statusIcon = "🎯"
				}

				fmt.Printf("  %s %s [%s]\n", statusIcon, p.Name, p.State)
				fmt.Printf("      Origin: %s | Scheduled: %s\n", p.Origin, p.ScheduledDate.Format("2006-01-02"))

				// Get move lines
				var moveLines []models.StockMoveLine
				db.Where("picking_id = ?", p.ID).Find(&moveLines)

				if len(moveLines) > 0 {
					fmt.Println("      Items:")
					for _, ml := range moveLines {
						var product models.ProductProduct
						db.First(&product, ml.ProductID)

						lotInfo := ""
						if ml.LotID != nil {
							var lot models.StockLot
							db.First(&lot, *ml.LotID)
							lotInfo = fmt.Sprintf(" [%s]", lot.Name)
						}

						fmt.Printf("        • %s x %.0f%s\n", product.DefaultCode, ml.QtyDone, lotInfo)
					}
				}
				fmt.Println()
			}
		}
	}

	// JSON export
	if len(os.Args) > 1 && os.Args[1] == "--json" {
		data := map[string]interface{}{
			"stats": map[string]int64{
				"products":   productCount,
				"locations":  locationCount,
				"lots":       lotCount,
				"quants":     quantCount,
				"pickings":   pickingCount,
				"move_lines": moveLineCount,
			},
		}
		jsonData, _ := json.MarshalIndent(data, "", "  ")
		fmt.Println("\n📄 JSON EXPORT:")
		fmt.Println(string(jsonData))
	}

	fmt.Println("══════════════════════════════════════════════════════════")
	fmt.Println("✨ eckWMS is working standalone without Odoo!")
	fmt.Println()
	fmt.Println("🚀 Start the web server:")
	fmt.Println("   go run ./cmd/api/main.go")
	fmt.Println("   Then visit: http://localhost:3210")
	fmt.Println("══════════════════════════════════════════════════════════")
}

func printLocationTree(db *gorm.DB, loc models.StockLocation, level int) {
	indent := ""
	for i := 0; i < level; i++ {
		indent += "  "
	}

	icon := "📁"
	if loc.Usage == "internal" {
		icon = "📦"
	} else if loc.Usage == "customer" {
		icon = "👤"
	} else if loc.Usage == "supplier" {
		icon = "🏭"
	}

	fmt.Printf("%s%s %s [%s]\n", indent, icon, loc.Name, loc.Usage)

	// Find children
	var children []models.StockLocation
	db.Where("location_id = ?", loc.ID).Order("name").Find(&children)
	for _, child := range children {
		printLocationTree(db, child, level+1)
	}
}
