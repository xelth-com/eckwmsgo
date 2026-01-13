package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/dmytrosurovtsev/eckwmsgo/internal/database"
	"github.com/dmytrosurovtsev/eckwmsgo/internal/middleware"
	"github.com/dmytrosurovtsev/eckwmsgo/internal/websocket"
	"github.com/dmytrosurovtsev/eckwmsgo/web"
	"github.com/gorilla/mux"
)

// Router wraps the mux router and database
type Router struct {
	*mux.Router
	db  *database.DB
	hub *websocket.Hub
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

	// Health check endpoint (support both prefixed and root)
	handle("/health", r.healthCheck, "GET")

	// Auth routes (Public)
	handle("/auth/login", r.login, "POST")
	handle("/auth/register", r.register, "POST")
	handle("/auth/logout", r.logout, "POST")

	// API status endpoint (protected)
	paths := []string{"/api"}
	if urlPrefix != "" {
		paths = append(paths, urlPrefix+"/api")
	}
	for _, p := range paths {
		api := r.PathPrefix(p).Subrouter()
		api.Use(middleware.AuthMiddleware)
		api.HandleFunc("/status", r.getStatus).Methods("GET")
	}

	// Register route groups with prefix support
	r.registerOrdersRoutes(urlPrefix)
	r.registerWarehouseRoutes(urlPrefix)
	r.registerItemsRoutes(urlPrefix)
	r.registerSetupRoutes(urlPrefix)
	r.registerPrintRoutes(urlPrefix)
	r.registerAIRoutes(urlPrefix, db)

	// WebSocket endpoint
	handle("/ws", func(w http.ResponseWriter, req *http.Request) {
		websocket.ServeWs(hub, w, req)
	})

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

	// SPA Handler Logic - use NotFoundHandler so it's called only when no route matches
	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		originalPath := req.URL.Path
		path := originalPath

		// Strip prefix for static file lookup
		if urlPrefix != "" && strings.HasPrefix(originalPath, urlPrefix) {
			path = strings.TrimPrefix(originalPath, urlPrefix)
			if path == "" {
				path = "/"
			}
		}

		// Debug logging for static files
		if strings.HasPrefix(path, "/internal") {
			// Log what we're trying to serve
			fmt.Printf("DEBUG: Serving static file: original=%s, path=%s, prefix=%s\n", originalPath, path, urlPrefix)
		}

		// Serve static files or SPA
		// Files with extension or /internal/ path get served as-is
		if strings.HasPrefix(path, "/internal") || strings.Contains(path, ".") {
			// For static files, we need to modify req.URL.Path to the stripped version
			// before calling the file server
			if urlPrefix != "" && strings.HasPrefix(originalPath, urlPrefix) {
				// Create a new request with modified path
				req.URL.Path = path
			}
			spaHandler.ServeHTTP(w, req)
			return
		}

		// Otherwise serve index.html (SPA)
		req.URL.Path = "/"
		spaHandler.ServeHTTP(w, req)
	})

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
		"status": "running",
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
		wh.HandleFunc("/{id}", r.getWarehouse).Methods("GET")

		// Racks (nested under warehouse)
		wh.HandleFunc("/racks", r.createRack).Methods("POST")
		wh.HandleFunc("/racks/{id}", r.updateRack).Methods("PUT")
		wh.HandleFunc("/racks/{id}", r.deleteRack).Methods("DELETE")
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

		// Public registration route needs to be handled on main router
		regPath := p + "/register-device"
		r.HandleFunc(regPath, r.registerDevice).Methods("POST")
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
