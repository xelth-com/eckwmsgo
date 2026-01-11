package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// JSONB type for PostgreSQL JSONB fields
type JSONB map[string]interface{}

// Scan implements sql.Scanner interface
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = make(JSONB)
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal JSONB value: %v", value)
	}

	result := make(JSONB)
	err := json.Unmarshal(bytes, &result)
	*j = result
	return err
}

// Value implements driver.Valuer interface
func (j JSONB) Value() (driver.Value, error) {
	if len(j) == 0 {
		return json.Marshal(map[string]interface{}{})
	}
	return json.Marshal(j)
}

// EntityChecksum represents checksum information for entities
type EntityChecksum struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	EntityType     string    `gorm:"type:varchar(50);not null;index:idx_entity_lookup" json:"entity_type"`
	EntityID       string    `gorm:"type:varchar(255);not null;index:idx_entity_lookup" json:"entity_id"`
	ContentHash    string    `gorm:"type:varchar(64);not null" json:"content_hash"`
	ChildrenHash   string    `gorm:"type:varchar(64)" json:"children_hash"`
	FullHash       string    `gorm:"type:varchar(64);not null;index:idx_full_hash" json:"full_hash"`
	ChildCount     int       `gorm:"default:0" json:"child_count"`
	LastUpdated    time.Time `gorm:"not null;default:CURRENT_TIMESTAMP;index:idx_updated" json:"last_updated"`
	SourceInstance string    `gorm:"type:varchar(255)" json:"source_instance"`
	SourceDevice   *string   `gorm:"type:varchar(255)" json:"source_device,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// TableName specifies the table name
func (EntityChecksum) TableName() string {
	return "entity_checksums"
}

// BeforeCreate hook
func (ec *EntityChecksum) BeforeCreate(tx *gorm.DB) error {
	if ec.LastUpdated.IsZero() {
		ec.LastUpdated = time.Now().UTC()
	}
	return nil
}

// SyncMetadata tracks synchronization status per entity type per instance
type SyncMetadata struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	InstanceID       string    `gorm:"type:varchar(255);not null;uniqueIndex:idx_instance_entity" json:"instance_id"`
	EntityType       string    `gorm:"type:varchar(100);not null;uniqueIndex:idx_instance_entity" json:"entity_type"`
	LastSyncAt       *time.Time `json:"last_sync_at"`
	LastFullSyncAt   *time.Time `json:"last_full_sync_at"`
	LastSyncStatus   string    `gorm:"type:varchar(50)" json:"last_sync_status"`
	RecordsSynced    int       `gorm:"default:0" json:"records_synced"`
	RecordsConflicts int       `gorm:"default:0" json:"records_conflicts"`
	SyncDurationMs   int       `json:"sync_duration_ms"`
	VectorClock      JSONB     `gorm:"type:jsonb;default:'{}'" json:"vector_clock"`
	ErrorMessage     *string   `gorm:"type:text" json:"error_message,omitempty"`
	CreatedAt        time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt        time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
}

// TableName specifies the table name
func (SyncMetadata) TableName() string {
	return "sync_metadata"
}

// SyncConflict represents a synchronization conflict
type SyncConflict struct {
	ID                      uint      `gorm:"primaryKey" json:"id"`
	EntityType              string    `gorm:"type:varchar(100);not null;index:idx_entity" json:"entity_type"`
	EntityID                string    `gorm:"type:varchar(255);not null;index:idx_entity" json:"entity_id"`
	ConflictType            string    `gorm:"type:varchar(50)" json:"conflict_type"`
	LocalData               JSONB     `gorm:"type:jsonb" json:"local_data"`
	LocalMetadata           JSONB     `gorm:"type:jsonb" json:"local_metadata"`
	RemoteData              JSONB     `gorm:"type:jsonb" json:"remote_data"`
	RemoteMetadata          JSONB     `gorm:"type:jsonb" json:"remote_metadata"`
	AutoResolutionStrategy  string    `gorm:"type:varchar(50)" json:"auto_resolution_strategy"`
	AutoResolutionWinner    string    `gorm:"type:varchar(50)" json:"auto_resolution_winner"`
	ManualResolution        JSONB     `gorm:"type:jsonb" json:"manual_resolution"`
	Status                  string    `gorm:"type:varchar(50);default:'pending';index:idx_pending" json:"status"`
	ResolvedAt              *time.Time `json:"resolved_at"`
	ResolvedBy              *string   `gorm:"type:varchar(255)" json:"resolved_by,omitempty"`
	CreatedAt               time.Time `gorm:"default:CURRENT_TIMESTAMP;index:idx_pending" json:"created_at"`
}

// TableName specifies the table name
func (SyncConflict) TableName() string {
	return "sync_conflicts"
}

// SyncQueue represents a queue of changes to be synchronized
type SyncQueue struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	EntityType     string    `gorm:"type:varchar(100);not null" json:"entity_type"`
	EntityID       string    `gorm:"type:varchar(255);not null" json:"entity_id"`
	Operation      string    `gorm:"type:varchar(20);not null" json:"operation"` // create, update, delete
	Payload        JSONB     `gorm:"type:jsonb" json:"payload"`
	Metadata       JSONB     `gorm:"type:jsonb" json:"metadata"`
	Priority       int       `gorm:"default:5;index:idx_pending" json:"priority"`
	RetryCount     int       `gorm:"default:0" json:"retry_count"`
	MaxRetries     int       `gorm:"default:3" json:"max_retries"`
	ScheduledAt    time.Time `gorm:"default:CURRENT_TIMESTAMP;index:idx_pending" json:"scheduled_at"`
	ProcessedAt    *time.Time `json:"processed_at"`
	Status         string    `gorm:"type:varchar(50);default:'pending';index:idx_pending" json:"status"`
	ErrorMessage   *string   `gorm:"type:text" json:"error_message,omitempty"`
	TargetInstance string    `gorm:"type:varchar(255);index:idx_target" json:"target_instance"`
	CreatedAt      time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
}

// TableName specifies the table name
func (SyncQueue) TableName() string {
	return "sync_queue"
}

// SyncRoute tracks synchronization routes and their health
type SyncRoute struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	InstanceID     string     `gorm:"type:varchar(255);not null;uniqueIndex:idx_instance_route" json:"instance_id"`
	RouteURL       string     `gorm:"type:varchar(500);not null;uniqueIndex:idx_instance_route" json:"route_url"`
	RouteType      string     `gorm:"type:varchar(50);not null" json:"route_type"` // primary, fallback, web
	IsActive       bool       `gorm:"default:true" json:"is_active"`
	LastSuccessAt  *time.Time `json:"last_success_at"`
	LastFailureAt  *time.Time `json:"last_failure_at"`
	SuccessCount   int        `gorm:"default:0" json:"success_count"`
	FailureCount   int        `gorm:"default:0" json:"failure_count"`
	AvgLatencyMs   int        `json:"avg_latency_ms"`
	CreatedAt      time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at"`
}

// TableName specifies the table name
func (SyncRoute) TableName() string {
	return "sync_routes"
}
