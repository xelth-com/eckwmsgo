package sync

import (
	"log"
	"time"

	"github.com/xelth-com/eckwmsgo/internal/models"
	"gorm.io/gorm"
)

// RegisterHooks registers GORM callbacks for syncable models
func RegisterHooks(db *gorm.DB, calculator *ChecksumCalculator, instanceID string) {
	// Callback function forAfterCreate and AfterUpdate
	checksumCallback := func(db *gorm.DB) {
		if db.Error != nil || db.Statement.Schema == nil {
			return
		}

		// Check if the model implements SyncableEntity
		model := db.Statement.Model
		syncable, ok := model.(models.SyncableEntity)
		if !ok {
			// Not a syncable entity, skip
			return
		}

		// Calculate new hash
		hash, err := calculator.ComputeChecksum(model)
		if err != nil {
			log.Printf("🔴 SyncHook: Failed to calculate checksum for %s:%s - %v",
				syncable.GetEntityType(), syncable.GetEntityID(), err)
			return
		}

		// Update Checksum Table
		checksum := models.EntityChecksum{
			EntityType:     syncable.GetEntityType(),
			EntityID:       syncable.GetEntityID(),
			ContentHash:    hash,
			FullHash:       hash, // Simple entities have no children
			LastUpdated:    time.Now().UTC(),
			SourceInstance: instanceID,
		}

		// Use a new database session to avoid transaction conflicts
		// We need to upsert the checksum record
		err = db.Session(&gorm.Session{NewDB: true}).
			Where("entity_type = ? AND entity_id = ?", checksum.EntityType, checksum.EntityID).
			Assign(models.EntityChecksum{
				ContentHash:    checksum.ContentHash,
				FullHash:       checksum.FullHash,
				LastUpdated:    checksum.LastUpdated,
				SourceInstance: checksum.SourceInstance,
			}).
			FirstOrCreate(&checksum).Error

		if err != nil {
			log.Printf("🔴 SyncHook: Failed to update checksum table for %s:%s - %v",
				syncable.GetEntityType(), syncable.GetEntityID(), err)
			return
		}

		log.Printf("🪝 SyncHook: Updated checksum for %s:%s -> %s",
			syncable.GetEntityType(), syncable.GetEntityID(), hash[:8])
	}

	// Callback for AfterDelete
	deleteCallback := func(db *gorm.DB) {
		if db.Error != nil || db.Statement.Schema == nil {
			return
		}

		model := db.Statement.Model
		syncable, ok := model.(models.SyncableEntity)
		if !ok {
			return
		}

		// Delete the checksum record
		err := db.Session(&gorm.Session{NewDB: true}).
			Where("entity_type = ? AND entity_id = ?", syncable.GetEntityType(), syncable.GetEntityID()).
			Delete(&models.EntityChecksum{}).Error

		if err != nil {
			log.Printf("🔴 SyncHook: Failed to delete checksum for %s:%s - %v",
				syncable.GetEntityType(), syncable.GetEntityID(), err)
			return
		}

		log.Printf("🪝 SyncHook: Deleted checksum for %s:%s",
			syncable.GetEntityType(), syncable.GetEntityID())
	}

	// Register callbacks for Create, Update, and Delete
	db.Callback().Create().After("gorm:create").Register("sync:after_create", checksumCallback)
	db.Callback().Update().After("gorm:update").Register("sync:after_update", checksumCallback)
	db.Callback().Delete().After("gorm:delete").Register("sync:after_delete", deleteCallback)

	log.Println("✅ GORM sync hooks registered successfully")
}
