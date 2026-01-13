package sync

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"time"

	"github.com/dmytrosurovtsev/eckwmsgo/internal/models"
)

// SyncWithRelay initiates a full push/pull cycle with configured relays
func (se *SyncEngine) SyncWithRelay() error {
	// Only Peers and Edges should sync with Relay
	if se.GetRole() != RolePeer && se.GetRole() != RoleEdge {
		return fmt.Errorf("current role %s cannot sync with relay (requires private key)", se.GetRole())
	}

	totalRecords := 0

	// 1. PUSH Changes
	if err := se.PushToRelay(); err != nil {
		log.Printf("Relay Push error: %v", err)
		se.updateRelaySyncMetadata(false, 0)
		return err
	}

	// 2. PULL Changes
	if err := se.PullFromRelay(); err != nil {
		log.Printf("Relay Pull error: %v", err)
		se.updateRelaySyncMetadata(false, 0)
		return err
	}

	se.updateRelaySyncMetadata(true, totalRecords)
	return nil
}

// PushToRelay finds local changes, encrypts them, and sends to relay
func (se *SyncEngine) PushToRelay() error {
	packets := make([]models.EncryptedSyncPacket, 0)

	// Get last sync timestamp or default to 1 hour
	lastSync := se.getLastRelaySyncTime()
	cutoff := time.Now().Add(-1 * time.Hour)
	if lastSync.After(time.Time{}) {
		cutoff = lastSync
	}

	// --- 1. Collect Items ---
	var items []models.Item
	query := se.db.DB.Where("updated_at > ?", cutoff)
	if err := query.Find(&items).Error; err != nil {
		return fmt.Errorf("failed to query items: %w", err)
	}

	for _, item := range items {
		meta := NewEntityMetadata(EntityTypeItem, fmt.Sprintf("%d", item.ID), se.instanceID)
		meta.Version = item.SyncVersion
		meta.VectorClock = parseVectorClock(item.VectorClock)

		packet, err := se.securityLayer.EncryptPacket(meta, item)
		if err != nil {
			log.Printf("Error encrypting item %d: %v", item.ID, err)
			continue
		}
		packets = append(packets, *packet)
	}

	// --- 2. Collect Orders ---
	var orders []models.Order
	se.db.Where("updated_at > ?", cutoff).Find(&orders)

	for _, order := range orders {
		meta := NewEntityMetadata(EntityTypeOrder, fmt.Sprintf("%d", order.ID), se.instanceID)
		// Orders don't have SyncVersion in current model, use default

		packet, err := se.securityLayer.EncryptPacket(meta, order)
		if err != nil {
			log.Printf("Error encrypting order %d: %v", order.ID, err)
			continue
		}
		packets = append(packets, *packet)
	}

	if len(packets) == 0 {
		return nil
	}

	// --- 3. Send to Relay ---
	relayURL := se.getRelayURL()
	if relayURL == "" {
		return fmt.Errorf("no relay route configured")
	}

	log.Printf("Pushing %d encrypted packets to relay %s", len(packets), relayURL)
	return se.sendPackets(relayURL, packets)
}

