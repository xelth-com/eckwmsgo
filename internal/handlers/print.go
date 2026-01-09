package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/dmytrosurovtsev/eckwmsgo/internal/services/printer"
)

// generateLabels handles the PDF generation request with full puzzle layout support
func (r *Router) generateLabels(w http.ResponseWriter, req *http.Request) {
	var config printer.LabelConfig
	if err := json.NewDecoder(req.Body).Decode(&config); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}

	// Apply robust defaults matching Node.js behavior
	if config.Cols == 0 {
		config.Cols = 2
	}
	if config.Rows == 0 {
		config.Rows = 8
	}
	if config.Count == 0 {
		config.Count = config.Cols * config.Rows
	}
	if config.Type == "" {
		config.Type = "i"
	}

	// Default margins (in mm) if not provided
	if config.MarginTop == 0 && config.MarginBottom == 0 && config.MarginLeft == 0 && config.MarginRight == 0 {
		config.MarginTop = 7
		config.MarginBottom = 7
		config.MarginLeft = 7
		config.MarginRight = 7
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
