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
	EntityType     string    `gorm:"type:varchar(50);not null;index:idx_entity_lookup" json:"entityType"`
	EntityID       string    `gorm:"type:varchar(255);not null;index:idx_entity_lookup" json:"entityId"`
	ContentHash    string    `gorm:"type:varchar(64);not null" json:"contentHash"`
	ChildrenHash   string    `gorm:"type:varchar(64)" json:"childrenHash"`
	FullHash       string    `gorm:"type:varchar(64);not null;index:idx_full_hash" json:"fullHash"`
	ChildCount     int       `gorm:"default:0" json:"childCount"`
	LastUpdated    time.Time `gorm:"not null;default:CURRENT_TIMESTAMP;index:idx_updated" json:"lastUpdated"`
	SourceInstance string    `gorm:"type:varchar(255)" json:"sourceInstance"`
	SourceDevice   *string   `gorm:"type:varchar(255)" json:"sourceDevice,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
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
	ID               uint       `gorm:"primaryKey" json:"id"`
	InstanceID       string     `gorm:"type:varchar(255);not null;uniqueIndex:idx_instance_entity" json:"instance_id"`
	EntityType       string     `gorm:"type:varchar(100);not null;uniqueIndex:idx_instance_entity" json:"entityType"`
	LastSyncAt       *time.Time `json:"lastSyncAt"`
	LastFullSyncAt   *time.Time `json:"lastFullSyncAt"`
	LastSyncStatus   string     `gorm:"type:varchar(50)" json:"lastSyncStatus"`
	RecordsSynced    int        `gorm:"default:0" json:"recordsSynced"`
	RecordsConflicts int        `gorm:"default:0" json:"recordsConflicts"`
	SyncDurationMs   int        `json:"syncDurationMs"`
	VectorClock      JSONB      `gorm:"type:jsonb;default:'{}'" json:"vector_clock"`
	ErrorMessage     *string    `gorm:"type:text" json:"errorMessage,omitempty"`
	CreatedAt        time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"createdAt"`
	UpdatedAt        time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"updatedAt"`
}

// TableName specifies the table name
func (SyncMetadata) TableName() string {
	return "sync_metadata"
}

// SyncConflict represents a synchronization conflict
type SyncConflict struct {
	ID                     uint       `gorm:"primaryKey" json:"id"`
	EntityType             string     `gorm:"type:varchar(100);not null;index:idx_entity" json:"entityType"`
	EntityID               string     `gorm:"type:varchar(255);not null;index:idx_entity" json:"entityId"`
	ConflictType           string     `gorm:"type:varchar(50)" json:"conflictType"`
	LocalData              JSONB      `gorm:"type:jsonb" json:"local_data"`
	LocalMetadata          JSONB      `gorm:"type:jsonb" json:"local_metadata"`
	RemoteData             JSONB      `gorm:"type:jsonb" json:"remoteData"`
	RemoteMetadata         JSONB      `gorm:"type:jsonb" json:"remoteMetadata"`
	AutoResolutionStrategy string     `gorm:"type:varchar(50)" json:"autoResolutionStrategy"`
	AutoResolutionWinner   string     `gorm:"type:varchar(50)" json:"autoResolutionWinner"`
	ManualResolution       JSONB      `gorm:"type:jsonb" json:"manual_resolution"`
	Status                 string     `gorm:"type:varchar(50);default:'pending';index:idx_pending" json:"status"`
	ResolvedAt             *time.Time `json:"resolvedAt"`
	ResolvedBy             *string    `gorm:"type:varchar(255)" json:"resolvedBy,omitempty"`
	CreatedAt              time.Time  `gorm:"default:CURRENT_TIMESTAMP;index:idx_pending" json:"createdAt"`
}

// TableName specifies the table name
func (SyncConflict) TableName() string {
	return "sync_conflicts"
}

// SyncQueue represents a queue of changes to be synchronized
type SyncQueue struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	EntityType     string     `gorm:"type:varchar(100);not null" json:"entityType"`
	EntityID       string     `gorm:"type:varchar(255);not null" json:"entityId"`
	Operation      string     `gorm:"type:varchar(20);not null" json:"operation"` // create, update, delete
	Payload        JSONB      `gorm:"type:jsonb" json:"payload"`
	Metadata       JSONB      `gorm:"type:jsonb" json:"metadata"`
	Priority       int        `gorm:"default:5;index:idx_pending" json:"priority"`
	RetryCount     int        `gorm:"default:0" json:"retryCount"`
	MaxRetries     int        `gorm:"default:3" json:"max_retries"`
	ScheduledAt    time.Time  `gorm:"default:CURRENT_TIMESTAMP;index:idx_pending" json:"scheduledAt"`
	ProcessedAt    *time.Time `json:"processed_at"`
	Status         string     `gorm:"type:varchar(50);default:'pending';index:idx_pending" json:"status"`
	ErrorMessage   *string    `gorm:"type:text" json:"errorMessage,omitempty"`
	TargetInstance string     `gorm:"type:varchar(255);index:idx_target" json:"targetInstance"`
	CreatedAt      time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"createdAt"`
}

// TableName specifies the table name
func (SyncQueue) TableName() string {
	return "sync_queue"
}

// SyncRoute tracks synchronization routes and their health
type SyncRoute struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	InstanceID    string     `gorm:"type:varchar(255);not null;uniqueIndex:idx_instance_route" json:"instance_id"`
	RouteURL      string     `gorm:"type:varchar(500);not null;uniqueIndex:idx_instance_route" json:"route_url"`
	RouteType     string     `gorm:"type:varchar(50);not null" json:"route_type"` // primary, fallback, web
	IsActive      bool       `gorm:"default:true" json:"isActive"`
	LastSuccessAt *time.Time `json:"lastSuccessAt"`
	LastFailureAt *time.Time `json:"lastFailureAt"`
	SuccessCount  int        `gorm:"default:0" json:"success_count"`
	FailureCount  int        `gorm:"default:0" json:"failure_count"`
	AvgLatencyMs  int        `json:"avgLatencyMs"`
	CreatedAt     time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"createdAt"`
	UpdatedAt     time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"updatedAt"`
}

// TableName specifies the table name
func (SyncRoute) TableName() string {
	return "sync_routes"
}
