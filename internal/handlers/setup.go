package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/xelth-com/eckwmsgo/internal/config"
	"github.com/xelth-com/eckwmsgo/internal/models"
	"github.com/xelth-com/eckwmsgo/internal/utils"
	"github.com/skip2/go-qrcode"
)

// GeneratePairingQR generates a QR code for device pairing (ECK-P1-ALPHA protocol)
// Protocol: ECK$1$COMPACTUUID$PUBKEY_HEX$URL
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

	// 3. URL (from Config or Host header)
	// Prefer environment variable, fallback to request host
	serverURL := os.Getenv("GLOBAL_SERVER_URL")
	if serverURL == "" {
		scheme := "http"
		if req.TLS != nil {
			scheme = "https"
		}
		serverURL = fmt.Sprintf("%s://%s", scheme, req.Host)
	}
	serverURL = strings.ToUpper(serverURL)

	// Construct Protocol String
	qrString := fmt.Sprintf("ECK$1$%s$%s$%s", compactUUID, pubKeyHex, serverURL)

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
	// The client signs the JSON string: {"deviceId":"...","devicePublicKey":"..."}
	// We must recreate this string EXACTLY as the client does.
	message := fmt.Sprintf("{\"deviceId\":\"%s\",\"devicePublicKey\":\"%s\"}", body.DeviceID, body.DevicePublicKey)

	valid, err := utils.VerifySignature(body.DevicePublicKey, message, body.Signature)
	if err != nil || !valid {
		respondError(w, http.StatusForbidden, "Invalid signature")
		return
	}

	// 3. Update/Create Device in DB
	var device models.RegisteredDevice
	result := r.db.First(&device, "device_id = ?", body.DeviceID)

	var finalStatus models.DeviceStatus

	if result.Error != nil {
		// Create new device (Pending by default)
		finalStatus = models.DeviceStatusPending
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
		// Update existing device
		// We DO NOT change the status here. If it was blocked, it stays blocked.
		// We update keys and name in case they changed (re-install app).
		finalStatus = device.Status
		device.PublicKey = body.DevicePublicKey
		device.Name = body.DeviceName
		device.LastSeenAt = time.Now()
		r.db.Save(&device)
	}

	// 4. Generate JWT Token if ACTIVE
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

	// 5. Respond
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"status":  finalStatus,
		"token":   accessToken, // Will be empty if not active
		"message": "Device handshake complete",
	})
}
