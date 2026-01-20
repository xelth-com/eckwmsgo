package sync

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/xelth-com/eckwmsgo/internal/config"
	"github.com/xelth-com/eckwmsgo/internal/database"
	"github.com/xelth-com/eckwmsgo/internal/models"
	"gorm.io/gorm"
)

// SyncEngine orchestrates all synchronization operations
type SyncEngine struct {
	mu sync.RWMutex

	// Core components
	db                *database.DB
	config            *config.SyncConfig
	instanceID        string
	connectionManager *ConnectionManager
	conflictResolver  *ConflictResolver
	checksumCalc      *ChecksumCalculator
	securityLayer     *SecurityLayer // Added security layer

	// Mesh config
	meshSecret string
	baseURL    string
	nodeRole   string

	// State
	isRunning      bool
	lastSync       time.Time
	syncInProgress bool

	// Channels
	stopChan chan struct{}
	syncChan chan SyncRequest
}

// SyncRequest represents a sync request
type SyncRequest struct {
	EntityType EntityType
	EntityID   string
	Operation  string // sync, pull, push
	Priority   int
}

// SyncResult represents the result of a sync operation
type SyncResult struct {
	Success          bool
	EntitiesSynced   int
	ConflictsFound   int
	ConflictsResolved int
	Errors           []error
	Duration         time.Duration
	Timestamp        time.Time
}

// MeshConfig holds mesh network configuration for SyncEngine
type MeshConfig struct {
	InstanceID string
	MeshSecret string
	BaseURL    string
	NodeRole   string
}

// NewSyncEngine creates a new sync engine
func NewSyncEngine(db *database.DB, cfg *config.SyncConfig, meshCfg *MeshConfig) *SyncEngine {
	instanceID := meshCfg.InstanceID

	// Build routes from config
	routes := make([]SyncRouteConfig, 0)
	for _, r := range cfg.Routes {
		routes = append(routes, SyncRouteConfig{
			URL:      r.URL,
			Type:     RouteType(r.Type),
			Timeout:  r.Timeout,
			Priority: r.Priority,
		})
	}

	// Initialize Security Layer based on Role
	role := SyncNodeRole(cfg.Role)
	if role == "" {
		role = RolePeer
	}
	secLayer := NewSecurityLayer(role)

	engine := &SyncEngine{
		db:                db,
		config:            cfg,
		instanceID:        instanceID,
		meshSecret:        meshCfg.MeshSecret,
		baseURL:           meshCfg.BaseURL,
		nodeRole:          meshCfg.NodeRole,
		connectionManager: NewConnectionManager(instanceID, routes),
		conflictResolver:  NewConflictResolver(instanceID, ConflictResolutionStrategy(cfg.ConflictResolution)),
		checksumCalc:      NewChecksumCalculator(instanceID),
		securityLayer:     secLayer,
		stopChan:          make(chan struct{}),
		syncChan:          make(chan SyncRequest, 100),
	}

	return engine
}

// Start starts the sync engine
func (se *SyncEngine) Start() error {
	se.mu.Lock()
	defer se.mu.Unlock()

	if se.isRunning {
		return fmt.Errorf("sync engine already running")
	}

	se.isRunning = true
	log.Println("🔄 Sync Engine starting...")

	// Start connection manager
	se.connectionManager.Start()

	// Start sync worker
	go se.syncWorker()

	// Start auto-sync if enabled
	if se.config.AutoSyncEnabled {
		go se.autoSyncLoop()
	}

	// Sync on startup if enabled
	if se.config.SyncOnStartup {
		go func() {
			time.Sleep(5 * time.Second) // Wait for initialization
			se.RequestFullSync()
		}()
	}

	log.Println("✅ Sync Engine started")
	return nil
}

// Stop stops the sync engine
func (se *SyncEngine) Stop() {
	se.mu.Lock()
	defer se.mu.Unlock()

	if !se.isRunning {
		return
	}

	log.Println("🛑 Stopping Sync Engine...")
	se.isRunning = false
	close(se.stopChan)
	se.connectionManager.Stop()
	log.Println("✅ Sync Engine stopped")
}

