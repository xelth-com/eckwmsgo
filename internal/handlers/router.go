package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
	"github.com/xelth-com/eckwmsgo/internal/ai"
	"github.com/xelth-com/eckwmsgo/internal/database"
	"github.com/xelth-com/eckwmsgo/internal/middleware"
	"github.com/xelth-com/eckwmsgo/internal/models"
	deliveryService "github.com/xelth-com/eckwmsgo/internal/services/delivery"
	odooService "github.com/xelth-com/eckwmsgo/internal/services/odoo"
	"github.com/xelth-com/eckwmsgo/internal/sync"
	"github.com/xelth-com/eckwmsgo/internal/websocket"
	"github.com/xelth-com/eckwmsgo/web"
)

// Router wraps the mux router and database
type Router struct {
	*mux.Router
	db              *database.DB
	hub             *websocket.Hub
	odooService     interface{}              // Set via SetOdooService for Odoo sync routes
	deliveryService *deliveryService.Service // Set via SetDeliveryService for delivery routes
	syncEngine      *sync.SyncEngine         // Set via SetSyncEngine for mesh sync routes
	aiClient        *ai.GeminiClient         // Set via SetAIClient for AI features
}

// NewRouter creates a new HTTP router with all routes
func NewRouter(db *database.DB) *Router {
	// Initialize WebSocket Hub
	hub := websocket.NewHub()
	go hub.Run()

	r := &Router{
		Router: mux.NewRouter(),
		db:     db,
		hub:    hub,
	}

	// Detect path prefix from environment (e.g. "/E")
	urlPrefix := os.Getenv("HTTP_PATH_PREFIX")
	// Ensure prefix starts with / and has no trailing / if set
	if urlPrefix != "" {
		if !strings.HasPrefix(urlPrefix, "/") {
			urlPrefix = "/" + urlPrefix
		}
		urlPrefix = strings.TrimRight(urlPrefix, "/")
		// Convert to lowercase to match CaseInsensitiveMiddleware behavior
		urlPrefix = strings.ToLower(urlPrefix)
	}

	// Helper to register simple routes (adds both /path and /PREFIX/path)
	handle := func(path string, f func(http.ResponseWriter, *http.Request), methods ...string) {
		r.HandleFunc(path, f).Methods(methods...)
		if urlPrefix != "" {
			r.HandleFunc(urlPrefix+path, f).Methods(methods...)
		}
	}

	// 1. Health check endpoint (Public)
	handle("/health", r.healthCheck, "GET")

	// Mesh routes (Public - uses JWT token auth)
	handle("/mesh/handshake", r.handleHandshake, "POST")
	handle("/mesh/nodes", r.listMeshNodes, "GET")

	// 2. Auth routes (Public)
	handle("/auth/login", r.login, "POST")
	handle("/auth/register", r.register, "POST")
	handle("/auth/logout", r.logout, "POST")

	// 3. Public Device Registration (Specific /api route - MUST BE BEFORE generic /api)
	handle("/api/internal/register-device", r.registerDevice, "POST")

	// 4. Image Upload Stub (Specific /api route)
	r.registerUploadRoutes(urlPrefix)

	// 5. Register Specific Route Groups (Protected)
	// These define their own subrouters. Since they are specific PathPrefixes,
	// they should be registered before the generic /api catch-all.
	// r.registerRMARoutes(urlPrefix) // TODO: Implement RMA handlers
	r.registerRackRoutes(urlPrefix) // Must be before warehouse to avoid /racks being caught
	r.registerWarehouseRoutes(urlPrefix)
	r.registerItemsRoutes(urlPrefix)
	r.registerSetupRoutes(urlPrefix) // Protected parts of setup
	r.registerPrintRoutes(urlPrefix)
	r.registerAIRoutes(urlPrefix, db)
	r.registerAdminRoutes(urlPrefix) // Admin endpoints for device management

	// 6. Generic API endpoints (Protected)
	// This captures remaining /api/* requests like /api/status or /api/scan
	paths := []string{"/api"}
	if urlPrefix != "" {
		paths = append(paths, urlPrefix+"/api")
	}
	for _, p := range paths {
		api := r.PathPrefix(p).Subrouter()
		api.Use(middleware.AuthMiddleware)

		// System endpoints
		api.HandleFunc("/status", r.getStatus).Methods("GET")

		// Universal Scan Endpoint
		api.HandleFunc("/scan", r.handleScan).Methods("POST")

		// AI Feedback Endpoint
		api.HandleFunc("/ai/respond", r.handleAiRespond).Methods("POST")
	}

	// WebSocket endpoint (needs GET method for upgrade handshake)
	handle("/ws", func(w http.ResponseWriter, req *http.Request) {
		websocket.ServeWs(hub, w, req)
	}, "GET")

	// --- Static Files (Svelte Frontend) ---
	assets, err := web.GetFileSystem()
	if err != nil {
		publicDir := os.Getenv("FRONTEND_DIR")
		if publicDir == "" {
			publicDir = "web/build"
		}
		assets = os.DirFS(publicDir)
	}

	spaHandler := http.FileServer(http.FS(assets))

	// Static file handler for /i/ paths (must be BEFORE SPA catch-all)
	staticHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		path := req.URL.Path

		// Strip prefix if present
		if urlPrefix != "" {
			urlPrefixUpper := strings.ToUpper(urlPrefix)
			if strings.HasPrefix(path, urlPrefix) {
				path = strings.TrimPrefix(path, urlPrefix)
			} else if strings.HasPrefix(path, urlPrefixUpper) {
				path = strings.TrimPrefix(path, urlPrefixUpper)
			}
		}

		// Serve static file
		req.URL.Path = path
		spaHandler.ServeHTTP(w, req)
	})

	// Register static file handler first (higher priority)
	staticPaths := []string{"/i"}
	if urlPrefix != "" {
		// Add both lowercase and uppercase prefix for case-insensitive matching
		staticPaths = append(staticPaths, urlPrefix+"/i")
		urlPrefixUpper := strings.ToUpper(urlPrefix)
		if urlPrefixUpper != urlPrefix {
			staticPaths = append(staticPaths, urlPrefixUpper+"/i")
		}
	}
	for _, sp := range staticPaths {
		r.PathPrefix(sp).Handler(staticHandler)
	}

	// SPA Handler Logic - register as catch-all with matcher that excludes API routes
	r.PathPrefix("/").MatcherFunc(func(req *http.Request, rm *mux.RouteMatch) bool {
		path := req.URL.Path

		// Check if this is an API path (with or without prefix)
		if strings.HasPrefix(path, "/api") || strings.HasPrefix(path, "/auth") ||
			strings.HasPrefix(path, "/ws") || strings.HasPrefix(path, "/health") ||
			strings.HasPrefix(path, "/mesh") || strings.HasPrefix(path, "/i") {
			return false // Don't match - let API handlers handle it
		}

		// Check for prefixed API paths (case insensitive prefix check)
		if urlPrefix != "" {
			// Check both lowercase and uppercase prefix because CaseInsensitiveMiddleware
			// doesn't convert paths with /i/ to lowercase
			urlPrefixUpper := strings.ToUpper(urlPrefix)
			if strings.HasPrefix(path, urlPrefix) || strings.HasPrefix(path, urlPrefixUpper) {
				pathWithoutPrefix := path
				if strings.HasPrefix(path, urlPrefix) {
					pathWithoutPrefix = strings.TrimPrefix(path, urlPrefix)
				} else {
					pathWithoutPrefix = strings.TrimPrefix(path, urlPrefixUpper)
				}

				if strings.HasPrefix(pathWithoutPrefix, "/api") ||
					strings.HasPrefix(pathWithoutPrefix, "/auth") ||
					strings.HasPrefix(pathWithoutPrefix, "/ws") ||
					strings.HasPrefix(pathWithoutPrefix, "/health") ||
					strings.HasPrefix(pathWithoutPrefix, "/mesh") ||
					strings.HasPrefix(pathWithoutPrefix, "/i") {
					return false // Don't match - let API handlers handle it
				}
			}
		}

		return true // Match - this is for SPA handler
	}).Handler(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		originalPath := req.URL.Path
		path := originalPath

		// If prefix is configured, redirect non-prefixed paths to prefixed version
		if urlPrefix != "" {
			urlPrefixUpper := strings.ToUpper(urlPrefix)
			hasPrefix := strings.HasPrefix(originalPath, urlPrefix) || strings.HasPrefix(originalPath, urlPrefixUpper)

			if !hasPrefix {
				// Redirect to prefixed path (use uppercase for user-friendly URLs)
				redirectPath := urlPrefixUpper + originalPath
				http.Redirect(w, req, redirectPath, http.StatusFound)
				return
			}

			// Strip prefix for static file lookup (case insensitive)
			if strings.HasPrefix(originalPath, urlPrefix) {
				path = strings.TrimPrefix(originalPath, urlPrefix)
			} else if strings.HasPrefix(originalPath, urlPrefixUpper) {
				path = strings.TrimPrefix(originalPath, urlPrefixUpper)
			}
			if path == "" {
				path = "/"
			}
		}

		// Serve static files or SPA
		// Files with extension or /i/ path get served as-is
		if strings.HasPrefix(path, "/i") || strings.Contains(path, ".") {
			// For static files, modify req.URL.Path to the stripped version
			req.URL.Path = path
			spaHandler.ServeHTTP(w, req)
			return
		}

		// Otherwise serve index.html (SPA)
		req.URL.Path = "/"
		spaHandler.ServeHTTP(w, req)
	}))

	return r
}

