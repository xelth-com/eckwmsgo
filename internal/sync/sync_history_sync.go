package sync

import (
	"log"
	"time"

	"github.com/xelth-com/eckwmsgo/internal/config"
	"github.com/xelth-com/eckwmsgo/internal/models"
)

// syncSyncHistory syncs SyncHistory records (logs) between nodes
// Uses simple time-based sync with limit (no checksums needed for logs)
func (se *SyncEngine) syncSyncHistory(cfg config.EntitySyncConfig) (int, int, error) {
	log.Printf("🔄 Syncing SyncHistory (last %d records)...", cfg.MaxRecords)

	// Get recent sync history records
	var history []models.SyncHistory
	query := se.db.DB.Order("started_at DESC")

	// Apply max records limit (default 30)
	limit := cfg.MaxRecords
	if limit <= 0 {
		limit = 30
	}
	query = query.Limit(limit)

	// Apply history depth (days)
	if cfg.HistoryDepth > 0 {
		since := time.Now().AddDate(0, 0, -cfg.HistoryDepth)
		query = query.Where("started_at > ?", since)
	}

	if err := query.Find(&history).Error; err != nil {
		log.Printf("❌ Failed to fetch sync history: %v", err)
		return 0, 0, err
	}

	log.Printf("   Found %d sync history records", len(history))

	// Sync history is pushed via mesh_sync.go's SyncWithRelay
	// This function just logs what we have locally

	return len(history), 0, nil
}
