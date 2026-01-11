package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/dmytrosurovtsev/eckwmsgo/internal/database"
	"github.com/dmytrosurovtsev/eckwmsgo/internal/sync"
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
	var req struct {
		EntityTypes []string `json:"entity_types"`
		Since       *string  `json:"since"` // ISO timestamp
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// TODO: Implement pull logic

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Pull updates initiated",
		"status":  "processing",
	})
}

// PushUpdates pushes local updates to remote
func (sh *SyncHandler) PushUpdates(w http.ResponseWriter, r *http.Request) {
	var req struct {
		EntityTypes []string `json:"entity_types"`
		Since       *string  `json:"since"` // ISO timestamp
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// TODO: Implement push logic

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Push updates initiated",
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