// healthCheck returns the health status of the API
func (r *Router) healthCheck(w http.ResponseWriter, req *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"server": "local",
	})
}

// getStatus returns the current status
func (r *Router) getStatus(w http.ResponseWriter, req *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{
		"status":  "running",
		"version": "1.0.0",
	})
}

// respondJSON sends a JSON response
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// respondError sends an error response
func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{
		"error": message,
	})
}

// Handler returns the router wrapped with case-insensitive middleware
// This allows API endpoints to be accessed regardless of case
// Example: /API/status and /api/status both work
func (r *Router) Handler() http.Handler {
	return middleware.CaseInsensitiveMiddleware(r.Router)
}

// registerOrdersRoutes registers order-related routes with optional prefix
func (r *Router) registerOrdersRoutes(prefix string) {
	paths := []string{"/api/orders"}
	if prefix != "" {
		paths = append(paths, prefix+"/api/orders")
	}

	for _, p := range paths {
		orders := r.PathPrefix(p).Subrouter()
		orders.Use(middleware.AuthMiddleware)
		orders.HandleFunc("", r.listOrders).Methods("GET")
		orders.HandleFunc("", r.createOrder).Methods("POST")
		orders.HandleFunc("/{id}", r.getOrder).Methods("GET")
		orders.HandleFunc("/{id}", r.updateOrder).Methods("PUT")
		orders.HandleFunc("/{id}", r.deleteOrder).Methods("DELETE")
	}
}

