package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/xelth-com/eckwmsgo/internal/database"
	"github.com/xelth-com/eckwmsgo/internal/models"
	"github.com/xelth-com/eckwmsgo/internal/sync"
	"github.com/gorilla/mux"
)

// SyncHandler handles synchronization requests
type SyncHandler struct {
	db         *database.DB
	syncEngine *sync.SyncEngine
}

// NewSyncHandler creates a new sync handler
func NewSyncHandler(db *database.DB, syncEngine *sync.SyncEngine) *SyncHandler {
	return &SyncHandler{
		db:         db,
		syncEngine: syncEngine,
	}
}

// RegisterRoutes registers sync routes
func (sh *SyncHandler) RegisterRoutes(r *mux.Router) {
	// Checksum endpoints
	r.HandleFunc("/api/sync/checksums/{entity_type}/{entity_id}", sh.GetChecksum).Methods("GET")
	r.HandleFunc("/api/sync/checksums/{entity_type}", sh.GetAllChecksums).Methods("GET")
	r.HandleFunc("/api/sync/checksums/compare", sh.CompareChecksums).Methods("POST")
	r.HandleFunc("/api/sync/checksums/rebuild", sh.RebuildChecksums).Methods("POST")

	// Sync control endpoints
	r.HandleFunc("/api/sync/status", sh.GetSyncStatus).Methods("GET")
	r.HandleFunc("/api/sync/start", sh.StartSync).Methods("POST")
	r.HandleFunc("/api/sync/full", sh.TriggerFullSync).Methods("POST")

	// Entity sync endpoints
	r.HandleFunc("/api/sync/entity/{entity_type}/{entity_id}", sh.SyncEntity).Methods("POST")
	r.HandleFunc("/api/sync/pull", sh.PullUpdates).Methods("POST")
	r.HandleFunc("/api/sync/push", sh.PushUpdates).Methods("POST")

	// Mesh sync endpoints (for node-to-node synchronization)
	r.HandleFunc("/api/mesh/pull", sh.MeshPull).Methods("POST")
	r.HandleFunc("/api/mesh/push", sh.MeshPush).Methods("POST")
	r.HandleFunc("/api/mesh/trigger", sh.TriggerMeshSync).Methods("POST")
}

// GetChecksum returns the checksum for a specific entity
func (sh *SyncHandler) GetChecksum(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	entityType := sync.EntityType(vars["entity_type"])
	entityID := vars["entity_id"]

	checksum, err := sh.syncEngine.GetChecksum(entityType, entityID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(checksum)
}

// GetAllChecksums returns all checksums for an entity type
func (sh *SyncHandler) GetAllChecksums(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	entityType := sync.EntityType(vars["entity_type"])

	checksums, err := sh.syncEngine.GetAllChecksums(entityType)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"entity_type": entityType,
		"count":       len(checksums),
		"checksums":   checksums,
	})
}

// CompareChecksums compares local and remote checksums
func (sh *SyncHandler) CompareChecksums(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Entities []struct {
			EntityType string `json:"entity_type"`
			EntityID   string `json:"entity_id"`
			Hash       string `json:"hash"`
		} `json:"entities"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	matches := make([]string, 0)
	mismatches := make([]map[string]interface{}, 0)

	for _, entity := range req.Entities {
		match, err := sh.syncEngine.CompareChecksums(
			sync.EntityType(entity.EntityType),
			entity.EntityID,
			entity.Hash,
		)

		if err != nil {
			mismatches = append(mismatches, map[string]interface{}{
				"entity_type": entity.EntityType,
				"entity_id":   entity.EntityID,
				"error":       err.Error(),
				"action":      "error",
			})
			continue
		}

		if match {
			matches = append(matches, entity.EntityType+":"+entity.EntityID)
		} else {
			localChecksum, _ := sh.syncEngine.GetChecksum(
				sync.EntityType(entity.EntityType),
				entity.EntityID,
			)

			localHash := ""
			if localChecksum != nil {
				localHash = localChecksum.FullHash
			}

			mismatches = append(mismatches, map[string]interface{}{
				"entity_type": entity.EntityType,
				"entity_id":   entity.EntityID,
				"local_hash":  localHash,
				"remote_hash": entity.Hash,
				"action":      determineAction(localHash, entity.Hash),
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"matches":    matches,
		"mismatches": mismatches,
	})
}

// RebuildChecksums rebuilds checksums for entities
func (sh *SyncHandler) RebuildChecksums(w http.ResponseWriter, r *http.Request) {
	var req struct {
		EntityType string  `json:"entity_type"`
		EntityID   *string `json:"entity_id"`
		Recursive  bool    `json:"recursive"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Trigger a full sync to rebuild checksums
	sh.syncEngine.RequestFullSync()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Checksum rebuild triggered",
		"status":  "processing",
	})
}

// GetSyncStatus returns the current sync status
func (sh *SyncHandler) GetSyncStatus(w http.ResponseWriter, r *http.Request) {
	status := sh.syncEngine.GetSyncStatus()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// StartSync starts the sync engine if not already running
func (sh *SyncHandler) StartSync(w http.ResponseWriter, r *http.Request) {
	if err := sh.syncEngine.Start(); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Sync engine started",
		"status":  "running",
	})
}

// TriggerFullSync triggers a full synchronization
func (sh *SyncHandler) TriggerFullSync(w http.ResponseWriter, r *http.Request) {
	sh.syncEngine.RequestFullSync()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Full sync triggered",
		"status":  "processing",
	})
}

