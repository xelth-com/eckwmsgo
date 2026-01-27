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
	fmt.Println("🏗️  Rebuilding Checksums for Sync History...")

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	db, err := database.Connect(cfg.Database)
	if err != nil {
		log.Fatal(err)
	}

	calc := sync.NewChecksumCalculator(cfg.InstanceID)

	// Get last 30 sync_history records
	var history []models.SyncHistory
	db.Order("started_at DESC").Limit(30).Find(&history)
	fmt.Printf("📝 Processing %d sync_history records...\n", len(history))

	created := 0
	for _, h := range history {
		hash, err := calc.ComputeChecksum(h)
		if err != nil {
			log.Printf("Warning: Failed to compute checksum for sync_history %d: %v", h.ID, err)
			continue
		}

		checksum := models.EntityChecksum{
			EntityType:     "sync_history",
			EntityID:       h.GetEntityID(),
			ContentHash:    hash,
			FullHash:       hash,
			LastUpdated:    time.Now().UTC(),
			SourceInstance: cfg.InstanceID,
		}

		// Check if exists
		var existing models.EntityChecksum
		result := db.Where("entity_type = ? AND entity_id = ?", "sync_history", h.GetEntityID()).First(&existing)

		if result.Error != nil {
			db.Create(&checksum)
			created++
		} else {
			db.Model(&existing).Updates(checksum)
		}
	}

	fmt.Printf("✅ Sync History: %d checksums created/updated\n", created)
	fmt.Println("✅ Done! Sync history will now sync between servers.")
}