// registerWarehouseRoutes registers warehouse-related routes with optional prefix
func (r *Router) registerWarehouseRoutes(prefix string) {
	paths := []string{"/api/warehouse"}
	if prefix != "" {
		paths = append(paths, prefix+"/api/warehouse")
	}

	for _, p := range paths {
		wh := r.PathPrefix(p).Subrouter()
		wh.Use(middleware.AuthMiddleware)
		wh.HandleFunc("", r.listWarehouses).Methods("GET")
		wh.HandleFunc("", r.createWarehouse).Methods("POST")

		// Location search (New)
		wh.HandleFunc("/locations/search", r.searchLocations).Methods("GET")

		// Racks (must be before /{id} to prevent /racks being caught as an id)
		wh.HandleFunc("/racks", r.listRacks).Methods("GET")
		wh.HandleFunc("/racks", r.createRack).Methods("POST")
		wh.HandleFunc("/racks/{id}", r.updateRack).Methods("PUT")
		wh.HandleFunc("/racks/{id}", r.deleteRack).Methods("DELETE")

		// Single warehouse and map
		wh.HandleFunc("/{id}", r.getWarehouse).Methods("GET")
		wh.HandleFunc("/{id}/map", r.getWarehouseMap).Methods("GET") // New Map Endpoint
	}
}