// RequestFullSync requests a full synchronization
func (se *SyncEngine) RequestFullSync() {
	log.Println("📥 Full sync requested")
	se.syncChan <- SyncRequest{
		Operation: "full_sync",
		Priority:  10,
	}
}

// RequestEntitySync requests synchronization for a specific entity
func (se *SyncEngine) RequestEntitySync(entityType EntityType, entityID string) {
	se.syncChan <- SyncRequest{
		EntityType: entityType,
		EntityID:   entityID,
		Operation:  "sync",
		Priority:   5,
	}
}

// syncWorker processes sync requests
func (se *SyncEngine) syncWorker() {
	for {
		select {
		case req := <-se.syncChan:
			se.processSyncRequest(req)
		case <-se.stopChan:
			return
		}
	}
}

// processSyncRequest processes a single sync request
func (se *SyncEngine) processSyncRequest(req SyncRequest) {
	se.mu.Lock()
	if se.syncInProgress {
		se.mu.Unlock()
		log.Println("⏳ Sync already in progress, queuing request")
		return
	}
	se.syncInProgress = true
	se.mu.Unlock()

	defer func() {
		se.mu.Lock()
		se.syncInProgress = false
		se.mu.Unlock()
	}()

	start := time.Now()
	log.Printf("🔄 Processing sync request: %s %s", req.Operation, req.EntityType)

	var result *SyncResult

	switch req.Operation {
	case "full_sync":
		result = se.performFullSync()
	case "sync":
		result = se.syncEntity(req.EntityType, req.EntityID)
	case "relay_sync":
		err := se.SyncWithRelay()
		if err != nil {
			result = &SyncResult{Success: false, Errors: []error{err}}
		} else {
			result = &SyncResult{Success: true}
		}
	default:
		log.Printf("Unknown sync operation: %s", req.Operation)
		return
	}

	duration := time.Since(start)
	log.Printf("✅ Sync completed in %v: %d entities, %d conflicts", duration, result.EntitiesSynced, result.ConflictsFound)

	se.mu.Lock()
	se.lastSync = time.Now()
	se.mu.Unlock()
}

// performFullSync performs a full synchronization
func (se *SyncEngine) performFullSync() *SyncResult {
	result := &SyncResult{
		Success:   true,
		Timestamp: time.Now(),
	}

	// Check if we're online
	if !se.connectionManager.IsOnline() {
		log.Println("⚠️ Cannot sync: offline")
		result.Success = false
		return result
	}

	// Sync each enabled entity type
	for entityName, entityCfg := range se.config.Entities {
		if !entityCfg.Enabled {
			continue
		}

		log.Printf("🔄 Syncing %s...", entityName)

		entityType := EntityType(entityName)
		count, conflicts, err := se.syncEntityType(entityType, entityCfg)

		if err != nil {
			log.Printf("⚠️ Error syncing %s: %v", entityName, err)
			result.Errors = append(result.Errors, err)
			result.Success = false
		} else {
			result.EntitiesSynced += count
			result.ConflictsFound += conflicts
		}
	}

	result.Duration = time.Since(result.Timestamp)
	return result
}

// syncEntityType syncs all entities of a specific type
func (se *SyncEngine) syncEntityType(entityType EntityType, cfg config.EntitySyncConfig) (int, int, error) {
	switch entityType {
	case EntityTypeItem:
		return se.syncItems(cfg)
	case EntityTypeWarehouse:
		return se.syncWarehouses(cfg)
	// Add other entity types...
	default:
		return 0, 0, fmt.Errorf("unsupported entity type: %s", entityType)
	}
}

// syncItems syncs items (stubbed during Odoo migration)
func (se *SyncEngine) syncItems(cfg config.EntitySyncConfig) (int, int, error) {
	// TODO: Re-implement for ProductProduct/StockLot models
	return 0, 0, nil
}

