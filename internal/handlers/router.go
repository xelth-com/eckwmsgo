package handlers

import (
	"encoding/json"
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

	// Health check endpoint
	r.HandleFunc("/health", r.healthCheck).Methods("GET")

	// API routes
	api := r.PathPrefix("/api").Subrouter()
	api.Use(middleware.AuthMiddleware)
	api.HandleFunc("/status", r.getStatus).Methods("GET")

	// Auth routes (Public)
	auth := r.PathPrefix("/auth").Subrouter()
	auth.HandleFunc("/login", r.login).Methods("POST")
	auth.HandleFunc("/register", r.register).Methods("POST")
	auth.HandleFunc("/logout", r.logout).Methods("POST")

	// Orders routes (protected) - Unified order table
	orders := r.PathPrefix("/api/orders").Subrouter()
	orders.Use(middleware.AuthMiddleware)
	orders.HandleFunc("", r.listOrders).Methods("GET")
	orders.HandleFunc("", r.createOrder).Methods("POST")
	orders.HandleFunc("/{id}", r.getOrder).Methods("GET")
	orders.HandleFunc("/{id}", r.updateOrder).Methods("PUT")
	orders.HandleFunc("/{id}", r.deleteOrder).Methods("DELETE")

	// Warehouse routes (protected)
	warehouse := r.PathPrefix("/api/warehouse").Subrouter()
	warehouse.Use(middleware.AuthMiddleware)
	warehouse.HandleFunc("", r.listWarehouses).Methods("GET")
	warehouse.HandleFunc("", r.createWarehouse).Methods("POST")
	warehouse.HandleFunc("/{id}", r.getWarehouse).Methods("GET")

	// Item routes (protected)
	items := r.PathPrefix("/api/items").Subrouter()
	items.Use(middleware.AuthMiddleware)
	items.HandleFunc("", r.listItems).Methods("GET")
	items.HandleFunc("", r.createItem).Methods("POST")
	items.HandleFunc("/{id}", r.getItem).Methods("GET")
	items.HandleFunc("/{id}", r.updateItem).Methods("PUT")

	// Rack routes (protected) - Nested under warehouse API usually, but flat here for simplicity
	racks := r.PathPrefix("/api/warehouse/racks").Subrouter()
	racks.Use(middleware.AuthMiddleware)
	racks.HandleFunc("", r.createRack).Methods("POST")
	racks.HandleFunc("/{id}", r.updateRack).Methods("PUT")
	racks.HandleFunc("/{id}", r.deleteRack).Methods("DELETE")

	// Setup & Device routes (protected)
	setup := r.PathPrefix("/api/internal").Subrouter()
	setup.Use(middleware.AuthMiddleware)
	setup.HandleFunc("/pairing-qr", r.generatePairingQR).Methods("GET")

	// Print routes (protected)
	print := r.PathPrefix("/api/print").Subrouter()
	print.Use(middleware.AuthMiddleware)
	print.HandleFunc("/labels", r.generateLabels).Methods("POST")

	// Public device registration (device calls this initially)
	r.HandleFunc("/api/internal/register-device", r.registerDevice).Methods("POST")

	// WebSocket endpoint
	r.HandleFunc("/ws", func(w http.ResponseWriter, req *http.Request) {
		websocket.ServeWs(hub, w, req)
	})

	// --- Static Files (Svelte Frontend) ---
	// Get filesystem (embedded or disk)
	assets, err := web.GetFileSystem()
	if err != nil {
		// Fallback if something is wrong with embed
		publicDir := os.Getenv("FRONTEND_DIR")
		if publicDir == "" {
			publicDir = "web/build" // Default for SvelteKit
		}
		assets = os.DirFS(publicDir)
	}

	// SPA Handler: Serve index.html for unknown routes (so Svelte router works)
	spaHandler := http.FileServer(http.FS(assets))

	r.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// If path looks like an API call or file extension, serve normally
		path := req.URL.Path
		if strings.HasPrefix(path, "/api") || strings.HasPrefix(path, "/auth") ||
			strings.HasPrefix(path, "/ws") || strings.HasPrefix(path, "/health") ||
			strings.HasPrefix(path, "/orders") || strings.Contains(path, ".") {
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