// registerItemsRoutes registers item-related routes with optional prefix
func (r *Router) registerItemsRoutes(prefix string) {
	paths := []string{"/api/items"}
	if prefix != "" {
		paths = append(paths, prefix+"/api/items")
	}

	for _, p := range paths {
		items := r.PathPrefix(p).Subrouter()
		items.Use(middleware.AuthMiddleware)
		items.HandleFunc("", r.listItems).Methods("GET")
		items.HandleFunc("", r.createItem).Methods("POST")
		items.HandleFunc("/{id}", r.getItem).Methods("GET")
		items.HandleFunc("/{id}", r.updateItem).Methods("PUT")
	}
}

// registerSetupRoutes registers setup and device-related routes with optional prefix
func (r *Router) registerSetupRoutes(prefix string) {
	paths := []string{"/api/internal"}
	if prefix != "" {
		paths = append(paths, prefix+"/api/internal")
	}

	for _, p := range paths {
		// Protected internal routes
		setup := r.PathPrefix(p).Subrouter()
		setup.Use(middleware.AuthMiddleware)
		setup.HandleFunc("/pairing-qr", r.generatePairingQR).Methods("GET")
	}
}

// registerPrintRoutes registers print-related routes with optional prefix
func (r *Router) registerPrintRoutes(prefix string) {
	paths := []string{"/api/print"}
	if prefix != "" {
		paths = append(paths, prefix+"/api/print")
	}

	for _, p := range paths {
		print := r.PathPrefix(p).Subrouter()
		print.Use(middleware.AuthMiddleware)
		print.HandleFunc("/labels", r.generateLabels).Methods("POST")
	}
}

// registerUploadRoutes registers image upload routes with optional prefix
func (r *Router) registerUploadRoutes(prefix string) {
	paths := []string{"/api/upload/image"}
	if prefix != "" {
		paths = append(paths, prefix+"/api/upload/image")
	}

	for _, p := range paths {
		// Stub handler - just returns success
		r.HandleFunc(p, r.handleImageUpload).Methods("POST")
	}
}

// handleImageUpload is a stub that returns success
func (r *Router) handleImageUpload(w http.ResponseWriter, req *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{
		"status":  "uploaded",
		"url":     "/storage/placeholder.jpg",
		"message": "Image upload stub successful",
	})
}

