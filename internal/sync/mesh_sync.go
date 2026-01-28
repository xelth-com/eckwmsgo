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
	Products  []models.ProductProduct       `json:"products,omitempty"`
	Locations []models.StockLocation        `json:"locations,omitempty"`
	Quants    []models.StockQuant           `json:"quants,omitempty"`
	Lots      []models.StockLot             `json:"lots,omitempty"`
	Packages  []models.StockQuantPackage    `json:"packages,omitempty"`
	Pickings  []models.StockPicking         `json:"pickings,omitempty"`
	Partners  []models.ResPartner           `json:"partners,omitempty"`
	Shipments []models.StockPickingDelivery `json:"shipments,omitempty"`
	Tracking  []models.DeliveryTracking     `json:"tracking,omitempty"`
	Devices   []models.RegisteredDevice     `json:"devices,omitempty"`
	SyncTime  time.Time                     `json:"sync_time"`
	NodeID    string                        `json:"node_id"`
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

// pushShipmentsToNode pushes local shipment data (from scrapers) to master
// Uses Checksum Negotiation instead of timestamp-based sync
func (se *SyncEngine) pushShipmentsToNode(node *mesh.NodeInfo) error {
	log.Printf("Mesh Sync: Negotiating shipments with %s (%s)", node.InstanceID, node.BaseURL)

	// 1. Get ALL local checksums for shipments from entity_checksums table
	var checksums []models.EntityChecksum
	if err := se.db.DB.Where("entity_type = ?", "shipment").Find(&checksums).Error; err != nil {
		return fmt.Errorf("failed to read local checksums: %w", err)
	}

	if len(checksums) == 0 {
		log.Printf("Mesh Sync: No local shipments to sync.")
		return nil
	}

	// 2. Build Negotiation Request (Manifest)
	items := make([]ChecksumItem, len(checksums))
	for i, c := range checksums {
		items[i] = ChecksumItem{
			EntityID: c.EntityID,
			Hash:     c.FullHash,
		}
	}

	reqBody := MeshNegotiationRequest{
		EntityType: "shipment",
		Items:      items,
	}

	reqJSON, _ := json.Marshal(reqBody)

	// 3. Send Manifest to Remote Node
	token, err := mesh.GenerateNodeToken(se.getMeshConfig())
	if err != nil {
		return fmt.Errorf("failed to generate token: %w", err)
	}

	url := node.BaseURL + "/api/mesh/negotiate"
	httpReq, _ := http.NewRequest("POST", url, bytes.NewBuffer(reqJSON))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("negotiation request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("negotiation failed HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// 4. Parse Response (List of IDs the remote needs)
	var negoResp MeshNegotiationResponse
	if err := json.NewDecoder(resp.Body).Decode(&negoResp); err != nil {
		return fmt.Errorf("failed to decode negotiation response: %w", err)
	}

	neededIDs := negoResp.RequestIDs
	log.Printf("Mesh Sync: Negotiation complete. Remote needs %d shipments out of %d offered.", len(neededIDs), len(items))

	if len(neededIDs) == 0 {
		return nil // Nothing to sync
	}

	// 5. Fetch Full Data for Needed IDs
	var shipments []models.StockPickingDelivery
	if err := se.db.DB.Where("id IN ?", neededIDs).Find(&shipments).Error; err != nil {
		return fmt.Errorf("failed to fetch shipment data: %w", err)
	}

	// Also fetch recent Tracking and Devices (time-based for now, negotiate separately later)
	var tracking []models.DeliveryTracking
	var devices []models.RegisteredDevice

	if len(shipments) > 0 {
		yesterday := time.Now().Add(-24 * time.Hour)
		se.db.DB.Where("created_at > ?", yesterday).Find(&tracking)
		se.db.DB.Unscoped().Where("updated_at > ?", yesterday).Find(&devices)
	}

	// 6. Build Push Payload
	data := &MeshSyncResponse{
		NodeID:    se.instanceID,
		SyncTime:  time.Now(),
		Shipments: shipments,
		Tracking:  tracking,
		Devices:   devices,
	}

	log.Printf("Mesh Sync: Pushing %d shipments, %d tracking, %d devices", len(shipments), len(tracking), len(devices))

	// 7. Push Data
	if err := se.PushToNode(node, data); err != nil {
		return err
	}

	// Update metadata (informational only, not used for sync logic anymore)
	now := time.Now()
	var syncMeta models.SyncMetadata
	se.db.DB.Where("instance_id = ? AND entity_type = ?", node.InstanceID, "shipments_push").Assign(models.SyncMetadata{
		InstanceID: node.InstanceID,
		EntityType: "shipments_push",
		LastSyncAt: &now,
	}).FirstOrCreate(&syncMeta)

	return nil
}