// PullFromRelay fetches encrypted packets, decrypts them, and applies to DB
func (se *SyncEngine) PullFromRelay() error {
	relayURL := se.getRelayURL()
	if relayURL == "" {
		return fmt.Errorf("no relay route configured")
	}

	// Get last sync version for incremental sync
	lastVersion := se.getLastRelayVersion()

	// Build pull request
	pullReq := map[string]interface{}{
		"since_version": lastVersion,
		"instance_id":   se.instanceID,
	}
	reqData, _ := json.Marshal(pullReq)

	resp, err := http.Post(relayURL+"/api/sync/pull", "application/json", bytes.NewBuffer(reqData))
	if err != nil {
		return fmt.Errorf("failed to connect to relay: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("relay returned status %d", resp.StatusCode)
	}

	var packets []models.EncryptedSyncPacket
	if err := json.NewDecoder(resp.Body).Decode(&packets); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	log.Printf("Received %d encrypted packets from relay", len(packets))

	// Decrypt and Apply
	successCount := 0
	for _, packet := range packets {
		if err := se.processEncryptedPacket(&packet); err != nil {
			log.Printf("Failed to process packet %s/%s: %v", packet.EntityType, packet.EntityID, err)
		} else {
			successCount++
		}
	}

	log.Printf("Successfully processed %d/%d packets", successCount, len(packets))
	return nil
}

// processEncryptedPacket decrypts and saves a single packet
func (se *SyncEngine) processEncryptedPacket(packet *models.EncryptedSyncPacket) error {
	var target interface{}

	switch EntityType(packet.EntityType) {
	case EntityTypeItem:
		target = &models.Item{}
	case EntityTypeOrder:
		target = &models.Order{}
	default:
		return fmt.Errorf("unknown entity type: %s", packet.EntityType)
	}

	// Decrypt
	if err := se.securityLayer.DecryptPacket(packet, target); err != nil {
		return fmt.Errorf("decryption failed: %w", err)
	}

	// Save to DB using reflection
	val := reflect.ValueOf(target).Elem()
	if err := se.db.DB.Save(val.Addr().Interface()).Error; err != nil {
		return fmt.Errorf("failed to save: %w", err)
	}

	return nil
}

// getRelayURL finds the first configured relay route
func (se *SyncEngine) getRelayURL() string {
	for _, r := range se.config.Routes {
		if r.Type == "web" || r.Type == "relay" {
			return r.URL
		}
	}
	return ""
}

// sendPackets performs the HTTP POST
func (se *SyncEngine) sendPackets(url string, packets []models.EncryptedSyncPacket) error {
	data, err := json.Marshal(packets)
	if err != nil {
		return err
	}

	resp, err := http.Post(url+"/api/sync/push", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("relay rejected push: %d", resp.StatusCode)
	}
	return nil
}

// getLastRelaySyncTime returns the last successful relay sync timestamp
func (se *SyncEngine) getLastRelaySyncTime() time.Time {
	var meta models.SyncMetadata
	err := se.db.DB.Where("instance_id = ?", se.instanceID).
		Where("entity_type = ?", "relay").
		First(&meta).Error
	if err != nil {
		return time.Time{} // Zero time (very old)
	}
	if meta.LastSyncAt == nil {
		return time.Time{}
	}
	return *meta.LastSyncAt
}

// getLastRelayVersion returns the last processed packet version
func (se *SyncEngine) getLastRelayVersion() int64 {
	var maxVersion int64
	se.db.DB.Model(&models.EncryptedSyncPacket{}).
		Where("source_instance != ?", se.instanceID).
		Select("COALESCE(MAX(version), 0)").
		Scan(&maxVersion)
	return maxVersion
}

// updateRelaySyncMetadata updates the sync metadata after a relay sync
func (se *SyncEngine) updateRelaySyncMetadata(success bool, records int) {
	lastStatus := "failed"
	if success {
		lastStatus = "success"
	}
	meta := models.SyncMetadata{
		InstanceID:     se.instanceID,
		EntityType:     "relay",
		LastSyncAt:     timePtr(time.Now()),
		LastSyncStatus: lastStatus,
		RecordsSynced:  records,
	}
	se.db.DB.Where("instance_id = ? AND entity_type = ?", se.instanceID, "relay").
		Assign(meta).
		FirstOrCreate(&meta)
}

func timePtr(t time.Time) *time.Time {
	return &t
}

// parseVectorClock converts JSONB to VectorClock
func parseVectorClock(vc models.JSONB) VectorClock {
	if vc == nil || len(vc) == 0 {
		return NewVectorClock()
	}
	data, err := json.Marshal(vc)
	if err != nil {
		return NewVectorClock()
	}
	var clock VectorClock
	if err := json.Unmarshal(data, &clock); err != nil {
		return NewVectorClock()
	}
	return clock
}