// registerAIRoutes registers AI agent routes with optional prefix
func (r *Router) registerAIRoutes(prefix string, db *database.DB) {
	// AI Agent routes (protected with AI auth middleware)
	agentPaths := []string{"/api/ai"}
	if prefix != "" {
		agentPaths = append(agentPaths, prefix+"/api/ai")
	}
	for _, p := range agentPaths {
		aiAgent := r.PathPrefix(p).Subrouter()
		aiAgent.Use(middleware.AIAuthMiddleware(db))
		aiAgent.HandleFunc("/execute", r.executeAIFunction).Methods("POST")
		aiAgent.HandleFunc("/functions", r.listAIFunctions).Methods("GET")
		aiAgent.HandleFunc("/status", r.getAIAgentStatus).Methods("GET")
		aiAgent.HandleFunc("/permissions", r.getAIAgentPermissions).Methods("GET")
	}

	// AI Admin routes (protected with regular user auth - admin only)
	adminPaths := []string{"/api/admin/ai"}
	if prefix != "" {
		adminPaths = append(adminPaths, prefix+"/api/admin/ai")
	}
	for _, p := range adminPaths {
		aiAdmin := r.PathPrefix(p).Subrouter()
		aiAdmin.Use(middleware.AuthMiddleware)
		aiAdmin.HandleFunc("/agents", r.listAIAgents).Methods("GET")
		aiAdmin.HandleFunc("/agents", r.createAIAgent).Methods("POST")
		aiAdmin.HandleFunc("/agents/{agent_id}/status", r.updateAIAgentStatus).Methods("PUT")
		aiAdmin.HandleFunc("/agents/{agent_id}/permissions", r.grantAIPermission).Methods("POST")
		aiAdmin.HandleFunc("/agents/{agent_id}/permissions", r.revokeAIPermission).Methods("DELETE")
		aiAdmin.HandleFunc("/agents/{agent_id}/audit", r.getAIAuditLogs).Methods("GET")
	}
}

// registerRackRoutes registers standalone rack routes
func (r *Router) registerRackRoutes(prefix string) {
	paths := []string{"/api/warehouse/racks"}
	if prefix != "" {
		paths = append(paths, prefix+"/api/warehouse/racks")
	}

	for _, p := range paths {
		racks := r.PathPrefix(p).Subrouter()
		racks.Use(middleware.AuthMiddleware)
		racks.HandleFunc("", r.listRacks).Methods("GET")
		racks.HandleFunc("", r.createRack).Methods("POST")
		racks.HandleFunc("/{id}", r.updateRack).Methods("PUT")
		racks.HandleFunc("/{id}", r.deleteRack).Methods("DELETE")
	}
}

// SetOdooService sets the Odoo sync service and registers its routes
func (r *Router) SetOdooService(service interface{}) {
	r.odooService = service

	urlPrefix := os.Getenv("HTTP_PATH_PREFIX")
	if urlPrefix != "" {
		if !strings.HasPrefix(urlPrefix, "/") {
			urlPrefix = "/" + urlPrefix
		}
		urlPrefix = strings.TrimRight(strings.ToLower(urlPrefix), "/")
	}

	r.registerOdooRoutes(urlPrefix, service)
}

// registerOdooRoutes registers Odoo sync API routes
func (r *Router) registerOdooRoutes(prefix string, svc interface{}) {
	if svc == nil {
		return
	}

	paths := []string{"/api/odoo"}
	if prefix != "" {
		paths = append(paths, prefix+"/api/odoo")
	}

	for _, p := range paths {
		odoo := r.PathPrefix(p).Subrouter()
		odoo.Use(middleware.AuthMiddleware)
		odoo.HandleFunc("/sync/trigger", r.triggerOdooSync).Methods("POST")
		odoo.HandleFunc("/sync/status", r.getOdooSyncStatus).Methods("GET")
		odoo.HandleFunc("/pickings", r.listOdooPickings).Methods("GET")
		odoo.HandleFunc("/pickings/{id}", r.getOdooPicking).Methods("GET")
	}
}

func (r *Router) triggerOdooSync(w http.ResponseWriter, req *http.Request) {
	if r.odooService == nil {
		http.Error(w, "Odoo sync not configured", http.StatusServiceUnavailable)
		return
	}

	service := r.odooService.(*odooService.SyncService)
	go service.TriggerManualSync()

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Odoo sync started in background",
	})
}

