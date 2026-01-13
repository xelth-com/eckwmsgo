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
	EntityTypeItem      EntityType = "item"
	EntityTypeBox       EntityType = "box"
	EntityTypePlace     EntityType = "place"
	EntityTypeRack      EntityType = "rack"
	EntityTypeWarehouse EntityType = "warehouse"
	EntityTypeOrder     EntityType = "order"
	EntityTypeUser      EntityType = "user"
	EntityTypeDevice    EntityType = "device"
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
