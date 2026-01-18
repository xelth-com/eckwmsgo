package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/xelth-com/eckwmsgo/internal/models"
	"github.com/gorilla/mux"
)

// listWarehouses returns all locations (formerly warehouses)
func (r *Router) listWarehouses(w http.ResponseWriter, req *http.Request) {
	var locations []models.StockLocation
	// Fetch top level locations (usually warehouses)
	if err := r.db.Where("location_id IS NULL").Find(&locations).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch locations")
		return
	}
	respondJSON(w, http.StatusOK, locations)
}

// getWarehouse returns a location hierarchy
func (r *Router) getWarehouse(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id, _ := strconv.ParseInt(vars["id"], 10, 64)

	var loc models.StockLocation
	if err := r.db.Preload("Children").First(&loc, id).Error; err != nil {
		respondError(w, http.StatusNotFound, "Location not found")
		return
	}
	respondJSON(w, http.StatusOK, loc)
}

// createWarehouse creates a new StockLocation
func (r *Router) createWarehouse(w http.ResponseWriter, req *http.Request) {
	var loc models.StockLocation
	if err := json.NewDecoder(req.Body).Decode(&loc); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid payload")
		return
	}
	// TODO: Generate ID if not present (since we use int64 PKs now matching Odoo)
	// For local-only creation, we might need negative IDs or a separate sequence.
	if err := r.db.Create(&loc).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create location")
		return
	}
	respondJSON(w, http.StatusCreated, loc)
}

// listItems returns all products
func (r *Router) listItems(w http.ResponseWriter, req *http.Request) {
	var products []models.ProductProduct
	if err := r.db.Find(&products).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch products")
		return
	}
	respondJSON(w, http.StatusOK, products)
}

// getItem returns a single product
func (r *Router) getItem(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id, _ := strconv.ParseInt(vars["id"], 10, 64)

	var product models.ProductProduct
	if err := r.db.First(&product, id).Error; err != nil {
		respondError(w, http.StatusNotFound, "Product not found")
		return
	}
	respondJSON(w, http.StatusOK, product)
}

// Stubs for removed functionality
func (r *Router) createItem(w http.ResponseWriter, req *http.Request)  { respondJSON(w, 200, nil) }
func (r *Router) updateItem(w http.ResponseWriter, req *http.Request)  { respondJSON(w, 200, nil) }
func (r *Router) createRack(w http.ResponseWriter, req *http.Request)  { respondJSON(w, 200, nil) }
func (r *Router) updateRack(w http.ResponseWriter, req *http.Request)  { respondJSON(w, 200, nil) }
func (r *Router) deleteRack(w http.ResponseWriter, req *http.Request)  { respondJSON(w, 200, nil) }