func (r *Router) getOdooSyncStatus(w http.ResponseWriter, req *http.Request) {
	var productCount, locationCount, pickingCount, quantCount int64

	r.db.Model(&models.ProductProduct{}).Count(&productCount)
	r.db.Model(&models.StockLocation{}).Count(&locationCount)
	r.db.Model(&models.StockPicking{}).Count(&pickingCount)
	r.db.Model(&models.StockQuant{}).Count(&quantCount)

	var lastProduct models.ProductProduct
	var lastLocation models.StockLocation
	var lastPicking models.StockPicking

	r.db.Order("last_synced_at DESC").First(&lastProduct)
	r.db.Order("last_synced_at DESC").First(&lastLocation)
	r.db.Order("scheduled_date DESC").First(&lastPicking)

	status := map[string]interface{}{
		"products": map[string]interface{}{
			"count":       productCount,
			"last_synced": lastProduct.LastSyncedAt,
		},
		"locations": map[string]interface{}{
			"count":       locationCount,
			"last_synced": lastLocation.LastSyncedAt,
		},
		"pickings": map[string]interface{}{
			"count":         pickingCount,
			"last_received": lastPicking.ScheduledDate,
		},
		"quants": map[string]interface{}{
			"count": quantCount,
		},
	}

	respondJSON(w, http.StatusOK, status)
}

func (r *Router) listOdooPickings(w http.ResponseWriter, req *http.Request) {
	state := req.URL.Query().Get("state")
	limit := 100

	query := r.db.Model(&models.StockPicking{})
	if state != "" {
		query = query.Where("state = ?", state)
	}

	var pickings []models.StockPicking
	if err := query.Order("scheduled_date DESC").Limit(limit).Find(&pickings).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch pickings")
		return
	}

	respondJSON(w, http.StatusOK, pickings)
}

func (r *Router) getOdooPicking(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	pickingID := vars["id"]

	var picking models.StockPicking
	if err := r.db.First(&picking, pickingID).Error; err != nil {
		respondError(w, http.StatusNotFound, "Picking not found")
		return
	}

	var moveLines []models.StockMoveLine
	r.db.Where("picking_id = ?", pickingID).Find(&moveLines)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"picking":    picking,
		"move_lines": moveLines,
	})
}

// SetDeliveryService sets the delivery service and registers delivery routes
func (r *Router) SetDeliveryService(svc *deliveryService.Service) {
	r.deliveryService = svc

	urlPrefix := os.Getenv("HTTP_PATH_PREFIX")
	if urlPrefix != "" {
		if !strings.HasPrefix(urlPrefix, "/") {
			urlPrefix = "/" + urlPrefix
		}
		urlPrefix = strings.TrimRight(strings.ToLower(urlPrefix), "/")
	}

	r.registerDeliveryRoutes(urlPrefix, svc)
}

// registerDeliveryRoutes registers delivery API routes
func (r *Router) registerDeliveryRoutes(prefix string, svc *deliveryService.Service) {
	if svc == nil {
		return
	}

	paths := []string{"/api/delivery"}
	if prefix != "" {
		paths = append(paths, prefix+"/api/delivery")
	}

	for _, p := range paths {
		delivery := r.PathPrefix(p).Subrouter()
		delivery.Use(middleware.AuthMiddleware)

		// Provider configuration check
		delivery.HandleFunc("/config", r.getDeliveryConfig).Methods("GET")

		// Shipment management
		delivery.HandleFunc("/shipments", r.createShipment).Methods("POST")
		delivery.HandleFunc("/shipments", r.listShipments).Methods("GET")
		delivery.HandleFunc("/shipments/{id}", r.getShipment).Methods("GET")
		delivery.HandleFunc("/shipments/{id}/cancel", r.cancelShipment).Methods("POST")

		// Import from OPAL
		delivery.HandleFunc("/import/opal", r.triggerOpalImport).Methods("POST")

		// Import from DHL
		delivery.HandleFunc("/import/dhl", r.triggerDhlImport).Methods("POST")

		// Carrier management
		delivery.HandleFunc("/carriers", r.listCarriers).Methods("GET")
		delivery.HandleFunc("/carriers", r.createCarrier).Methods("POST")
		delivery.HandleFunc("/carriers/{id}", r.getCarrier).Methods("GET")
		delivery.HandleFunc("/carriers/{id}/toggle", r.toggleCarrier).Methods("POST")
	}
}

