package main

import (
	"fmt"
	"log"
	"time"

	"github.com/xelth-com/eckwmsgo/internal/config"
	"github.com/xelth-com/eckwmsgo/internal/database"
	"github.com/xelth-com/eckwmsgo/internal/models"
	"github.com/xelth-com/eckwmsgo/internal/sync"
)

func main() {
	fmt.Println("Testing Shipment Synchronization Logic...")

	// 1. Init DB
	cfg, _ := config.Load()
	db, err := database.Connect(cfg.Database)
	if err != nil {
		log.Fatalf("DB Error: %v", err)
	}

	// 2. Create Dummy Data (Simulate Scraper)
	testTracking := fmt.Sprintf("TEST-SYNC-%d", time.Now().Unix())
	shipment := models.StockPickingDelivery{
		TrackingNumber: testTracking,
		Status:         "delivered",
		CarrierID:      nil,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := db.Create(&shipment).Error; err != nil {
		log.Fatalf("Failed to create dummy shipment: %v", err)
	}
	fmt.Printf("[OK] Created dummy shipment: %s (ID: %d)\n", testTracking, shipment.ID)

	// 3. Init Sync Engine (Partial)
	// We only need the logic part, not the full network listener
	syncCfg := config.LoadSyncConfig()
	meshCfg := &sync.MeshConfig{
		InstanceID: "test-node",
		MeshSecret: "secret",
		NodeRole:   "peer",
	}
	engine := sync.NewSyncEngine(db, syncCfg, meshCfg)

	// 4. Simulate Pull Request from a Peer
	// We ask for data changed in the last 1 hour (our new record should appear)
	oneHourAgo := time.Now().Add(-1 * time.Hour)
	req := &sync.MeshSyncRequest{
		EntityTypes: []string{"shipments"},
		Since:       &oneHourAgo,
	}

	fmt.Println("[...] Simulating Mesh Pull Request...")
	resp, err := engine.GetDataForPull(req)
	if err != nil {
		log.Fatalf("Sync Engine Error: %v", err)
	}

	// 5. Verify Result
	found := false
	for _, s := range resp.Shipments {
		if s.TrackingNumber == testTracking {
			found = true
			fmt.Printf("[OK] FOUND shipment in sync packet!\n")
			fmt.Printf("     - Status: %s\n", s.Status)
			fmt.Printf("     - UpdatedAt: %v\n", s.UpdatedAt)
			break
		}
	}

	if !found {
		// Cleanup before failing
		db.Unscoped().Delete(&shipment)
		log.Fatalf("[FAIL] Created shipment was NOT found in sync response.")
	}

	// Cleanup
	db.Unscoped().Delete(&shipment)
	fmt.Println("\n[SUCCESS] Shipment Sync Logic is VERIFIED working.")
}
