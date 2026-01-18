package mesh

import "time"

// NodeInfo represents a node in the mesh network
type NodeInfo struct {
	InstanceID string    `json:"instance_id"`
	Role       string    `json:"role"`
	BaseURL    string    `json:"base_url"`
	Weight     int       `json:"weight"`
	LastSeen   time.Time `json:"last_seen"`
	IsOnline   bool      `json:"is_online"`
}
