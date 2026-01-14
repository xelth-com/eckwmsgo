package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dmytrosurovtsev/eckwmsgo/internal/models"
	"github.com/dmytrosurovtsev/eckwmsgo/internal/utils"
	"gorm.io/gorm"
)

// ScanRequest represents the payload from a scanner
type ScanRequest struct {
	Barcode string `json:"barcode"`
}

// ScanResponse standardizes the scan result
type ScanResponse struct {
	Type    string      `json:"type"`           // item, box, place, label
	Message string      `json:"message"`        // Human readable status
	Action  string      `json:"action"`         // created, found, error
	Data    interface{} `json:"data,omitempty"` // The resulting object
}

// handleScan is the universal entry point for all barcode scans
func (r *Router) handleScan(w http.ResponseWriter, req *http.Request) {
	var body ScanRequest
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	barcode := strings.TrimSpace(body.Barcode)
	if len(barcode) < 1 {
		respondError(w, http.StatusBadRequest, "Empty barcode")
		return
	}

	// 1. Identify Type by Prefix
	prefix := string(barcode[0])
	var resp ScanResponse
	var err error

	switch prefix {
	case "i":
		resp, err = r.processItemScan(barcode)
	case "b":
		resp, err = r.processBoxScan(barcode)
	case "p":
		resp, err = r.processPlaceScan(barcode)
	case "l":
		resp, err = r.processLabelScan(barcode)
	default:
		// Fallback: Try to find a product by EAN directly (legacy/retail barcode)
		resp, err = r.processLegacyScan(barcode)
	}

	if err != nil {
		// Log error but return 400 to client
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, resp)
}

// processBoxScan handles 'b' codes (dimensions + weight + type)
func (r *Router) processBoxScan(code string) (ScanResponse, error) {
	// 1. Decode
	data, err := utils.DecodeSmartBox(code)
	if err != nil {
		return ScanResponse{}, fmt.Errorf("invalid box code: %v", err)
	}

	// 2. Check existence
	var pkg models.StockQuantPackage
	err = r.db.Where("name = ?", code).Preload("PackageType").First(&pkg).Error

	if err == nil {
		return ScanResponse{Type: "box", Action: "found", Message: "Box found", Data: pkg}, nil
	}
	if err != gorm.ErrRecordNotFound {
		return ScanResponse{}, err
	}

	// 3. Lazy Creation (Offline First)
	// We use a large int64 for ID to avoid immediate conflict with Odoo's low integers
	localID := time.Now().UnixNano()

	// A. Find or Create Package Type based on dimensions
	var pkgType models.StockPackageType
	// Naming convention: "SmartBox [LxWxH]"
	typeName := fmt.Sprintf("SmartBox %dx%dx%d", data.Length, data.Width, data.Height)

	// Try to find matching type first
	err = r.db.Where("length = ? AND width = ? AND height = ?", data.Length, data.Width, data.Height).First(&pkgType).Error
	if err == gorm.ErrRecordNotFound {
		pkgType = models.StockPackageType{
			ID:        localID, // Temp local ID
			Name:      typeName,
			Barcode:   data.Type,
			Length:    data.Length,
			Width:     data.Width,
			Height:    data.Height,
			MaxWeight: 50.0, // Default safe max
		}
		if err := r.db.Create(&pkgType).Error; err != nil {
			return ScanResponse{}, fmt.Errorf("failed to create package type: %v", err)
		}
		// Increment for next usage
		localID++
	}

	// B. Create Package
	newPkg := models.StockQuantPackage{
		ID:            localID,
		Name:          code,
		PackageTypeID: &pkgType.ID,
		PackDate:      time.Now(),
	}

	if err := r.db.Create(&newPkg).Error; err != nil {
		return ScanResponse{}, fmt.Errorf("failed to create package: %v", err)
	}

	// Return with loaded relation
	newPkg.PackageType = &pkgType
	return ScanResponse{Type: "box", Action: "created", Message: "Box registered locally", Data: newPkg}, nil
}

// processItemScan handles 'i' codes (Serial + EAN)
func (r *Router) processItemScan(code string) (ScanResponse, error) {
	// 1. Decode
	data, err := utils.DecodeSmartItem(code)
	if err != nil {
		return ScanResponse{}, fmt.Errorf("invalid item code: %v", err)
	}

	// 2. Resolve Product (by EAN/RefID)
	var product models.ProductProduct
	err = r.db.Where("barcode = ?", data.RefID).First(&product).Error

	if err == gorm.ErrRecordNotFound {
		// Lazy Create Stub Product
		localID := time.Now().UnixNano()
		product = models.ProductProduct{
			ID:          localID,
			Name:        "Unknown Product (" + data.RefID + ")",
			Barcode:     data.RefID,
			DefaultCode: "STUB-" + data.RefID,
			Active:      true,
			Type:        "product",
		}
		if err := r.db.Create(&product).Error; err != nil {
			return ScanResponse{}, fmt.Errorf("failed to create stub product: %v", err)
		}
	} else if err != nil {
		return ScanResponse{}, err
	}

	// 3. Resolve Serial (StockLot)
	var lot models.StockLot
	err = r.db.Where("name = ?", data.Serial).First(&lot).Error

	if err == gorm.ErrRecordNotFound {
		// Register Serial
		lot = models.StockLot{
			ID:         time.Now().UnixNano(),
			Name:       data.Serial,
			ProductID:  product.ID,
			Ref:        code, // Store full smart code for reference
			CreateDate: time.Now(),
		}
		if err := r.db.Create(&lot).Error; err != nil {
			return ScanResponse{}, fmt.Errorf("failed to register serial: %v", err)
		}
		return ScanResponse{Type: "item", Action: "created", Message: "Item registered", Data: lot}, nil
	}

	return ScanResponse{Type: "item", Action: "found", Message: "Item scanned", Data: lot}, nil
}

// processPlaceScan handles 'p' codes (Locations)
func (r *Router) processPlaceScan(code string) (ScanResponse, error) {
	var loc models.StockLocation
	// Odoo stores the barcode in the 'barcode' field
	if err := r.db.Where("barcode = ?", code).First(&loc).Error; err != nil {
		return ScanResponse{}, fmt.Errorf("location not found: %s", code)
	}
	return ScanResponse{Type: "place", Action: "found", Message: loc.CompleteName, Data: loc}, nil
}

// processLabelScan handles 'l' codes (Actions/Metas)
func (r *Router) processLabelScan(code string) (ScanResponse, error) {
	data, err := utils.DecodeSmartLabel(code)
	if err != nil {
		return ScanResponse{}, err
	}
	// Just return decoded data for frontend to act on
	return ScanResponse{Type: "label", Action: "decoded", Message: "Smart Label", Data: data}, nil
}

// processLegacyScan tries to find a product by raw barcode (EAN)
func (r *Router) processLegacyScan(barcode string) (ScanResponse, error) {
	var product models.ProductProduct
	if err := r.db.Where("barcode = ?", barcode).First(&product).Error; err != nil {
		return ScanResponse{}, fmt.Errorf("unknown barcode")
	}
	return ScanResponse{Type: "product", Action: "found", Message: product.Name, Data: product}, nil
}
