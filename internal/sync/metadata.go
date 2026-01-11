package sync

import (
	"encoding/json"
	"time"
)

// EntityMetadata contains synchronization metadata for any entity
type EntityMetadata struct {
	// ========== IDENTIFICATION ==========
	EntityID   string     `json:"entity_id"`
	EntityType EntityType `json:"entity_type"`

	// ========== VERSIONING ==========
	Version   int64     `json:"version"`    // Monotonically increasing version
	UpdatedAt time.Time `json:"updated_at"` // UTC timestamp

	// ========== TRUTH SOURCE ==========
	Source         TruthSource   `json:"source"`           // Who created this change
	SourcePriority TruthPriority `json:"source_priority"`  // Priority of the source
	InstanceID     string        `json:"instance_id"`      // ID of device/server
	DeviceID       *string       `json:"device_id"`        // PDA device ID if applicable
	UserID         *string       `json:"user_id"`          // User who made the change

	// ========== CHECKSUMS ==========
	ContentHash   string `json:"content_hash"`   // SHA256 of content
	HierarchyHash string `json:"hierarchy_hash"` // Aggregated hash of hierarchy

	// ========== VECTOR CLOCK ==========
	VectorClock VectorClock `json:"vector_clock"` // Causality tracking

	// ========== SYNCHRONIZATION ==========
	SyncedToWeb   bool   `json:"synced_to_web"`
	SyncedToLocal bool   `json:"synced_to_local"`
	SyncAttempts  int    `json:"sync_attempts"`
	LastSyncError string `json:"last_sync_error,omitempty"`

	// ========== CONFLICTS ==========
	HasConflict  bool     `json:"has_conflict"`
	ConflictWith []string `json:"conflict_with,omitempty"` // IDs of conflicting versions
}

// NewEntityMetadata creates a new EntityMetadata with defaults
func NewEntityMetadata(entityType EntityType, entityID, instanceID string) *EntityMetadata {
	return &EntityMetadata{
		EntityID:      entityID,
		EntityType:    entityType,
		Version:       1,
		UpdatedAt:     time.Now().UTC(),
		Source:        TruthSourceInternal,
		SourcePriority: PriorityUndefined,
		InstanceID:    instanceID,
		VectorClock:   NewVectorClock(),
		SyncAttempts:  0,
		HasConflict:   false,
	}
}

// IncrementVersion increments the version and updates the vector clock
func (em *EntityMetadata) IncrementVersion() {
	em.Version++
	em.VectorClock.Increment(em.InstanceID)
	em.UpdatedAt = time.Now().UTC()
}

// SetSource sets the truth source and its priority
func (em *EntityMetadata) SetSource(source TruthSource, deviceID *string) {
	em.Source = source
	em.DeviceID = deviceID

	// Set priority based on source
	switch source {
	case TruthSourcePDA:
		em.SourcePriority = PriorityPhysical
	case TruthSourceLocal:
		em.SourcePriority = PriorityLocal
	case TruthSourceWeb:
		em.SourcePriority = PriorityGlobal
	case TruthSourceAPI:
		em.SourcePriority = PriorityExternal
	default:
		em.SourcePriority = PriorityUndefined
	}
}

// MarkConflict marks this entity as having a conflict
func (em *EntityMetadata) MarkConflict(conflictWithID string) {
	em.HasConflict = true
	if !contains(em.ConflictWith, conflictWithID) {
		em.ConflictWith = append(em.ConflictWith, conflictWithID)
	}
}

// ClearConflict clears the conflict status
func (em *EntityMetadata) ClearConflict() {
	em.HasConflict = false
	em.ConflictWith = nil
}

// ToJSON converts metadata to JSON string
func (em *EntityMetadata) ToJSON() (string, error) {
	data, err := json.Marshal(em)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FromJSON parses metadata from JSON string
func (em *EntityMetadata) FromJSON(data string) error {
	return json.Unmarshal([]byte(data), em)
}

// ChecksumEntity represents checksum information for an entity
type ChecksumEntity struct {
	// Entity identification
	EntityType EntityType `json:"entity_type"`
	EntityID   string     `json:"entity_id"`

	// Checksums
	ContentHash  string `json:"content_hash"`  // Hash of own data
	ChildrenHash string `json:"children_hash"` // Hash of children (empty if no children)
	FullHash     string `json:"full_hash"`     // Hash(content + children)

	// Metadata
	ChildCount  int       `json:"child_count"`
	LastUpdated time.Time `json:"last_updated"`

	// Source information
	SourceInstance string  `json:"source_instance"`
	SourceDevice   *string `json:"source_device,omitempty"`

	// Children checksums (for tree traversal)
	Children map[string]string `json:"children,omitempty"` // {child_id: hash}
}

// NewChecksumEntity creates a new ChecksumEntity
func NewChecksumEntity(entityType EntityType, entityID string) *ChecksumEntity {
	return &ChecksumEntity{
		EntityType:  entityType,
		EntityID:    entityID,
		LastUpdated: time.Now().UTC(),
		ChildCount:  0,
		Children:    make(map[string]string),
	}
}

// SetContentHash sets the content hash
func (ce *ChecksumEntity) SetContentHash(hash string) {
	ce.ContentHash = hash
	ce.LastUpdated = time.Now().UTC()
}

// SetChildrenHash sets the children hash and count
func (ce *ChecksumEntity) SetChildrenHash(hash string, count int) {
	ce.ChildrenHash = hash
	ce.ChildCount = count
	ce.LastUpdated = time.Now().UTC()
}

// AddChild adds a child hash to the children map
func (ce *ChecksumEntity) AddChild(childID, childHash string) {
	if ce.Children == nil {
		ce.Children = make(map[string]string)
	}
	ce.Children[childID] = childHash
}

// HasChildren returns true if entity has children
func (ce *ChecksumEntity) HasChildren() bool {
	return ce.ChildCount > 0
}

// Matches returns true if the full hash matches the provided hash
func (ce *ChecksumEntity) Matches(hash string) bool {
	return ce.FullHash == hash
}

// Helper function
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
