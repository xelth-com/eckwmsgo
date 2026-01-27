package main

import (
	"fmt"
	"log"

	"github.com/xelth-com/eckwmsgo/internal/config"
	"github.com/xelth-com/eckwmsgo/internal/database"
)

func main() {
	fmt.Println("🔄 Force Reset Sync Metadata for Shipments...")

	// 1. Init
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	db, err := database.Connect(cfg.Database)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("📍 Current Instance: %s\n", cfg.InstanceID)

	// 2. Show current state
	var currentMeta []struct {
		InstanceID  string
		EntityType  string
		LastSyncAt  string
	}
	db.Raw("SELECT instance_id, entity_type, last_sync_at FROM sync_metadata WHERE entity_type LIKE '%shipment%' OR entity_type LIKE '%tracking%'").Scan(&currentMeta)

	if len(currentMeta) > 0 {
		fmt.Println("\n📊 Current Sync Metadata:")
		for _, m := range currentMeta {
			fmt.Printf("  %s -> %s: %s\n", m.InstanceID, m.EntityType, m.LastSyncAt)
		}
	} else {
		fmt.Println("\n📊 No sync metadata found for shipments/tracking")
	}

	// 3. Reset Timestamp for 'shipments_push' and 'tracking_push'
	// This tells the Mesh Sync engine: "You have never synced before"
	// So it will select ALL records from the beginning of time.
	fmt.Println("\n🔄 Resetting to epoch (2000-01-01)...")

	result := db.Exec(`
		UPDATE sync_metadata
		SET last_sync_at = '2000-01-01 00:00:00'
		WHERE entity_type IN ('shipments_push', 'tracking_push')
	`)

	if result.Error != nil {
		log.Fatalf("❌ Error: %v", result.Error)
	}

	if result.RowsAffected == 0 {
		fmt.Println("⚠️  No existing metadata found. Will be created on first sync.")
	} else {
		fmt.Printf("✅ Reset %d metadata records\n", result.RowsAffected)
	}

	// 4. Show new state
	db.Raw("SELECT instance_id, entity_type, last_sync_at FROM sync_metadata WHERE entity_type LIKE '%shipment%' OR entity_type LIKE '%tracking%'").Scan(&currentMeta)

	if len(currentMeta) > 0 {
		fmt.Println("\n📊 Updated Sync Metadata:")
		for _, m := range currentMeta {
			fmt.Printf("  %s -> %s: %s\n", m.InstanceID, m.EntityType, m.LastSyncAt)
		}
	}

	fmt.Println("\n✅ Done! Full push will happen on next Mesh Sync cycle.")
	fmt.Println("💡 Tip: Restart the server or wait for next auto-sync")
}
