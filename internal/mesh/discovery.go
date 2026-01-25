package mesh

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/xelth-com/eckwmsgo/internal/config"
)

// StartDiscovery begins the mesh node discovery process
func StartDiscovery(cfg *config.Config) {
	if len(cfg.BootstrapNodes) == 0 {
		log.Printf("Mesh: No bootstrap nodes configured, running in standalone mode")
		return
	}

	go func() {
		// Initial delay to allow server to start
		time.Sleep(2 * time.Second)

		for {
			for _, nodeURL := range cfg.BootstrapNodes {
				if nodeURL == "" {
					continue
				}

				// Ensure URL doesn't end with slash
				target := strings.TrimRight(nodeURL, "/")
				handshakeURL := target + "/mesh/handshake"

				token, err := GenerateNodeToken(cfg)
				if err != nil {
					log.Printf("Mesh: Failed to generate token: %v", err)
					continue
				}

				// Send my info
				myInfo := NodeInfo{
					InstanceID: cfg.InstanceID,
					Role:       string(cfg.NodeRole),
					BaseURL:    cfg.BaseURL,
					Weight:     cfg.NodeWeight,
				}
				body, _ := json.Marshal(myInfo)

				req, _ := http.NewRequest("POST", handshakeURL, io.NopCloser(bytes.NewBuffer(body)))
				req.Header.Set("Authorization", "Bearer "+token)
				req.Header.Set("Content-Type", "application/json")

				client := &http.Client{Timeout: 5 * time.Second}
				resp, err := client.Do(req)

				if err == nil && resp.StatusCode == 200 {
					var peerInfo NodeInfo
					if err := json.NewDecoder(resp.Body).Decode(&peerInfo); err == nil {
						// Check if this is ourselves (same INSTANCE_ID)
						if peerInfo.InstanceID == cfg.InstanceID {
							log.Printf("Mesh: Skipping self-connection (same INSTANCE_ID: %s)", peerInfo.InstanceID)
							resp.Body.Close()
							continue
						}

						GlobalRegistry.RegisterNode(peerInfo)
						log.Printf("Mesh: Handshake success with %s (%s, ID: %s)", peerInfo.BaseURL, peerInfo.Role, peerInfo.InstanceID)
					}
					resp.Body.Close()
				} else if err != nil {
					log.Printf("Mesh: Failed to contact %s: %v", target, err)
				}
			}
			time.Sleep(30 * time.Second)
		}
	}()
}
