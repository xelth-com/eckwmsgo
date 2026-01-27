package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/xelth-com/eckwmsgo/internal/config"
	"github.com/xelth-com/eckwmsgo/internal/database"
	"github.com/xelth-com/eckwmsgo/internal/models"
	deliveryService "github.com/xelth-com/eckwmsgo/internal/services/delivery"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	db, err := database.Connect(cfg.Database)
	if err != nil {
		log.Fatal(err)
	}

	// Create service
	service := deliveryService.NewService(db, cfg)

	// Get shipments like API does
	shipments, err := service.ListShipments("", 50)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	fmt.Printf("=== API would return %d shipments ===\n\n", len(shipments))

	if len(shipments) > 0 {
		// Show first shipment as JSON
		jsonData, err := json.MarshalIndent(shipments[0], "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("First shipment JSON:")
		fmt.Println(string(jsonData))
	}

	// Check sync history
	var syncHistory []models.SyncHistory
	result := db.Order("started_at DESC").Limit(5).Find(&syncHistory)
	if result.Error != nil {
		log.Printf("Error: %v", result.Error)
	} else {
		fmt.Printf("\n=== Sync History: %d records ===\n", len(syncHistory))
		if len(syncHistory) > 0 {
			jsonData, _ := json.MarshalIndent(syncHistory[0], "", "  ")
			fmt.Println("First sync history JSON:")
			fmt.Println(string(jsonData))
		}
	}
}
