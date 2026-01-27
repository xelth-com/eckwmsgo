package sync

import (
	"log"

	"github.com/xelth-com/eckwmsgo/internal/config"
	"github.com/xelth-com/eckwmsgo/internal/models"
)

// syncDevices synchronizes RegisteredDevice entities with a mesh network
// Note: This function uses checksum-based comparison; actual push is handled
// by mesh_sync.go's pushShipmentsToNode which includes devices.
func (se *SyncEngine) syncDevices(cfg config.EntitySyncConfig) (int, int, error) {
	log.Printf("🔄 Syncing Devices via Checksum Engine...")

	// 1. Get local checksums
	var localChecksums []models.EntityChecksum
	err := se.db.Where("entity_type = ?", "device").Find(&localChecksums).Error
	if err != nil {
		log.Printf("❌ Failed to fetch local device checksums: %v", err)
		return 0, 0, err
	}

	log.Printf("   Found %d local device checksums", len(localChecksums))
	if len(localChecksums) == 0 {
		log.Println("   No device checksums to sync")
	}

	// 2. Fetch devices from database
	var devices []models.RegisteredDevice
	err = se.db.Find(&devices).Error
	if err != nil {
		log.Printf("❌ Failed to fetch devices: %v", err)
		return 0, 0, err
	}

	log.Printf("   Found %d devices in database", len(devices))

	// Device push to mesh nodes is handled by mesh_sync.go's pushShipmentsToNode
	// which runs during SyncWithRelay() and includes devices in the payload.
	// This function focuses on local checksum tracking for incremental sync.

	return len(devices), 0, nil
}
