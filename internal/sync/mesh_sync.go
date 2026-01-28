package sync

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/xelth-com/eckwmsgo/internal/mesh"
	"github.com/xelth-com/eckwmsgo/internal/models"
)

// MeshSyncRequest represents a request to sync data between mesh nodes
type MeshSyncRequest struct {
	EntityTypes []string   `json:"entity_types"`
	Since       *time.Time `json:"since,omitempty"`
	Limit       int        `json:"limit,omitempty"`
}

// MeshSyncResponse represents the response from a mesh sync request
type MeshSyncResponse struct {
	Products    []models.ProductProduct       `json:"products,omitempty"`
	Locations   []models.StockLocation        `json:"locations,omitempty"`
	Quants      []models.StockQuant           `json:"quants,omitempty"`
	Lots        []models.StockLot             `json:"lots,omitempty"`
	Packages    []models.StockQuantPackage    `json:"packages,omitempty"`
	Pickings    []models.StockPicking         `json:"pickings,omitempty"`
	Partners    []models.ResPartner           `json:"partners,omitempty"`
	Shipments   []models.StockPickingDelivery `json:"shipments,omitempty"`
	Tracking    []models.DeliveryTracking     `json:"tracking,omitempty"`
	Devices     []models.RegisteredDevice     `json:"devices,omitempty"`
	SyncHistory []models.SyncHistory          `json:"sync_history,omitempty"`
	SyncTime    time.Time                     `json:"sync_time"`
	NodeID      string                        `json:"node_id"`
}

// SyncWithRelay synchronizes data with mesh nodes
func (se *SyncEngine) SyncWithRelay() error {
	nodes := mesh.GlobalRegistry.GetNodes()
	if len(nodes) == 0 {
		log.Println("Mesh Sync: No nodes registered, skipping sync")
		return nil
	}

	role := se.GetRole()
	log.Printf("Mesh Sync: Starting sync as %s with %d known nodes", role, len(nodes))

	for _, node := range nodes {
		if !node.IsOnline {
			continue
		}

		// Skip self
		if node.InstanceID == se.instanceID {
			continue
		}

		// Peers pull from master AND push local data (shipments from scrapers)
		if role == RolePeer || role == RoleEdge {
			if node.Role == "master" {
				// Pull master data (products, locations, etc.)
				if err := se.pullFromNode(node); err != nil {
					log.Printf("Mesh Sync: Failed to pull from %s: %v", node.InstanceID, err)
				}
				// Push local shipment data to master (scraper results)
				if err := se.pushShipmentsToNode(node); err != nil {
					log.Printf("Mesh Sync: Failed to push shipments to %s: %v", node.InstanceID, err)
				}
			}
		} else if role == RoleMaster {
			// Master can push to peers if needed
			if node.Role == "peer" || node.Role == "edge" {
				log.Printf("Mesh Sync: Master ready to push to %s (on-demand)", node.InstanceID)
			}
		}
	}

	return nil
}