// SyncEntity syncs a specific entity
func (sh *SyncHandler) SyncEntity(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	entityType := sync.EntityType(vars["entity_type"])
	entityID := vars["entity_id"]

	sh.syncEngine.RequestEntitySync(entityType, entityID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":     "Entity sync requested",
		"entity_type": entityType,
		"entity_id":   entityID,
		"status":      "processing",
	})
}

// PullUpdates pulls updates from remote
func (sh *SyncHandler) PullUpdates(w http.ResponseWriter, r *http.Request) {
	// 1. Check if we are a Blind Relay
	if sh.syncEngine.GetRole() == sync.RoleBlindRelay {
		sh.handleRelayPull(w, r)
		return
	}

	// Standard peer pull logic
	var req struct {
		EntityTypes []string `json:"entity_types"`
		Since       *string  `json:"since"` // ISO timestamp
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// TODO: Implement standard pull logic for peers

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Standard pull not implemented yet",
		"status":  "processing",
	})
}

// PushUpdates pushes local updates to remote
func (sh *SyncHandler) PushUpdates(w http.ResponseWriter, r *http.Request) {
	// 1. Check if we are a Blind Relay
	if sh.syncEngine.GetRole() == sync.RoleBlindRelay {
		sh.handleRelayPush(w, r)
		return
	}

	// Standard peer push logic
	var req struct {
		EntityTypes []string `json:"entity_types"`
		Since       *string  `json:"since"` // ISO timestamp
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// TODO: Implement standard push logic for peers

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Standard push not implemented yet",
		"status":  "processing",
	})
}

// handleRelayPush accepts encrypted packets and stores them blindly
func (sh *SyncHandler) handleRelayPush(w http.ResponseWriter, r *http.Request) {
	var packets []models.EncryptedSyncPacket
	if err := json.NewDecoder(r.Body).Decode(&packets); err != nil {
		http.Error(w, "Invalid encrypted packet format", http.StatusBadRequest)
		return
	}

	log.Printf("🔒 Relay: Received %d encrypted packets", len(packets))

	savedCount := 0
	for _, p := range packets {
		// Blind Upsert based on EntityID + EntityType
		// Relay doesn't check conflict logic deeply, just stores the latest version
		result := sh.db.Where("entity_type = ? AND entity_id = ?", p.EntityType, p.EntityID).
			Assign(p).
			FirstOrCreate(&models.EncryptedSyncPacket{})

		if result.Error == nil {
			savedCount++
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"saved":  savedCount,
	})
}

// handleRelayPull serves encrypted packets blindly
func (sh *SyncHandler) handleRelayPull(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SinceVersion int64 `json:"since_version"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req) // Optional filter

	var packets []models.EncryptedSyncPacket
	// In a real scenario, we would filter by client's last known version/vector clock
	// For now, return all packets newer than requested version
	query := sh.db.Order("updated_at DESC").Limit(100)

	if req.SinceVersion > 0 {
		query = query.Where("version > ?", req.SinceVersion)
	}

	if err := query.Find(&packets).Error; err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	log.Printf("🔒 Relay: Serving %d encrypted packets", len(packets))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(packets)
}

// MeshPull handles data pull requests from other mesh nodes
func (sh *SyncHandler) MeshPull(w http.ResponseWriter, r *http.Request) {
	var req sync.MeshSyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("Mesh Pull: Request from peer for entities: %v", req.EntityTypes)

	resp, err := sh.syncEngine.GetDataForPull(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// MeshPush handles data push from other mesh nodes
func (sh *SyncHandler) MeshPush(w http.ResponseWriter, r *http.Request) {
	var data sync.MeshSyncResponse
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("Mesh Push: Received data from %s (products: %d, locations: %d)",
		data.NodeID, len(data.Products), len(data.Locations))

	// Apply updates using transaction
	tx := sh.db.DB.Begin()

	// Upsert products
	for _, p := range data.Products {
		if p.ID == 0 {
			continue
		}
		tx.Where("id = ?", p.ID).Assign(p).FirstOrCreate(&models.ProductProduct{})
	}

	// Upsert locations
	for _, l := range data.Locations {
		if l.ID == 0 {
			continue
		}
		tx.Where("id = ?", l.ID).Assign(l).FirstOrCreate(&models.StockLocation{})
	}

	// Upsert quants
	for _, q := range data.Quants {
		if q.ID == 0 {
			continue
		}
		tx.Where("id = ?", q.ID).Assign(q).FirstOrCreate(&models.StockQuant{})
	}

	// Upsert lots
	for _, lot := range data.Lots {
		if lot.ID == 0 {
			continue
		}
		tx.Where("id = ?", lot.ID).Assign(lot).FirstOrCreate(&models.StockLot{})
	}

	// Upsert packages
	for _, pkg := range data.Packages {
		if pkg.ID == 0 {
			continue
		}
		tx.Where("id = ?", pkg.ID).Assign(pkg).FirstOrCreate(&models.StockQuantPackage{})
	}

	// Upsert partners
	for _, partner := range data.Partners {
		if partner.ID == 0 {
			continue
		}
		tx.Where("id = ?", partner.ID).Assign(partner).FirstOrCreate(&models.ResPartner{})
	}

	if err := tx.Commit().Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Data received and applied",
	})
}

// TriggerMeshSync triggers synchronization with mesh nodes
func (sh *SyncHandler) TriggerMeshSync(w http.ResponseWriter, r *http.Request) {
	go func() {
		if err := sh.syncEngine.SyncWithRelay(); err != nil {
			log.Printf("Mesh Sync Error: %v", err)
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Mesh sync triggered",
		"status":  "processing",
	})
}

// Helper functions

func determineAction(localHash, remoteHash string) string {
	if localHash == "" {
		return "pull"
	}
	if remoteHash == "" {
		return "push"
	}
	return "conflict"
}
