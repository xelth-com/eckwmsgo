package sync

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/xelth-com/eckwmsgo/internal/config"
	"github.com/xelth-com/eckwmsgo/internal/database"
	"github.com/xelth-com/eckwmsgo/internal/models"
)

// NewHTTPClient creates IPv4-only HTTP client for mesh operations
func NewHTTPClient() *http.Client {
	ipv4Dialer := &net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	return &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return ipv4Dialer.DialContext(ctx, "tcp4", addr)
			},
			MaxIdleConns:       100,
			IdleConnTimeout:    90 * time.Second,
			DisableCompression: true,
		},
	}
}

// PushDeviceToNode pushes a device to a bootstrap node via POST /api/mesh/push
func PushDeviceToNode(db *database.DB, device models.RegisteredDevice, nodeURL, nodeID string) error {
	// Create push data
	pushData := map[string]interface{}{
		"entity_type": "device",
		"entity_id":   device.DeviceID,
		"operation":   "upsert",
		"data": map[string]interface{}{
			"device_id":    device.DeviceID,
			"device_name":  device.Name,
			"public_key":   device.PublicKey,
			"status":       string(device.Status),
			"last_seen_at": device.LastSeenAt.Format(time.RFC3339),
			"created_at":   device.CreatedAt.Format(time.RFC3339),
			"updated_at":   device.UpdatedAt.Format(time.RFC3339),
		},
		"node_id": nodeID,
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(pushData)
	if err != nil {
		return fmt.Errorf("failed to marshal push data: %w", err)
	}

	// Create POST request
	pushURL := nodeURL + "/api/mesh/push"
	req, err := http.NewRequest("POST", pushURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Node-ID", nodeID)
	req.Header.Set("X-Instance-ID", nodeID) // Use nodeID as instanceID for relay

	// Use IPv4-only client
	client := NewHTTPClient()

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send push: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("push failed: HTTP %d, response: %s", resp.StatusCode, string(body))
	}

	log.Printf("✅ Device %s pushed successfully to %s (HTTP %d)", device.DeviceID, pushURL, resp.StatusCode)

	// Update device last_seen_at
	db.Model(&device).Update("last_seen_at").Error

	return nil
}
