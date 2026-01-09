package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/dmytrosurovtsev/eckwmsgo/internal/services/printer"
)

// generateLabels handles the PDF generation request
func (r *Router) generateLabels(w http.ResponseWriter, req *http.Request) {
	var config printer.LabelConfig
	if err := json.NewDecoder(req.Body).Decode(&config); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Validate defaults
	if config.Cols == 0 {
		config.Cols = 3
	}
	if config.Rows == 0 {
		config.Rows = 7
	}
	if config.Count == 0 {
		config.Count = 21
	}
	if config.Type == "" {
		config.Type = "i"
	}

	pdfBytes, err := printer.GenerateLabelsPDF(config)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to generate PDF: %v", err))
		return
	}

	// Set headers for download
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"labels_%s_%d.pdf\"", config.Type, config.StartNumber))
	w.Header().Set("Content-Length", strconv.Itoa(len(pdfBytes)))

	w.Write(pdfBytes)
}
