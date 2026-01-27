package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/xelth-com/eckwmsgo/internal/models"
)

// createDocument handles the submission of new documents from Android
func (r *Router) createDocument(w http.ResponseWriter, req *http.Request) {
	var doc models.Document
	if err := json.NewDecoder(req.Body).Decode(&doc); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	// Basic validation
	if doc.Type == "" {
		respondError(w, http.StatusBadRequest, "Document type is required")
		return
	}

	// Force initial status
	doc.Status = "pending"

	// Save to DB
	if err := r.db.Create(&doc).Error; err != nil {
		log.Printf("❌ Failed to save document: %v", err)
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}

	log.Printf("📄 Document received: %s (Type: %s) from Device: %s", doc.ID, doc.Type, doc.DeviceID)

	// In the future: Trigger AI processing here based on doc.Type

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"success":    true,
		"documentId": doc.ID,
		"message":    "Document received and queued for processing",
	})
}

// listDocuments returns recent documents (for dashboard/debugging)
func (r *Router) listDocuments(w http.ResponseWriter, req *http.Request) {
	var docs []models.Document
	// Return last 50 documents
	if err := r.db.Order("created_at DESC").Limit(50).Find(&docs).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch documents")
		return
	}
	respondJSON(w, http.StatusOK, docs)
}
