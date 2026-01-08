package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/dmytrosurovtsev/eckwmsgo/internal/models"
	"github.com/gorilla/mux"
)

// listWarehouses returns all warehouses
func (r *Router) listWarehouses(w http.ResponseWriter, req *http.Request) {
	var warehouses []models.Warehouse
	if err := r.db.Preload("Racks").Find(&warehouses).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch warehouses")
		return
	}

	respondJSON(w, http.StatusOK, warehouses)
}

// getWarehouse returns a single warehouse by ID
func (r *Router) getWarehouse(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid warehouse ID")
		return
	}

	var warehouse models.Warehouse
	if err := r.db.Preload("Racks.Places").First(&warehouse, id).Error; err != nil {
		respondError(w, http.StatusNotFound, "Warehouse not found")
		return
	}

	respondJSON(w, http.StatusOK, warehouse)
}

// createWarehouse creates a new warehouse
func (r *Router) createWarehouse(w http.ResponseWriter, req *http.Request) {
	var warehouse models.Warehouse
	if err := json.NewDecoder(req.Body).Decode(&warehouse); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if err := r.db.Create(&warehouse).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create warehouse")
		return
	}

	respondJSON(w, http.StatusCreated, warehouse)
}

// listItems returns all items
func (r *Router) listItems(w http.ResponseWriter, req *http.Request) {
	var items []models.Item
	if err := r.db.Preload("Place").Preload("Box").Find(&items).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch items")
		return
	}

	respondJSON(w, http.StatusOK, items)
}

// getItem returns a single item by ID
func (r *Router) getItem(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid item ID")
		return
	}

	var item models.Item
	if err := r.db.Preload("Place").Preload("Box").First(&item, id).Error; err != nil {
		respondError(w, http.StatusNotFound, "Item not found")
		return
	}

	respondJSON(w, http.StatusOK, item)
}

// createItem creates a new item
func (r *Router) createItem(w http.ResponseWriter, req *http.Request) {
	var item models.Item
	if err := json.NewDecoder(req.Body).Decode(&item); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if err := r.db.Create(&item).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create item")
		return
	}

	respondJSON(w, http.StatusCreated, item)
}