// Delivery handlers

func (r *Router) getDeliveryConfig(w http.ResponseWriter, req *http.Request) {
	// Check for credentials in env to determine which providers are configured
	config := map[string]bool{
		"opal": os.Getenv("OPAL_USERNAME") != "" && os.Getenv("OPAL_PASSWORD") != "",
		"dhl":  os.Getenv("DHL_USERNAME") != "" && os.Getenv("DHL_PASSWORD") != "",
	}

	respondJSON(w, http.StatusOK, config)
}

func (r *Router) createShipment(w http.ResponseWriter, req *http.Request) {
	if r.deliveryService == nil {
		respondError(w, http.StatusServiceUnavailable, "Delivery service not configured")
		return
	}

	var reqBody struct {
		PickingID    int64  `json:"picking_id"`
		ProviderCode string `json:"provider_code"` // e.g., "opal", "dhl"
	}

	if err := json.NewDecoder(req.Body).Decode(&reqBody); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	err := r.deliveryService.CreateShipment(req.Context(), reqBody.PickingID, reqBody.ProviderCode)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Get the created delivery record
	delivery, _ := r.deliveryService.GetDeliveryStatus(reqBody.PickingID)
	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"message":  "Shipment created and queued for processing",
		"shipment": delivery,
	})
}

func (r *Router) listShipments(w http.ResponseWriter, req *http.Request) {
	state := req.URL.Query().Get("state")
	limit := 50 // Show last 50 shipments (newest first)

	shipments, err := r.deliveryService.ListShipments(state, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list shipments")
		return
	}

	respondJSON(w, http.StatusOK, shipments)
}

func (r *Router) getShipment(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	idStr := vars["id"]
	var id int64
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid shipment ID")
		return
	}

	shipment, err := r.deliveryService.GetShipment(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Shipment not found")
		return
	}

	respondJSON(w, http.StatusOK, shipment)
}

func (r *Router) cancelShipment(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	idStr := vars["id"]
	var pickingID int64
	if _, err := fmt.Sscanf(idStr, "%d", &pickingID); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid shipment ID")
		return
	}

	if err := r.deliveryService.CancelShipment(req.Context(), pickingID); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

func (r *Router) listCarriers(w http.ResponseWriter, req *http.Request) {
	carriers, err := r.deliveryService.ListCarriers()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list carriers")
		return
	}

	respondJSON(w, http.StatusOK, carriers)
}

func (r *Router) createCarrier(w http.ResponseWriter, req *http.Request) {
	var reqBody struct {
		Name         string `json:"name"`
		ProviderCode string `json:"provider_code"`
		ConfigJSON   string `json:"config_json"`
	}

	if err := json.NewDecoder(req.Body).Decode(&reqBody); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	carrier, err := r.deliveryService.CreateCarrier(reqBody.Name, reqBody.ProviderCode, reqBody.ConfigJSON)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, carrier)
}

func (r *Router) getCarrier(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	idStr := vars["id"]
	var id int64
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid carrier ID")
		return
	}

	carrier, err := r.deliveryService.GetCarrier(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Carrier not found")
		return
	}

	respondJSON(w, http.StatusOK, carrier)
}

func (r *Router) triggerOpalImport(w http.ResponseWriter, req *http.Request) {
	if r.deliveryService == nil {
		respondError(w, http.StatusServiceUnavailable, "Delivery service not configured")
		return
	}

	// Run import in background to not block the HTTP request
	go func() {
		// Use a fresh context as the request context will be cancelled
		if err := r.deliveryService.ImportOpalOrders(context.Background()); err != nil {
			fmt.Printf("Manual OPAL import failed: %v\n", err)
		}
	}()

	respondJSON(w, http.StatusOK, map[string]string{
		"status":  "started",
		"message": "OPAL synchronization started in background. Refresh in a few seconds.",
	})
}

