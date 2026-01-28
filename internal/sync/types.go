package sync

// TruthSource represents the origin of data change
type TruthSource string

const (
	TruthSourcePDA      TruthSource = "pda"
	TruthSourceLocal    TruthSource = "local_server"
	TruthSourceWeb      TruthSource = "web_server"
	TruthSourceAPI      TruthSource = "api_external"
	TruthSourceInternal TruthSource = "internal"
)

// TruthPriority represents the priority level of a data source
type TruthPriority int

const (
	PriorityPhysical  TruthPriority = 100 // Physical scanning/action
	PriorityLocal     TruthPriority = 80  // Local server
	PriorityRegional  TruthPriority = 60  // Regional server
	PriorityGlobal    TruthPriority = 40  // Global web server
	PriorityExternal  TruthPriority = 20  // External API
	PriorityUndefined TruthPriority = 0   // Unknown source
)

// SyncMode defines the synchronization strategy
type SyncMode string

const (
	SyncModeFull        SyncMode = "full"        // Full synchronization
	SyncModeIncremental SyncMode = "incremental" // Only changes
	SyncModeSelective   SyncMode = "selective"   // Specific entities
	SyncModeCache       SyncMode = "cache"       // Minimal cache
	SyncModeMaster      SyncMode = "master"      // Master server
)

// SyncDirection defines the direction of synchronization
type SyncDirection string

const (
	SyncDirectionBidirectional SyncDirection = "bidirectional"
	SyncDirectionPullOnly      SyncDirection = "pull_only"
	SyncDirectionPushOnly      SyncDirection = "push_only"
)

// ConflictResolutionStrategy defines how to resolve conflicts
type ConflictResolutionStrategy string

const (
	ConflictServerWins     ConflictResolutionStrategy = "server_wins"
	ConflictClientWins     ConflictResolutionStrategy = "client_wins"
	ConflictLastWriteWins  ConflictResolutionStrategy = "last_write_wins"
	ConflictManual         ConflictResolutionStrategy = "manual"
	ConflictVersioning     ConflictResolutionStrategy = "versioning"
	ConflictPriorityBased  ConflictResolutionStrategy = "priority_based"
	ConflictPhysicalAction ConflictResolutionStrategy = "physical_action"
)

// EntityType represents the type of entity being synchronized
type EntityType string

const (
	// New Odoo-aligned entity types (match config keys)
	EntityTypeProduct  EntityType = "products"  // ProductProduct (was "item")
	EntityTypeLocation EntityType = "locations" // StockLocation (was "place")
	EntityTypeQuant    EntityType = "quants"    // StockQuant (new)
	EntityTypeLot      EntityType = "lots"      // StockLot (new)
	EntityTypePackage  EntityType = "packages"  // StockQuantPackage (was "box")
	EntityTypePicking  EntityType = "pickings"  // StockPicking (new)
	EntityTypePartner  EntityType = "partners"  // ResPartner (new)

	// Legacy entity types (kept for backward compatibility)
	EntityTypeItem      EntityType = "items"      // Deprecated, use EntityTypeProduct
	EntityTypeBox       EntityType = "boxes"      // Deprecated, use EntityTypePackage
	EntityTypePlace     EntityType = "places"     // Deprecated, use EntityTypeLocation
	EntityTypeRack      EntityType = "racks"      // Legacy
	EntityTypeWarehouse EntityType = "warehouses" // Legacy
	EntityTypeOrder     EntityType = "orders"     // Legacy
	EntityTypeUser      EntityType = "users"      // Legacy
	EntityTypeDevice    EntityType = "devices"    // Legacy
	EntityTypeShipment  EntityType = "shipments"  // Legacy
	EntityTypeTracking  EntityType = "tracking"   // Legacy
)

// SyncStatus represents the status of a sync operation
type SyncStatus string

const (
	SyncStatusPending    SyncStatus = "pending"
	SyncStatusProcessing SyncStatus = "processing"
	SyncStatusCompleted  SyncStatus = "completed"
	SyncStatusFailed     SyncStatus = "failed"
	SyncStatusConflict   SyncStatus = "conflict"
)

// ConflictStatus represents the status of a conflict
type ConflictStatus string

const (
	ConflictStatusPending  ConflictStatus = "pending"
	ConflictStatusResolved ConflictStatus = "resolved"
	ConflictStatusIgnored  ConflictStatus = "ignored"
)

// SyncNodeRole defines the role of the server in the mesh
type SyncNodeRole string

const (
	RoleMaster     SyncNodeRole = "master"      // Full DB, Read/Write, Authoritative
	RolePeer       SyncNodeRole = "peer"        // Full DB, Read/Write, Trusted
	RoleEdge       SyncNodeRole = "edge"        // Partial DB, Trusted
	RoleBlindRelay SyncNodeRole = "blind_relay" // No DB access, Encrypted storage only, Untrusted
)

// SyncPacketType defines if payload is plain or encrypted
type SyncPacketType string

const (
	PacketPlain     SyncPacketType = "plain"
	PacketEncrypted SyncPacketType = "encrypted"
)

// ChecksumItem represents a minimal hash pair for negotiation
type ChecksumItem struct {
	EntityID string `json:"id"`
	Hash     string `json:"h"` // Short json tag for bandwidth
}

// MeshNegotiationRequest is sent by source to check what needs syncing
type MeshNegotiationRequest struct {
	EntityType string         `json:"type"`
	Items      []ChecksumItem `json:"items"`
}

// MeshNegotiationResponse is returned by target with list of IDs to fetch
type MeshNegotiationResponse struct {
	RequestIDs []string `json:"req_ids"` // IDs that are missing or different on target
}
