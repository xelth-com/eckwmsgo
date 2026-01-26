package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/xelth-com/eckwmsgo/internal/models"
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
func (r *Router) createItem(w http.ResponseWriter, req *http.Request) { respondJSON(w, 200, nil) }
func (r *Router) updateItem(w http.ResponseWriter, req *http.Request) { respondJSON(w, 200, nil) }

// --- Warehouse Rack Handlers (Blueprint Editor) ---

// listRacks returns all racks, optionally filtered by warehouse_id
func (r *Router) listRacks(w http.ResponseWriter, req *http.Request) {
	warehouseID := req.URL.Query().Get("warehouse_id")

	var racks []models.WarehouseRack
	query := r.db.Order("id ASC")

	if warehouseID != "" {
		whID, err := strconv.ParseInt(warehouseID, 10, 64)
		if err == nil {
			query = query.Where("warehouse_id = ?", whID)
		}
	}

	if err := query.Find(&racks).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch racks")
		return
	}
	respondJSON(w, http.StatusOK, racks)
}

// createRack creates or updates a rack (upsert logic)
func (r *Router) createRack(w http.ResponseWriter, req *http.Request) {
	var rack models.WarehouseRack
	if err := json.NewDecoder(req.Body).Decode(&rack); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid payload")
		return
	}

	// Upsert: if ID exists, update; otherwise create
	if rack.ID > 0 {
		var existing models.WarehouseRack
		if err := r.db.First(&existing, rack.ID).Error; err == nil {
			// Update existing rack
			if err := r.db.Model(&existing).Updates(&rack).Error; err != nil {
				respondError(w, http.StatusInternalServerError, "Failed to update rack")
				return
			}
			respondJSON(w, http.StatusOK, existing)
			return
		}
	}

	// Auto-calculate start_index if not provided or -1
	if rack.StartIndex <= 0 {
		var maxIndex int
		r.db.Model(&models.WarehouseRack{}).
			Where("warehouse_id = ?", rack.WarehouseID).
			Select("COALESCE(MAX(start_index + columns * rows), 0)").
			Scan(&maxIndex)
		rack.StartIndex = maxIndex + 1
	}

	// Create new rack
	if err := r.db.Create(&rack).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create rack")
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"id":                rack.ID,
		"calculatedStartId": rack.StartIndex,
	})
}

// updateRack updates an existing rack
func (r *Router) updateRack(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id, _ := strconv.ParseInt(vars["id"], 10, 64)

	var existing models.WarehouseRack
	if err := r.db.First(&existing, id).Error; err != nil {
		respondError(w, http.StatusNotFound, "Rack not found")
		return
	}

	var updates models.WarehouseRack
	if err := json.NewDecoder(req.Body).Decode(&updates); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid payload")
		return
	}

	if err := r.db.Model(&existing).Updates(&updates).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update rack")
		return
	}

	respondJSON(w, http.StatusOK, existing)
}

// deleteRack deletes a rack
func (r *Router) deleteRack(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id, _ := strconv.ParseInt(vars["id"], 10, 64)

	if err := r.db.Delete(&models.WarehouseRack{}, id).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete rack")
		return
	}

	respondJSON(w, http.StatusOK, map[string]bool{"success": true})
}
