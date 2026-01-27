package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/skip2/go-qrcode"
	"github.com/xelth-com/eckwmsgo/internal/config"
	"github.com/xelth-com/eckwmsgo/internal/models"
	"github.com/xelth-com/eckwmsgo/internal/utils"
	"gorm.io/gorm"
)

// GeneratePairingQR generates a QR code for device pairing (ECK-P1-ALPHA protocol)
// Protocol: ECK$2$COMPACTUUID$PUBKEY_HEX$URL_LIST[$INVITE_TOKEN]
func (r *Router) generatePairingQR(w http.ResponseWriter, req *http.Request) {
	identity := utils.GetServerIdentity()
	if identity == nil {
		respondError(w, http.StatusInternalServerError, "Server identity not initialized")
		return
	}

	// 1. Compact UUID (remove dashes, uppercase)
	compactUUID := strings.ToUpper(strings.ReplaceAll(identity.InstanceID, "-", ""))

	// 2. Public Key (Hex, uppercase)
	pubKeyHex, err := identity.GetPublicKeyHex()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Invalid server key")
		return
	}
	pubKeyHex = strings.ToUpper(pubKeyHex)

	// 3. Construct Connection Candidates List (Protocol v2)
	var candidates []string
	port := os.Getenv("PORT")
	if port == "" {
		port = "3210"
	}

	// A. Add Local IPs (Fastest/Preferred)
	localIPs := utils.GetLocalIPs()

	// Load path prefix directly from environment
	prefix := os.Getenv("HTTP_PATH_PREFIX")
	if prefix != "" && !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	prefix = strings.TrimSuffix(prefix, "/")

	for _, ip := range localIPs {
		candidates = append(candidates, fmt.Sprintf("http://%s:%s%s", ip, port, prefix))
	}

	// B. Add Global URL (Fallback/Remote)
	globalURL := os.Getenv("GLOBAL_SERVER_URL")
	if globalURL != "" {
		if !strings.HasSuffix(globalURL, "/") {
			globalURL += "/"
		}
		candidates = append(candidates, globalURL)
	}

	// If list is empty (edge case), try to use Host header
	if len(candidates) == 0 {
		scheme := "http"
		if req.TLS != nil {
			scheme = "https"
		}
		candidates = append(candidates, fmt.Sprintf("%s://%s", scheme, req.Host))
	}

	// Join with commas and uppercase
	connectionString := strings.ToUpper(strings.Join(candidates, ","))

	// 4. Handle VIP/Invite Token
	qrType := req.URL.Query().Get("type")
	inviteToken := ""

	if qrType == "vip" {
		cfg, _ := config.Load()
		token, err := utils.GenerateInviteToken(cfg)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to generate invite token")
			return
		}
		inviteToken = "$" + token
	}

	// Construct Protocol String (Version 2)
	// Format: ECK$2$UUID$KEY$URL1,URL2...[$TOKEN]
	qrString := fmt.Sprintf("ECK$2$%s$%s$%s%s", compactUUID, pubKeyHex, connectionString, inviteToken)

	// Generate QR
	png, err := qrcode.Encode(qrString, qrcode.Medium, 512)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate QR")
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Write(png)
}

// DeviceRegisterRequest payload from Android client
type DeviceRegisterRequest struct {
	DeviceID        string `json:"deviceId"`
	DeviceName      string `json:"deviceName"`
	DevicePublicKey string `json:"devicePublicKey"` // Base64
	Signature       string `json:"signature"`       // Base64
	InviteToken     string `json:"inviteToken"`     // Optional JWT for auto-approval
}

// RegisterDevice handles the handshake from a mobile device
func (r *Router) registerDevice(w http.ResponseWriter, req *http.Request) {
	var body DeviceRegisterRequest
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// 1. Verify Payload
	if body.DeviceID == "" || body.DevicePublicKey == "" || body.Signature == "" {
		respondError(w, http.StatusBadRequest, "Missing required fields")
		return
	}

	// 2. Verify Signature
	message := fmt.Sprintf("{\"deviceId\":\"%s\",\"devicePublicKey\":\"%s\"}", body.DeviceID, body.DevicePublicKey)

	valid, err := utils.VerifySignature(body.DevicePublicKey, message, body.Signature)
	if err != nil || !valid {
		respondError(w, http.StatusForbidden, "Invalid signature")
		return
	}

	// 3. Determine Initial Status
	finalStatus := models.DeviceStatusPending

	// Check Invite Token if present
	if body.InviteToken != "" {
		cfg, _ := config.Load()
		claims, err := utils.ValidateToken(body.InviteToken, cfg.JWTSecret)
		if err == nil {
			if typeVal, ok := claims["type"].(string); ok && typeVal == "invite" {
				finalStatus = models.DeviceStatusActive
			}
		}
	}

	// 4. Update/Create Device in DB
	var device models.RegisteredDevice
	// Use Unscoped to find even soft-deleted devices (for restoration on re-registration)
	result := r.db.Unscoped().Where("device_id = ?", body.DeviceID).First(&device)

	if result.Error != nil {
		// Create new device
		newDevice := models.RegisteredDevice{
			DeviceID:   body.DeviceID,
			Name:       body.DeviceName,
			PublicKey:  body.DevicePublicKey,
			Status:     finalStatus,
			LastSeenAt: time.Now(),
		}
		if err := r.db.Create(&newDevice).Error; err != nil {
			respondError(w, http.StatusInternalServerError, "Database error")
			return
		}
	} else {
		// Update existing device (restore if deleted)
		if device.DeletedAt.Valid {
			// Device was deleted, restore it
			device.DeletedAt = gorm.DeletedAt{}
			device.Status = finalStatus // Reset to new status
		} else {
			// Only update status if it was pending and we have a valid token
			// Do NOT unblock blocked devices automatically
			if device.Status == models.DeviceStatusPending && finalStatus == models.DeviceStatusActive {
				device.Status = models.DeviceStatusActive
			}
		}

		device.PublicKey = body.DevicePublicKey
		device.Name = body.DeviceName
		device.LastSeenAt = time.Now()
		// Use Unscoped to save, allowing DeletedAt to be cleared
		r.db.Unscoped().Save(&device)

		finalStatus = device.Status
	}

	// 5. Generate JWT Token if ACTIVE
	var accessToken string
	if finalStatus == models.DeviceStatusActive {
		cfg, _ := config.Load()

		// Create a mock user context for the device
		mockUser := &models.UserAuth{
			ID:       "device_" + body.DeviceID,
			Username: "device_" + body.DeviceID,
			Role:     "device",
			UserType: "individual",
			Email:    "device@" + body.DeviceID + ".local",
		}

		token, _, err := utils.GenerateTokens(mockUser, cfg)
		if err == nil {
			accessToken = token
		}
	}

	// 6. Respond
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"status":  finalStatus,
		"token":   accessToken,
		"message": "Device handshake complete",
	})
}
