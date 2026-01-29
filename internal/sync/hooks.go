package sync

import (
	"log"
	"reflect"
	"time"

	"github.com/xelth-com/eckwmsgo/internal/models"
	"gorm.io/gorm"
)

// RegisterHooks registers GORM callbacks for syncable models
func RegisterHooks(db *gorm.DB, calculator *ChecksumCalculator, instanceID string) {
	// Callback function forAfterCreate and AfterUpdate
	checksumCallback := func(db *gorm.DB) {
		// Debug: log that callback was triggered
		if db.Statement.Schema != nil {
			log.Printf("🔔 GORM Callback: %s operation on %s", db.Statement.ReflectValue.Type(), db.Statement.Schema.Table)
		}

		if db.Error != nil || db.Statement.Schema == nil {
			return
		}

		// Try multiple ways to get the model
		var syncable models.SyncableEntity
		var model interface{}

		// Method 1: Try db.Statement.Model directly
		if db.Statement.Model != nil {
			if s, ok := db.Statement.Model.(models.SyncableEntity); ok {
				syncable = s
				model = db.Statement.Model
			}
		}

		// Method 2: Try db.Statement.ReflectValue if Model didn't work
		if syncable == nil && db.Statement.ReflectValue.IsValid() {
			val := db.Statement.ReflectValue
			if val.Kind() == reflect.Ptr {
				val = val.Elem()
			}
			if val.CanAddr() {
				if s, ok := val.Addr().Interface().(models.SyncableEntity); ok {
					syncable = s
					model = val.Addr().Interface()
				}
			}
			if syncable == nil {
				if s, ok := val.Interface().(models.SyncableEntity); ok {
					syncable = s
					model = val.Interface()
				}
			}
		}

		if syncable == nil {
			// Not a syncable entity, skip
			if db.Statement.Schema != nil {
				log.Printf("⏭️ SyncHook: Skipping %s (not SyncableEntity)", db.Statement.Schema.Table)
			}
			return
		}

		entityType := syncable.GetEntityType()
		entityID := syncable.GetEntityID()

		// Skip if entityID is empty (happens with batch operations)
		if entityID == "" || entityID == "0" {
			return
		}

		log.Printf("🪝 SyncHook: Processing %s:%s", entityType, entityID)

		// Calculate new hash
		hash, err := calculator.ComputeChecksum(model)
		if err != nil {
			log.Printf("🔴 SyncHook: Failed to calculate checksum for %s:%s - %v",
				entityType, entityID, err)
			return
		}

		// Update Checksum Table
		checksum := models.EntityChecksum{
			EntityType:     entityType,
			EntityID:       entityID,
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
				entityType, entityID, err)
			return
		}

		log.Printf("🪝 SyncHook: Updated checksum for %s:%s -> %s",
			entityType, entityID, hash[:8])
	}

	// Callback for AfterDelete - handles SOFT deletes
	// GORM soft delete triggers AfterDelete (not AfterUpdate), but does UPDATE instead of DELETE
	// We need to UPDATE the checksum with the new DeletedAt value, not delete it
	softDeleteCallback := func(db *gorm.DB) {
		if db.Error != nil || db.Statement.Schema == nil {
			return
		}

		// Try multiple ways to get the model (same as checksumCallback)
		var syncable models.SyncableEntity
		var model interface{}

		if db.Statement.Model != nil {
			if s, ok := db.Statement.Model.(models.SyncableEntity); ok {
				syncable = s
				model = db.Statement.Model
			}
		}

		if syncable == nil && db.Statement.ReflectValue.IsValid() {
			val := db.Statement.ReflectValue
			if val.Kind() == reflect.Ptr {
				val = val.Elem()
			}
			if val.CanAddr() {
				if s, ok := val.Addr().Interface().(models.SyncableEntity); ok {
					syncable = s
					model = val.Addr().Interface()
				}
			}
			if syncable == nil {
				if s, ok := val.Interface().(models.SyncableEntity); ok {
					syncable = s
					model = val.Interface()
				}
			}
		}

		if syncable == nil {
			return
		}

		entityType := syncable.GetEntityType()
		entityID := syncable.GetEntityID()

		if entityID == "" || entityID == "0" {
			return
		}

		log.Printf("🪝 SyncHook: Soft delete detected for %s:%s", entityType, entityID)

		// Calculate new hash (now includes DeletedAt timestamp)
		hash, err := calculator.ComputeChecksum(model)
		if err != nil {
			log.Printf("🔴 SyncHook: Failed to calculate checksum for soft-deleted %s:%s - %v",
				entityType, entityID, err)
			return
		}

		// UPDATE the checksum record (not delete it!)
		checksum := models.EntityChecksum{
			EntityType:     entityType,
			EntityID:       entityID,
			ContentHash:    hash,
			FullHash:       hash,
			LastUpdated:    time.Now().UTC(),
			SourceInstance: instanceID,
		}

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
			log.Printf("🔴 SyncHook: Failed to update checksum for soft-deleted %s:%s - %v",
				entityType, entityID, err)
			return
		}

		log.Printf("🪝 SyncHook: Updated checksum for soft-deleted %s:%s -> %s",
			entityType, entityID, hash[:8])
	}

	// Register callbacks for Create, Update, and Delete (soft delete)
	db.Callback().Create().After("gorm:create").Register("sync:after_create", checksumCallback)
	db.Callback().Update().After("gorm:update").Register("sync:after_update", checksumCallback)
	db.Callback().Delete().After("gorm:delete").Register("sync:after_delete", softDeleteCallback)
	// Note: For soft deletes, GORM calls AfterDelete but performs UPDATE internally
	// The softDeleteCallback updates the checksum (including DeletedAt) instead of deleting it

	log.Println("✅ GORM sync hooks registered successfully")
}