func (r *Router) triggerDhlImport(w http.ResponseWriter, req *http.Request) {
	if r.deliveryService == nil {
		respondError(w, http.StatusServiceUnavailable, "Delivery service not configured")
		return
	}

	// Run import in background to not block the HTTP request
	go func() {
		// Use a fresh context as the request context will be cancelled
		if err := r.deliveryService.ImportDhlOrders(context.Background()); err != nil {
			fmt.Printf("Manual DHL import failed: %v\n", err)
		}
	}()

	respondJSON(w, http.StatusOK, map[string]string{
		"status":  "started",
		"message": "DHL synchronization started in background. Refresh in a few seconds.",
	})
}

func (r *Router) toggleCarrier(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	idStr := vars["id"]
	var id int64
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid carrier ID")
		return
	}

	if err := r.deliveryService.ToggleCarrier(id); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "toggled"})
}

// SetSyncEngine sets the sync engine and registers mesh sync routes
func (r *Router) SetSyncEngine(engine *sync.SyncEngine) {
	r.syncEngine = engine

	urlPrefix := os.Getenv("HTTP_PATH_PREFIX")
	if urlPrefix != "" {
		if !strings.HasPrefix(urlPrefix, "/") {
			urlPrefix = "/" + urlPrefix
		}
		urlPrefix = strings.TrimRight(strings.ToLower(urlPrefix), "/")
	}

	r.registerSyncRoutes(urlPrefix, engine)
}

// registerSyncRoutes registers mesh sync API routes
func (r *Router) registerSyncRoutes(prefix string, engine *sync.SyncEngine) {
	if engine == nil {
		return
	}

	syncHandler := NewSyncHandler(r.db, engine)

	// Use same pattern as Odoo routes
	meshPaths := []string{"/api/mesh"}
	syncPaths := []string{"/api/sync"}
	if prefix != "" {
		meshPaths = append(meshPaths, prefix+"/api/mesh")
		syncPaths = append(syncPaths, prefix+"/api/sync")
	}

	// Register mesh routes (no auth - uses mesh JWT)
	for _, p := range meshPaths {
		mesh := r.PathPrefix(p).Subrouter()
		mesh.HandleFunc("/pull", syncHandler.MeshPull).Methods("POST")
		mesh.HandleFunc("/push", syncHandler.MeshPush).Methods("POST")
		mesh.HandleFunc("/trigger", syncHandler.TriggerMeshSync).Methods("POST")
	}

	// Register sync routes (protected)
	for _, p := range syncPaths {
		syncApi := r.PathPrefix(p).Subrouter()
		syncApi.Use(middleware.AuthMiddleware)
		syncApi.HandleFunc("/status", syncHandler.GetSyncStatus).Methods("GET")
		syncApi.HandleFunc("/start", syncHandler.StartSync).Methods("POST")
		syncApi.HandleFunc("/full", syncHandler.TriggerFullSync).Methods("POST")
	}
}

// registerAdminRoutes registers device management routes with optional prefix
func (r *Router) registerAdminRoutes(prefix string) {
	paths := []string{"/api/admin"}
	if prefix != "" {
		paths = append(paths, prefix+"/api/admin")
	}

	for _, p := range paths {
		admin := r.PathPrefix(p).Subrouter()
		admin.Use(middleware.AuthMiddleware) // Protect these routes!

		admin.HandleFunc("/devices", r.listDevices).Methods("GET")
		admin.HandleFunc("/devices/{id}/status", r.updateDeviceStatus).Methods("PUT")
		admin.HandleFunc("/devices/{id}", r.deleteDevice).Methods("DELETE")
	}
}

// SetAIClient sets the AI client for AI-powered features
func (r *Router) SetAIClient(client *ai.GeminiClient) {
	r.aiClient = client
}
