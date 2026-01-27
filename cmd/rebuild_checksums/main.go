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
	fmt.Println("🏗️  Rebuilding Checksum Tree for Shipments...")

	// 1. Load Config & DB
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	db, err := database.Connect(cfg.Database)
	if err != nil {
		log.Fatal(err)
	}

	calc := sync.NewChecksumCalculator(cfg.InstanceID)

	// 2. Rebuild Shipments
	var shipments []models.StockPickingDelivery
	db.Find(&shipments)
	fmt.Printf("📦 Processing %d shipments...\n", len(shipments))

	created := 0
	updated := 0
	for _, s := range shipments {
		hash, err := calc.ComputeChecksum(s)
		if err != nil {
			log.Printf("Warning: Failed to compute checksum for shipment %d: %v", s.ID, err)
			continue
		}

		checksum := models.EntityChecksum{
			EntityType:     "shipment",
			EntityID:       s.GetEntityID(),
			ContentHash:    hash,
			FullHash:       hash,
			LastUpdated:    time.Now().UTC(),
			SourceInstance: cfg.InstanceID,
		}

		// Check if exists
		var existing models.EntityChecksum
		result := db.Where("entity_type = ? AND entity_id = ?", "shipment", s.GetEntityID()).First(&existing)

		if result.Error != nil {
			// Create new
			db.Create(&checksum)
			created++
		} else {
			// Update existing
			db.Model(&existing).Updates(checksum)
			updated++
		}
	}
	fmt.Printf("✅ Shipments: %d created, %d updated\n", created, updated)

	// 3. Rebuild Tracking
	var tracking []models.DeliveryTracking
	db.Find(&tracking)
	fmt.Printf("📝 Processing %d tracking records...\n", len(tracking))

	createdT := 0
	updatedT := 0
	for _, t := range tracking {
		hash, err := calc.ComputeChecksum(t)
		if err != nil {
			log.Printf("Warning: Failed to compute checksum for tracking %d: %v", t.ID, err)
			continue
		}

		checksum := models.EntityChecksum{
			EntityType:     "tracking",
			EntityID:       t.GetEntityID(),
			ContentHash:    hash,
			FullHash:       hash,
			LastUpdated:    time.Now().UTC(),
			SourceInstance: cfg.InstanceID,
		}

		// Check if exists
		var existing models.EntityChecksum
		result := db.Where("entity_type = ? AND entity_id = ?", "tracking", t.GetEntityID()).First(&existing)

		if result.Error != nil {
			// Create new
			db.Create(&checksum)
			createdT++
		} else {
			// Update existing
			db.Model(&existing).Updates(checksum)
			updatedT++
		}
	}
	fmt.Printf("✅ Tracking: %d created, %d updated\n", createdT, updatedT)

	fmt.Println("\n✅ Checksum tree rebuilt. Sync engine will now use Merkle-like comparison.")
	fmt.Printf("📊 Total checksums: %d shipments + %d tracking = %d\n",
		created+updated, createdT+updatedT, created+updated+createdT+updatedT)
}