// syncWarehouses syncs warehouses (stubbed during Odoo migration)
func (se *SyncEngine) syncWarehouses(cfg config.EntitySyncConfig) (int, int, error) {
	// TODO: Re-implement for StockLocation hierarchy
	return 0, 0, nil
}

// calculateWarehouseChecksum (stubbed during Odoo migration)
func (se *SyncEngine) calculateWarehouseChecksum() {
	// TODO: Re-implement for StockLocation hierarchy
}

// storeChecksum stores a checksum in the database
func (se *SyncEngine) storeChecksum(entityType EntityType, entityID, hash, childrenHash string, childCount int) {
	checksum := models.EntityChecksum{
		EntityType:     string(entityType),
		EntityID:       entityID,
		ContentHash:    hash,
		ChildrenHash:   childrenHash,
		FullHash:       hash,
		ChildCount:     childCount,
		LastUpdated:    time.Now(),
		SourceInstance: se.instanceID,
	}

	se.db.DB.Where("entity_type = ? AND entity_id = ?", entityType, entityID).
		Assign(checksum).
		FirstOrCreate(&checksum)
}

// syncEntity syncs a specific entity
func (se *SyncEngine) syncEntity(entityType EntityType, entityID string) *SyncResult {
	result := &SyncResult{
		Success:   true,
		Timestamp: time.Now(),
	}

	log.Printf("🔄 Syncing %s:%s", entityType, entityID)

	// TODO: Implement entity-specific sync logic

	result.EntitiesSynced = 1
	result.Duration = time.Since(result.Timestamp)
	return result
}

// autoSyncLoop periodically triggers automatic synchronization
func (se *SyncEngine) autoSyncLoop() {
	ticker := time.NewTicker(time.Duration(se.config.AutoSyncInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if se.config.AutoSyncEnabled {
				log.Println("Auto-sync triggered")
				se.RequestFullSync()

				// If we are a peer or edge, also sync with relay
				role := se.GetRole()
				if role == RolePeer || role == RoleEdge {
					se.syncChan <- SyncRequest{Operation: "relay_sync", Priority: 5}
				}
			}
		case <-se.stopChan:
			return
		}
	}
}

// GetSyncStatus returns the current sync status
func (se *SyncEngine) GetSyncStatus() map[string]interface{} {
	se.mu.RLock()
	defer se.mu.RUnlock()

	return map[string]interface{}{
		"is_running":       se.isRunning,
		"sync_in_progress": se.syncInProgress,
		"last_sync":        se.lastSync,
		"is_online":        se.connectionManager.IsOnline(),
		"current_route":    se.connectionManager.GetCurrentRoute(),
		"routes":           se.connectionManager.GetAllRouteStatuses(),
	}
}

// CompareChecksums compares local and remote checksums
func (se *SyncEngine) CompareChecksums(entityType EntityType, entityID string, remoteHash string) (bool, error) {
	var checksum models.EntityChecksum

	err := se.db.DB.Where("entity_type = ? AND entity_id = ?", entityType, entityID).
		First(&checksum).Error

	if err == gorm.ErrRecordNotFound {
		return false, nil // Local doesn't have this entity
	}

	if err != nil {
		return false, err
	}

	return checksum.FullHash == remoteHash, nil
}

// GetChecksum gets the checksum for an entity
func (se *SyncEngine) GetChecksum(entityType EntityType, entityID string) (*models.EntityChecksum, error) {
	var checksum models.EntityChecksum

	err := se.db.DB.Where("entity_type = ? AND entity_id = ?", entityType, entityID).
		First(&checksum).Error

	if err != nil {
		return nil, err
	}

	return &checksum, nil
}

// GetAllChecksums gets all checksums for an entity type
func (se *SyncEngine) GetAllChecksums(entityType EntityType) ([]models.EntityChecksum, error) {
	var checksums []models.EntityChecksum

	err := se.db.DB.Where("entity_type = ?", entityType).
		Find(&checksums).Error

	if err != nil {
		return nil, err
	}

	return checksums, nil
}

// GetRole returns the node role
func (se *SyncEngine) GetRole() SyncNodeRole {
	return se.securityLayer.GetRole()
}
