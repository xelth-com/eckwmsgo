package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/xelth-com/eckwmsgo/internal/ai"
	"github.com/xelth-com/eckwmsgo/internal/models"
	"github.com/xelth-com/eckwmsgo/internal/utils"
	"gorm.io/gorm"
)

// ScanRequest represents the payload from a scanner
type ScanRequest struct {
	Barcode  string `json:"barcode"`
	MsgID    string `json:"msgId"`
	DeviceID string `json:"deviceId"`
}

// ScanResponse standardizes the scan result
type ScanResponse struct {
	Type          string      `json:"type"`                     // item, box, place, label
	Message       string      `json:"message"`                  // Human readable status
	Action        string      `json:"action"`                   // created, found, error
	Checksum      string      `json:"checksum"`                 // For Android
	AiInteraction interface{} `json:"ai_interaction,omitempty"` // For Android
	Data          interface{} `json:"data,omitempty"`           // The resulting object
	MsgID         string      `json:"msgId,omitempty"`          // Echo back msgId
	Duplicate     bool        `json:"duplicate,omitempty"`      // Flag for duplicates
}

// handleScan is the universal entry point for all barcode scans
func (r *Router) handleScan(w http.ResponseWriter, req *http.Request) {
	var body ScanRequest
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Deduplication check - ignore duplicate messages
	if utils.IsDuplicate(body.MsgID) {
		respondJSON(w, http.StatusOK, ScanResponse{
			Type:      "duplicate",
			Message:   "Message already processed",
			Action:    "ignore",
			MsgID:     body.MsgID,
			Duplicate: true,
		})
		return
	}

	// SECURITY CHECK: Validate Device Status
	// Ensures that even if JWT is valid, blocked devices cannot scan
	if body.DeviceID != "" {
		var device models.RegisteredDevice
		if err := r.db.First(&device, "device_id = ?", body.DeviceID).Error; err != nil {
			respondError(w, http.StatusForbidden, "Device not registered")
			return
		}
		if device.Status != models.DeviceStatusActive {
			respondError(w, http.StatusForbidden, fmt.Sprintf("Device is %s", device.Status))
			return
		}
	}

	barcode := strings.TrimSpace(body.Barcode)
	if len(barcode) < 1 {
		respondError(w, http.StatusBadRequest, "Empty barcode")
		return
	}

	// 0. Decrypt if it's an ECK URL (Encrypted QR)
	// Format: ECK1.COM/ENCRYPTEDSTRINGXX or https://ECK1.COM/...
	if strings.Contains(barcode, "ECK") && strings.Contains(barcode, ".COM/") {
		// Clean up URL prefix if present
		cleanCode := barcode
		if strings.HasPrefix(cleanCode, "http://") {
			cleanCode = strings.TrimPrefix(cleanCode, "http://")
		}
		if strings.HasPrefix(cleanCode, "https://") {
			cleanCode = strings.TrimPrefix(cleanCode, "https://")
		}

		// Attempt decryption
		decrypted, err := utils.EckURLDecrypt(cleanCode)
		if err == nil {
			// Successfully decrypted, switch to the raw code
			barcode = decrypted
		} else {
			// Decryption failed - log but continue (might be a false positive)
			fmt.Printf("Decryption failed for %s: %v\n", cleanCode, err)
		}
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
		toolSvc := ai.NewToolService(r.db)
		if alias, found := toolSvc.SearchInventory(barcode); found {
			log.Printf("🧠 Memory Hit: %s is alias for %s", barcode, alias.InternalID)
			if strings.HasPrefix(alias.InternalID, "i") {
				resp, err = r.processItemScan(alias.InternalID)
			} else if strings.HasPrefix(alias.InternalID, "b") {
				resp, err = r.processBoxScan(alias.InternalID)
			} else {
				resp = ScanResponse{
					Type:    "alias",
					Action:  "found",
					Message: fmt.Sprintf("Alias for %s", alias.InternalID),
					Data:    alias,
				}
			}
			if err == nil {
				resp.MsgID = body.MsgID
				respondJSON(w, http.StatusOK, resp)
				return
			}
		}

		if r.aiClient != nil {
			// Build dynamic prompt with config
			systemPrompt := ai.BuildConsultantPrompt(
				r.config.AI.CompanyName,
				r.config.AI.ManufacturerURL,
				r.config.AI.SupportEmail,
			)
			prompt := fmt.Sprintf("Worker scanned unknown code: '%s'. Analyze it.", barcode)
			fullPrompt := systemPrompt + "\n\nUSER INPUT: " + prompt

			aiResponseStr, err := r.aiClient.GenerateContent(req.Context(), fullPrompt)
			if err == nil {
				cleanJson := utils.SanitizeJSON(aiResponseStr)
				var interaction map[string]interface{}
				if json.Unmarshal([]byte(cleanJson), &interaction) == nil {
					resp = ScanResponse{
						Type:          "ai_analysis",
						Message:       "AI Analysis",
						Action:        "interaction",
						AiInteraction: interaction,
					}
					resp.MsgID = body.MsgID
					respondJSON(w, http.StatusOK, resp)
					return
				} else {
					fmt.Printf("AI JSON Parse Error. Raw: %s\nClean: %s\n", aiResponseStr, cleanJson)
				}
			} else {
				fmt.Printf("AI Gen Error: %v\n", err)
			}
		}

		resp, err = r.processLegacyScan(barcode)
	}

	if err != nil {
		// Log error but return 400 to client
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Echo back MsgID for client correlation
	resp.MsgID = body.MsgID
	respondJSON(w, http.StatusOK, resp)
}

// AIResponseRequest represents the user feedback
type AIResponseRequest struct {
	InteractionID string `json:"interactionId"`
	Response      string `json:"response"`
	Barcode       string `json:"barcode"`
	DeviceID      string `json:"deviceId"`
}

// handleAiRespond processes user feedback from the Android app
func (r *Router) handleAiRespond(w http.ResponseWriter, req *http.Request) {
	var body AIResponseRequest
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	log.Printf("🤖 AI Response received: %s for %s (Interaction: %s)", body.Response, body.Barcode, body.InteractionID)

	actionTaken := "logged_only"

	if strings.ToLower(body.Response) == "yes" {
		toolSvc := ai.NewToolService(r.db)
		internalID := "b_receiving_dock"

		err := toolSvc.LinkCode(internalID, body.Barcode, "manual_confirmation", "android_feedback")
		if err != nil {
			log.Printf("❌ Failed to link code: %v", err)
			actionTaken = "error_linking"
		} else {
			actionTaken = "linked_to_db"
			log.Printf("✅ Successfully linked %s -> %s", body.Barcode, internalID)
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "AI response processed",
		"action":  actionTaken,
	})
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
			Barcode:   models.OdooString(data.Type),
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
			Barcode:     models.OdooString(data.RefID),
			DefaultCode: models.OdooString("STUB-" + data.RefID),
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
			Ref:        models.OdooString(code), // Store full smart code for reference
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