// pullFromNode pulls data from a specific mesh node
func (se *SyncEngine) pullFromNode(node *mesh.NodeInfo) error {
	log.Printf("Mesh Sync: Pulling data from %s (%s)", node.InstanceID, node.BaseURL)

	// Get last sync time for this node
	var syncMeta models.SyncMetadata
	se.db.DB.Where("instance_id = ?", node.InstanceID).First(&syncMeta)

	// Build request
	req := MeshSyncRequest{
		EntityTypes: []string{"products", "locations", "quants", "lots", "packages", "partners", "devices", "shipments", "tracking"},
		Limit:       1000,
	}
	if syncMeta.LastSyncAt != nil && !syncMeta.LastSyncAt.IsZero() {
		req.Since = syncMeta.LastSyncAt
	}

	body, _ := json.Marshal(req)

	// Generate auth token
	token, err := mesh.GenerateNodeToken(se.getMeshConfig())
	if err != nil {
		return fmt.Errorf("failed to generate token: %w", err)
	}

	// Make request
	url := node.BaseURL + "/api/mesh/pull"
	httpReq, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("pull failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response
	var syncResp MeshSyncResponse
	if err := json.NewDecoder(resp.Body).Decode(&syncResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Apply updates to local database
	if err := se.applyMeshUpdates(&syncResp); err != nil {
		return fmt.Errorf("failed to apply updates: %w", err)
	}

	// Update sync metadata
	syncTime := syncResp.SyncTime
	se.db.DB.Where("instance_id = ? AND entity_type = ?", node.InstanceID, "mesh").Assign(models.SyncMetadata{
		InstanceID:     node.InstanceID,
		EntityType:     "mesh",
		LastSyncAt:     &syncTime,
		LastSyncStatus: "success",
	}).FirstOrCreate(&models.SyncMetadata{})

	log.Printf("Mesh Sync: Successfully pulled from %s (products: %d, locations: %d, quants: %d, shipments: %d, tracking: %d)",
		node.InstanceID, len(syncResp.Products), len(syncResp.Locations), len(syncResp.Quants), len(syncResp.Shipments), len(syncResp.Tracking))

	return nil
}

// applyMeshUpdates applies updates received from another node
func (se *SyncEngine) applyMeshUpdates(resp *MeshSyncResponse) error {
	tx := se.db.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Upsert products
	for _, p := range resp.Products {
		if p.ID == 0 {
			continue
		}
		if err := tx.Where("id = ?", p.ID).Assign(p).FirstOrCreate(&models.ProductProduct{}).Error; err != nil {
			log.Printf("Mesh Sync: Failed to upsert product %d: %v", p.ID, err)
		}
	}

	// Upsert locations
	for _, l := range resp.Locations {
		if l.ID == 0 {
			continue
		}
		if err := tx.Where("id = ?", l.ID).Assign(l).FirstOrCreate(&models.StockLocation{}).Error; err != nil {
			log.Printf("Mesh Sync: Failed to upsert location %d: %v", l.ID, err)
		}
	}

	// Upsert quants
	for _, q := range resp.Quants {
		if q.ID == 0 {
			continue
		}
		if err := tx.Where("id = ?", q.ID).Assign(q).FirstOrCreate(&models.StockQuant{}).Error; err != nil {
			log.Printf("Mesh Sync: Failed to upsert quant %d: %v", q.ID, err)
		}
	}

	// Upsert lots
	for _, lot := range resp.Lots {
		if lot.ID == 0 {
			continue
		}
		if err := tx.Where("id = ?", lot.ID).Assign(lot).FirstOrCreate(&models.StockLot{}).Error; err != nil {
			log.Printf("Mesh Sync: Failed to upsert lot %d: %v", lot.ID, err)
		}
	}

	// Upsert packages
	for _, pkg := range resp.Packages {
		if pkg.ID == 0 {
			continue
		}
		if err := tx.Where("id = ?", pkg.ID).Assign(pkg).FirstOrCreate(&models.StockQuantPackage{}).Error; err != nil {
			log.Printf("Mesh Sync: Failed to upsert package %d: %v", pkg.ID, err)
		}
	}

	// Upsert partners
	for _, partner := range resp.Partners {
		if partner.ID == 0 {
			continue
		}
		if err := tx.Where("id = ?", partner.ID).Assign(partner).FirstOrCreate(&models.ResPartner{}).Error; err != nil {
			log.Printf("Mesh Sync: Failed to upsert partner %d: %v", partner.ID, err)
		}
	}

	// Upsert shipments
	for _, s := range resp.Shipments {
		if s.ID == 0 {
			continue
		}
		if err := tx.Where("id = ?", s.ID).Assign(s).FirstOrCreate(&models.StockPickingDelivery{}).Error; err != nil {
			log.Printf("Mesh Sync: Failed to upsert shipment %d: %v", s.ID, err)
		}
	}

	// Upsert tracking
	for _, t := range resp.Tracking {
		if t.ID == 0 {
			continue
		}
		if err := tx.Where("id = ?", t.ID).Assign(t).FirstOrCreate(&models.DeliveryTracking{}).Error; err != nil {
			log.Printf("Mesh Sync: Failed to upsert tracking %d: %v", t.ID, err)
		}
	}

	// Upsert sync history (logs from other nodes)
	for _, h := range resp.SyncHistory {
		if h.ID == 0 {
			continue
		}
		if err := tx.Where("id = ?", h.ID).Assign(h).FirstOrCreate(&models.SyncHistory{}).Error; err != nil {
			log.Printf("Mesh Sync: Failed to upsert sync_history %d: %v", h.ID, err)
		}
	}

	return tx.Commit().Error
}

// getMeshConfig returns config for generating mesh tokens
func (se *SyncEngine) getMeshConfig() *mesh.TokenConfig {
	return &mesh.TokenConfig{
		InstanceID: se.instanceID,
		MeshSecret: se.meshSecret,
		Role:       se.nodeRole,
		BaseURL:    se.baseURL,
	}
}

// GetDataForPull returns data for another node to pull
func (se *SyncEngine) GetDataForPull(req *MeshSyncRequest) (*MeshSyncResponse, error) {
	resp := &MeshSyncResponse{
		SyncTime: time.Now(),
		NodeID:   se.instanceID,
	}

	for _, entityType := range req.EntityTypes {
		switch entityType {
		case "products":
			var products []models.ProductProduct
			query := se.db.DB
			if req.Since != nil {
				query = query.Where("updated_at > ?", *req.Since)
			}
			if req.Limit > 0 {
				query = query.Limit(req.Limit)
			}
			if err := query.Find(&products).Error; err == nil {
				resp.Products = products
			}
		case "locations":
			var locations []models.StockLocation
			query := se.db.DB
			if req.Since != nil {
				query = query.Where("updated_at > ?", *req.Since)
			}
			if req.Limit > 0 {
				query = query.Limit(req.Limit)
			}
			if err := query.Find(&locations).Error; err == nil {
				resp.Locations = locations
			}
		case "quants":
			var quants []models.StockQuant
			query := se.db.DB
			if req.Since != nil {
				query = query.Where("updated_at > ?", *req.Since)
			}
			if req.Limit > 0 {
				query = query.Limit(req.Limit)
			}
			if err := query.Find(&quants).Error; err == nil {
				resp.Quants = quants
			}
		case "lots":
			var lots []models.StockLot
			query := se.db.DB
			if req.Since != nil {
				query = query.Where("updated_at > ?", *req.Since)
			}
			if req.Limit > 0 {
				query = query.Limit(req.Limit)
			}
			if err := query.Find(&lots).Error; err == nil {
				resp.Lots = lots
			}
		case "packages":
			var packages []models.StockQuantPackage
			query := se.db.DB
			if req.Since != nil {
				query = query.Where("updated_at > ?", *req.Since)
			}
			if req.Limit > 0 {
				query = query.Limit(req.Limit)
			}
			if err := query.Find(&packages).Error; err == nil {
				resp.Packages = packages
			}
		case "partners":
			var partners []models.ResPartner
			query := se.db.DB
			if req.Since != nil {
				query = query.Where("updated_at > ?", *req.Since)
			}
			if req.Limit > 0 {
				query = query.Limit(req.Limit)
			}
			if err := query.Find(&partners).Error; err == nil {
				resp.Partners = partners
			}
		case "shipments":
			var shipments []models.StockPickingDelivery
			query := se.db.DB
			if req.Since != nil {
				query = query.Where("updated_at > ?", *req.Since)
			}
			if req.Limit > 0 {
				query = query.Limit(req.Limit)
			}
			if err := query.Find(&shipments).Error; err == nil {
				resp.Shipments = shipments
			}
		case "tracking":
			// Tracking events use created_at (immutable records)
			var tracking []models.DeliveryTracking
			query := se.db.DB
			if req.Since != nil {
				query = query.Where("created_at > ?", *req.Since)
			}
			if req.Limit > 0 {
				query = query.Limit(req.Limit)
			}
			if err := query.Find(&tracking).Error; err == nil {
				resp.Tracking = tracking
			}
		case "sync_history":
			// Sync history logs - last 30 records, last 7 days
			var history []models.SyncHistory
			query := se.db.DB.Order("started_at DESC").Limit(30)
			since := time.Now().AddDate(0, 0, -7)
			query = query.Where("started_at > ?", since)
			if err := query.Find(&history).Error; err == nil {
				resp.SyncHistory = history
			}
		}
	}

	return resp, nil
}

// PushToNode pushes local data to a specific node
func (se *SyncEngine) PushToNode(node *mesh.NodeInfo, data *MeshSyncResponse) error {
	log.Printf("Mesh Sync: Pushing data to %s (%s)", node.InstanceID, node.BaseURL)

	body, _ := json.Marshal(data)

	token, err := mesh.GenerateNodeToken(se.getMeshConfig())
	if err != nil {
		return fmt.Errorf("failed to generate token: %w", err)
	}

	url := node.BaseURL + "/api/mesh/push"
	httpReq, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("push failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	log.Printf("Mesh Sync: Successfully pushed to %s", node.InstanceID)
	return nil
}

// pushShipmentsToNode pushes local data to master using Merkle Tree sync
func (se *SyncEngine) pushShipmentsToNode(node *mesh.NodeInfo) error {
	log.Printf("Merkle Sync: Starting sync with %s (%s)", node.InstanceID, node.BaseURL)

	token, err := mesh.GenerateNodeToken(se.getMeshConfig())
	if err != nil {
		return fmt.Errorf("failed to generate token: %w", err)
	}
	client := &http.Client{Timeout: 30 * time.Second}

	// Sync shipments using Merkle Tree
	shipments := se.merkleSync(node, token, client, "shipment")

	// Sync tracking using Merkle Tree
	var tracking []models.DeliveryTracking
	trackingIDs := se.merkleSync(node, token, client, "tracking")
	if len(trackingIDs) > 0 {
		se.db.DB.Where("id IN ?", trackingIDs).Find(&tracking)
	}

	// Sync devices using Merkle Tree
	var devices []models.RegisteredDevice
	deviceIDs := se.merkleSync(node, token, client, "device")
	if len(deviceIDs) > 0 {
		se.db.DB.Unscoped().Where("device_id IN ?", deviceIDs).Find(&devices)
	}

	// Sync sync_history (by ID, no checksums)
	var syncHistory []models.SyncHistory
	syncHistoryIDs := se.syncHistoryNegotiate(node, token, client)
	if len(syncHistoryIDs) > 0 {
		se.db.DB.Where("id IN ?", syncHistoryIDs).Find(&syncHistory)
	}

	// Fetch actual shipment data
	var shipmentData []models.StockPickingDelivery
	if len(shipments) > 0 {
		se.db.DB.Where("id IN ?", shipments).Find(&shipmentData)
	}

	// Build Push Payload
	data := &MeshSyncResponse{
		NodeID:      se.instanceID,
		SyncTime:    time.Now(),
		Shipments:   shipmentData,
		Tracking:    tracking,
		Devices:     devices,
		SyncHistory: syncHistory,
	}

	log.Printf("Merkle Sync: Pushing %d shipments, %d tracking, %d devices, %d sync_history",
		len(shipmentData), len(tracking), len(devices), len(syncHistory))

	// Skip push if nothing to send
	if len(shipmentData) == 0 && len(tracking) == 0 && len(devices) == 0 && len(syncHistory) == 0 {
		log.Printf("Merkle Sync: Nothing to push to %s", node.InstanceID)
		return nil
	}

	// Push Data
	if err := se.PushToNode(node, data); err != nil {
		return err
	}

	return nil
}

// merkleSync performs Merkle Tree based sync for an entity type
// Returns list of entity IDs that need to be pushed to remote
func (se *SyncEngine) merkleSync(node *mesh.NodeInfo, token string, client *http.Client, entityType string) []string {
	// 1. Build local Merkle tree
	localTree := NewMerkleTree(se.db, entityType)
	if err := localTree.Build(); err != nil {
		log.Printf("Merkle Sync: Failed to build local tree for %s: %v", entityType, err)
		return nil
	}

	localRoot := localTree.GetRootHash()
	if localRoot == "" {
		log.Printf("Merkle Sync: No local %s data", entityType)
		return nil
	}

	// 2. Get remote Merkle tree root
	reqBody := MerkleTreeRequest{EntityType: entityType, Level: 0}
	reqJSON, _ := json.Marshal(reqBody)

	url := node.BaseURL + "/api/mesh/merkle"
	httpReq, _ := http.NewRequest("POST", url, bytes.NewBuffer(reqJSON))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(httpReq)
	if err != nil {
		log.Printf("Merkle Sync: Failed to get remote tree for %s: %v", entityType, err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Merkle Sync: Remote returned %d for %s", resp.StatusCode, entityType)
		return nil
	}

	var remoteTree MerkleTreeResponse
	if err := json.NewDecoder(resp.Body).Decode(&remoteTree); err != nil {
		log.Printf("Merkle Sync: Failed to decode remote tree for %s: %v", entityType, err)
		return nil
	}

	// 3. Compare root hashes
	if localRoot == remoteTree.Hash {
		log.Printf("Merkle Sync: %s in sync (root=%s)", entityType, localRoot[:8])
		return nil
	}

	log.Printf("Merkle Sync: %s differs (local=%s, remote=%s)", entityType, localRoot[:8], remoteTree.Hash[:8])

	// 4. Find differing buckets
	localBuckets := localTree.GetBucketHashes()
	var differingBuckets []string

	for bucket, localHash := range localBuckets {
		remoteHash, exists := remoteTree.Children[bucket]
		if !exists || localHash != remoteHash {
			differingBuckets = append(differingBuckets, bucket)
		}
	}

	// Also check for buckets we have that remote doesn't
	for bucket := range localBuckets {
		if _, exists := remoteTree.Children[bucket]; !exists {
			// Already added above
		}
	}

	log.Printf("Merkle Sync: %s has %d differing buckets", entityType, len(differingBuckets))

	// 5. For each differing bucket, get entity-level comparison
	var neededIDs []string
	for _, bucket := range differingBuckets {
		// Get local entities in bucket
		localEntities, _ := localTree.GetBucketEntities(bucket)

		// Get remote entities in bucket
		bucketReq := MerkleTreeRequest{EntityType: entityType, Level: 1, BucketKey: bucket}
		bucketJSON, _ := json.Marshal(bucketReq)

		httpReq, _ := http.NewRequest("POST", url, bytes.NewBuffer(bucketJSON))
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+token)

		resp, err := client.Do(httpReq)
		if err != nil {
			// Can't get remote bucket, assume all local entities need sync
			for id := range localEntities {
				neededIDs = append(neededIDs, id)
			}
			continue
		}

		var remoteBucket MerkleTreeResponse
		json.NewDecoder(resp.Body).Decode(&remoteBucket)
		resp.Body.Close()

		// Compare entity hashes
		for id, localHash := range localEntities {
			remoteHash, exists := remoteBucket.Children[id]
			if !exists || localHash != remoteHash {
				neededIDs = append(neededIDs, id)
			}
		}
	}

	log.Printf("Merkle Sync: %s needs to push %d entities", entityType, len(neededIDs))
	return neededIDs
}

// syncHistoryNegotiate handles sync_history sync (by ID only, no checksums)
func (se *SyncEngine) syncHistoryNegotiate(node *mesh.NodeInfo, token string, client *http.Client) []string {
	var localHistory []models.SyncHistory
	weekAgo := time.Now().AddDate(0, 0, -7)
	se.db.DB.Where("started_at > ?", weekAgo).Order("started_at DESC").Limit(30).Find(&localHistory)

	if len(localHistory) == 0 {
		return nil
	}

	// Build negotiation request
	items := make([]ChecksumItem, len(localHistory))
	for i, h := range localHistory {
		items[i] = ChecksumItem{EntityID: fmt.Sprintf("%d", h.ID), Hash: ""}
	}

	reqBody := MeshNegotiationRequest{EntityType: "sync_history", Items: items}
	reqJSON, _ := json.Marshal(reqBody)

	url := node.BaseURL + "/api/mesh/negotiate"
	httpReq, _ := http.NewRequest("POST", url, bytes.NewBuffer(reqJSON))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var negoResp MeshNegotiationResponse
	if json.NewDecoder(resp.Body).Decode(&negoResp) == nil {
		log.Printf("Merkle Sync: sync_history needs %d records", len(negoResp.RequestIDs))
		return negoResp.RequestIDs
	}

	return nil
}
