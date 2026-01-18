package handlers

import (
	"crypto/ed25519"
	"encoding/base64"
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

// generatePairingQR creates the QR code for device pairing
func (r *Router) generatePairingQR(w http.ResponseWriter, req *http.Request) {
	instanceID := os.Getenv("INSTANCE_ID")
	pubKey := os.Getenv("SERVER_PUBLIC_KEY")
	serverURL := os.Getenv("GLOBAL_SERVER_URL")

	if instanceID == "" || pubKey == "" {
		respondError(w, http.StatusInternalServerError, "Server not configured for pairing")
		return
	}

	if serverURL == "" {
		serverURL = "http://localhost:3001"
	}

	// Compact UUID: Remove dashes and uppercase
	compactUUID := strings.ToUpper(strings.ReplaceAll(instanceID, "-", ""))

	// Decode public key from base64 and convert to hex uppercase
	pubKeyBytes, err := base64.StdEncoding.DecodeString(pubKey)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Invalid public key format")
		return
	}
	pubKeyHex := fmt.Sprintf("%x", pubKeyBytes)

	// Protocol: ECK$1$COMPACTUUID$PUBKEY_HEX$URL
	qrString := "ECK$1$" + compactUUID + "$" + pubKeyHex + "$" + strings.ToUpper(serverURL)

	png, err := qrcode.Encode(qrString, qrcode.Low, 256)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate QR")
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Write(png)
}

// DeviceRegisterRequest represents a device registration request
type DeviceRegisterRequest struct {
	DeviceID        string `json:"deviceId"`
	DeviceName      string `json:"deviceName"`
	DevicePublicKey string `json:"devicePublicKey"` // Base64
	Signature       string `json:"signature"`       // Base64
}

// registerDevice handles the cryptographic pairing handshake
func (r *Router) registerDevice(w http.ResponseWriter, req *http.Request) {
	var body DeviceRegisterRequest
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	// Validate required fields
	if body.DeviceID == "" || body.DevicePublicKey == "" || body.Signature == "" {
		respondError(w, http.StatusBadRequest, "Missing required fields")
		return
	}

	// 1. Verify Signature
	// Message format: {"deviceId":"...","devicePublicKey":"..."}
	msgStr := fmt.Sprintf("{\"deviceId\":\"%s\",\"devicePublicKey\":\"%s\"}", body.DeviceID, body.DevicePublicKey)
	msgBytes := []byte(msgStr)

	sigBytes, err := base64.StdEncoding.DecodeString(body.Signature)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid signature format")
		return
	}

	pubKeyBytes, err := base64.StdEncoding.DecodeString(body.DevicePublicKey)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid public key format")
		return
	}

	if len(pubKeyBytes) != ed25519.PublicKeySize {
		respondError(w, http.StatusBadRequest, "Invalid public key length")
		return
	}

	if len(sigBytes) != ed25519.SignatureSize {
		respondError(w, http.StatusBadRequest, "Invalid signature length")
		return
	}

	if !ed25519.Verify(pubKeyBytes, msgBytes, sigBytes) {
		respondError(w, http.StatusForbidden, "Invalid signature")
		return
	}

	// 2. Register or Update Device
	var device models.RegisteredDevice
	result := r.db.Where("device_id = ?", body.DeviceID).First(&device)

	if result.Error == nil {
		// Update existing
		device.PublicKey = body.DevicePublicKey
		device.IsActive = true
		device.Status = "active"
		if body.DeviceName != "" {
			device.DeviceName = body.DeviceName
		}
		r.db.Save(&device)
	} else {
		// Create new
		device = models.RegisteredDevice{
			DeviceID:   body.DeviceID,
			PublicKey:  body.DevicePublicKey,
			DeviceName: body.DeviceName,
			IsActive:   true,
			Status:     "active",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		r.db.Create(&device)
	}

	// 3. Generate Token for the device (Android requires this for subsequent calls)
	cfg, _ := config.Load()

	// Create a dummy user struct for token generation
	// The device acts as a user "device_[id]"
	mockUser := &models.UserAuth{
		ID:    "device_" + body.DeviceID,
		Role:  "device",
		Email: "device@" + body.DeviceID,
	}

	accessToken, _, err := utils.GenerateTokens(mockUser, cfg)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate device token")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "active",
		"token":   accessToken,
		"message": "Device registered and authorized",
	})
}
