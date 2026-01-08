package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/dmytrosurovtsev/eckwmsgo/internal/database"
	"github.com/gorilla/mux"
)

// Router wraps the mux router and database
type Router struct {
	*mux.Router
	db *database.DB
}

// NewRouter creates a new HTTP router with all routes
func NewRouter(db *database.DB) *Router {
	r := &Router{
		Router: mux.NewRouter(),
		db:     db,
	}

	// Health check endpoint
	r.HandleFunc("/health", r.healthCheck).Methods("GET")

	// API routes
	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/status", r.getStatus).Methods("GET")

	// Auth routes
	auth := r.PathPrefix("/auth").Subrouter()
	auth.HandleFunc("/login", r.login).Methods("POST")
	auth.HandleFunc("/register", r.register).Methods("POST")
	auth.HandleFunc("/logout", r.logout).Methods("POST")

	// RMA routes (protected)
	rma := r.PathPrefix("/rma").Subrouter()
	rma.HandleFunc("", r.listRMAs).Methods("GET")
	rma.HandleFunc("", r.createRMA).Methods("POST")
	rma.HandleFunc("/{id}", r.getRMA).Methods("GET")
	rma.HandleFunc("/{id}", r.updateRMA).Methods("PUT")
	rma.HandleFunc("/{id}", r.deleteRMA).Methods("DELETE")

	// Warehouse routes (protected)
	warehouse := r.PathPrefix("/api/warehouse").Subrouter()
	warehouse.HandleFunc("", r.listWarehouses).Methods("GET")
	warehouse.HandleFunc("", r.createWarehouse).Methods("POST")
	warehouse.HandleFunc("/{id}", r.getWarehouse).Methods("GET")

	// Item routes (protected)
	items := r.PathPrefix("/api/items").Subrouter()
	items.HandleFunc("", r.listItems).Methods("GET")
	items.HandleFunc("", r.createItem).Methods("POST")
	items.HandleFunc("/{id}", r.getItem).Methods("GET")

	// Static files - serve from ../eckwms/public
	publicDir := os.Getenv("FRONTEND_DIR")
	if publicDir == "" {
		// Default: assume eckwms is in parent directory
		publicDir = filepath.Join("..", "eckwms", "public")
	}
	r.PathPrefix("/").Handler(http.FileServer(http.Dir(publicDir)))

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
