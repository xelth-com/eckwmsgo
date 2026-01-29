package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/xelth-com/eckwmsgo/internal/models"
)

// ListDevices returns all registered devices ordered by status (pending first)
// Optional query param: ?include_deleted=true to include soft-deleted devices
func (r *Router) listDevices(w http.ResponseWriter, req *http.Request) {
	var devices []models.RegisteredDevice

	// Check if client wants to see deleted devices
	var err error
	if req.URL.Query().Get("include_deleted") == "true" {
		// Unscoped() returns *gorm.DB, use it directly
		err = r.db.Unscoped().Order("CASE WHEN status = 'pending' THEN 1 WHEN status = 'active' THEN 2 ELSE 3 END, created_at DESC").Find(&devices).Error
	} else {
		// Normal query excludes soft-deleted devices
		err = r.db.Order("CASE WHEN status = 'pending' THEN 1 WHEN status = 'active' THEN 2 ELSE 3 END, created_at DESC").Find(&devices).Error
	}

	if err != nil {
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
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	status := models.DeviceStatus(body.Status)
	if status != models.DeviceStatusActive && status != models.DeviceStatusBlocked && status != models.DeviceStatusPending {
		respondError(w, http.StatusBadRequest, "Invalid status")
		return
	}

	var device models.RegisteredDevice
	if err := r.db.Where("device_id = ?", id).First(&device).Error; err != nil {
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
// The deletion will be synchronized to all mesh nodes
func (r *Router) deleteDevice(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars["id"]

	var device models.RegisteredDevice
	// Use Unscoped to find even soft-deleted devices
	if err := r.db.Unscoped().Where("device_id = ?", id).First(&device).Error; err != nil {
		respondError(w, http.StatusNotFound, "Device not found")
		return
	}

	// Soft delete (sets DeletedAt timestamp) - will sync to other nodes
	// This triggers AfterUpdate hook which updates checksum with DeletedAt timestamp
	if err := r.db.Delete(&device).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete device")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Device deleted successfully (soft deleted for sync)",
		"id":      id,
	})
}

// RestoreDevice restores a soft-deleted device (clears DeletedAt timestamp)
// This allows re-adding a device that was previously deleted
func (r *Router) restoreDevice(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars["id"]

	var device models.RegisteredDevice
	// Find the deleted device using Unscoped
	if err := r.db.Unscoped().Where("device_id = ?", id).First(&device).Error; err != nil {
		respondError(w, http.StatusNotFound, "Device not found")
		return
	}

	if device.DeletedAt.Time.IsZero() {
		respondError(w, http.StatusBadRequest, "Device is not deleted")
		return
	}

	// Restore by clearing DeletedAt
	// This triggers AfterUpdate hook which updates checksum (now without DeletedAt)
	if err := r.db.Model(&device).Update("deleted_at", nil).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to restore device")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Device restored successfully (will sync to mesh)",
		"id":      id,
	})
}
