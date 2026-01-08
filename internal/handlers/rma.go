package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/dmytrosurovtsev/eckwmsgo/internal/models"
	"github.com/gorilla/mux"
)

// listRMAs returns all RMA requests
func (r *Router) listRMAs(w http.ResponseWriter, req *http.Request) {
	var rmas []models.RmaRequest
	if err := r.db.Find(&rmas).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch RMA requests")
		return
	}

	respondJSON(w, http.StatusOK, rmas)
}

// getRMA returns a single RMA request by ID
func (r *Router) getRMA(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid RMA ID")
		return
	}

	var rma models.RmaRequest
	if err := r.db.First(&rma, id).Error; err != nil {
		respondError(w, http.StatusNotFound, "RMA request not found")
		return
	}

	respondJSON(w, http.StatusOK, rma)
}

// createRMA creates a new RMA request
func (r *Router) createRMA(w http.ResponseWriter, req *http.Request) {
	var rma models.RmaRequest
	if err := json.NewDecoder(req.Body).Decode(&rma); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if err := r.db.Create(&rma).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create RMA request")
		return
	}

	respondJSON(w, http.StatusCreated, rma)
}

// updateRMA updates an existing RMA request
func (r *Router) updateRMA(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid RMA ID")
		return
	}

	var rma models.RmaRequest
	if err := r.db.First(&rma, id).Error; err != nil {
		respondError(w, http.StatusNotFound, "RMA request not found")
		return
	}

	if err := json.NewDecoder(req.Body).Decode(&rma); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if err := r.db.Save(&rma).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update RMA request")
		return
	}

	respondJSON(w, http.StatusOK, rma)
}

// deleteRMA deletes an RMA request
func (r *Router) deleteRMA(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid RMA ID")
		return
	}

	if err := r.db.Delete(&models.RmaRequest{}, id).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete RMA request")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "RMA request deleted successfully",
	})
}
