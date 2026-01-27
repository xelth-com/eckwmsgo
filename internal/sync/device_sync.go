package sync

import (
	"fmt"
	"log"
	"time"

	"github.com/xelth-com/eckwmsgo/internal/config"
	"github.com/xelth-com/eckwmsgo/internal/database"
	"github.com/xelth-com/eckwmsgo/internal/models"
)

// syncDevices synchronizes RegisteredDevice entities with a mesh network
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
		log.Println("   No devices to sync")
		return 0, 0, nil
	}

	// 2. Fetch devices from database
	var devices []models.RegisteredDevice
	err = se.db.Find(&devices).Error
	if err != nil {
		log.Printf("❌ Failed to fetch devices: %v", err)
		return 0, 0, err
	}

	log.Printf("   Found %d devices in database", len(devices))

	// 3. Push devices to bootstrap nodes
	pushedCount := 0
	for _, device := range devices {
		// Skip devices with empty device_id (deleted or invalid)
		if device.DeviceID == "" {
			continue
		}

		// For each bootstrap node (route), attempt to push
		for _, route := range se.connectionManager.routes {
			// Get node ID from route
			nodeID := route.URL // Use route.URL as node_id

			// Skip offline routes
			routeStatus := se.connectionManager.GetRouteStatus(route.URL)
			if routeStatus == nil || !routeStatus.IsAvailable {
				log.Printf("   Skipping offline route: %s", route.URL)
				continue
			}

			// Push to this node
			log.Printf("📤 Pushing device %s to %s", device.DeviceID, route.URL)
			
			// Prepare push data
			pushData := map[string]interface{}{
				"entity_type": "device",
				"entity_id":   device.DeviceID,
				"operation":    "upsert",
				"data": map[string]interface{}{
					"device_id":      device.DeviceID,
					"device_name":    device.Name,
					"public_key":     device.PublicKey,
					"status":        string(device.Status),
					"last_seen_at":   device.LastSeenAt.Format(time.RFC3339),
					"created_at":     device.CreatedAt.Format(time.RFC3339),
					"updated_at":     device.UpdatedAt.Format(time.RFC3339),
				},
				"node_id": nodeID,
			}

			// Marshal to JSON
			jsonData, err := json.Marshal(pushData)
			if err != nil {
				log.Printf("   ❌ Failed to marshal push data: %v", err)
				continue
			}

			// Send POST to /api/mesh/push
			pushURL := route.URL + "/api/mesh/push"
			err := se.httpClient.Post(pushURL, "application/json", bytes.NewBuffer(jsonData))
			if err != nil {
				log.Printf("   ❌ Failed to push device %s to %s", device.DeviceID, pushURL, err)
				continue
			}

			if err.StatusCode >= 400 {
				log.Printf("   ❌ Push failed (HTTP %d): %s", err.StatusCode, pushURL)
				continue
			}

			pushedCount++
			log.Printf("   ✅ Pushed device %s successfully to %s", device.DeviceID, pushURL)
	}

	log.Printf("✅ Device sync completed: pushed %d/%d devices", pushedCount, len(devices))
	return len(devices), 0, nil
}
