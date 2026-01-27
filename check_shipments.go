package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/xelth-com/eckwmsgo/internal/config"
	"github.com/xelth-com/eckwmsgo/internal/database"
	"github.com/xelth-com/eckwmsgo/internal/models"
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

	// Check shipments
	var shipments []models.StockPickingDelivery
	result := db.Order("COALESCE(last_activity_at, created_at) DESC").Limit(5).Find(&shipments)
	if result.Error != nil {
		log.Fatalf("Error querying shipments: %v", result.Error)
	}

	fmt.Printf("\n=== Found %d shipments in DB ===\n\n", len(shipments))

	for _, s := range shipments {
		fmt.Printf("ID: %d\n", s.ID)
		fmt.Printf("Tracking: %s\n", s.TrackingNumber)
		fmt.Printf("Status: %s\n", s.Status)
		fmt.Printf("PickingID: %v\n", s.PickingID)
		fmt.Printf("CarrierID: %v\n", s.CarrierID)
		fmt.Printf("CreatedAt: %s\n", s.CreatedAt.Format("2006-01-02 15:04:05"))

		// Try to parse rawResponse
		if s.RawResponse != "" {
			var raw map[string]interface{}
			if err := json.Unmarshal([]byte(s.RawResponse), &raw); err == nil {
				fmt.Printf("RawResponse keys: %v\n", getKeys(raw))
			} else {
				fmt.Printf("RawResponse: [Invalid JSON: %v]\n", err)
			}
		} else {
			fmt.Printf("RawResponse: [empty]\n")
		}

		fmt.Println("---")
	}

	// Check sync history
	var syncHistory []models.SyncHistory
	result = db.Order("started_at DESC").Limit(5).Find(&syncHistory)
	if result.Error != nil {
		log.Printf("Error querying sync history: %v", result.Error)
	} else {
		fmt.Printf("\n=== Found %d sync history records ===\n\n", len(syncHistory))
		for _, h := range syncHistory {
			fmt.Printf("ID: %d, Provider: %s, Status: %s, Created: %d, Updated: %d, Time: %s\n",
				h.ID, h.Provider, h.Status, h.Created, h.Updated, h.StartedAt.Format("2006-01-02 15:04:05"))
		}
	}

	// Count total shipments
	var count int64
	db.Model(&models.StockPickingDelivery{}).Count(&count)
	fmt.Printf("\n=== Total shipments in DB: %d ===\n", count)
}

func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
