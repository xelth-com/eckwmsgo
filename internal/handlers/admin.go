package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/xelth-com/eckwmsgo/internal/models"
	"github.com/gorilla/mux"
)

// ListDevices returns all registered devices ordered by status (pending first)
func (r *Router) listDevices(w http.ResponseWriter, req *http.Request) {
	var devices []models.RegisteredDevice
	// Pending first, then by date
	if err := r.db.Order("CASE WHEN status = 'pending' THEN 1 WHEN status = 'active' THEN 2 ELSE 3 END, created_at DESC").Find(&devices).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch devices")
		return
	}
	respondJSON(w, http.StatusOK, devices)
}

// UpdateDeviceStatus changes the status of a device (e.g. pending -> active)
func (r *Router) updateDeviceStatus(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars["id"]

	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	// Validate status enum
	status := models.DeviceStatus(body.Status)
	if status != models.DeviceStatusActive && status != models.DeviceStatusBlocked && status != models.DeviceStatusPending {
		respondError(w, http.StatusBadRequest, "Invalid status")
		return
	}

	var device models.RegisteredDevice
	if err := r.db.First(&device, "device_id = ?", id).Error; err != nil {
		respondError(w, http.StatusNotFound, "Device not found")
		return
	}

	device.Status = status
	if err := r.db.Save(&device).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update status")
		return
	}

	respondJSON(w, http.StatusOK, device)
}

// DeleteDevice soft-deletes a device (sets DeletedAt timestamp for mesh sync)
func (r *Router) deleteDevice(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars["id"]

	var device models.RegisteredDevice
	if err := r.db.Where("\"deviceId\" = ?", id).First(&device).Error; err != nil {
		respondError(w, http.StatusNotFound, "Device not found")
		return
	}

	// Soft delete (sets DeletedAt timestamp) - will sync to other nodes
	if err := r.db.Delete(&device).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete device")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Device deleted successfully",
		"id":      id,
	})
}
