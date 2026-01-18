package mesh

import (
	"sync"
	"time"
)

// Registry holds the list of known mesh nodes
type Registry struct {
	Nodes map[string]*NodeInfo
	mu    sync.RWMutex
}

// GlobalRegistry is the shared node registry
var GlobalRegistry = &Registry{
	Nodes: make(map[string]*NodeInfo),
}

// RegisterNode adds or updates a node in the registry
func (r *Registry) RegisterNode(info NodeInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()

	info.LastSeen = time.Now()
	info.IsOnline = true
	r.Nodes[info.InstanceID] = &info
}

// GetNodes returns all registered nodes
func (r *Registry) GetNodes() []*NodeInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	nodes := make([]*NodeInfo, 0, len(r.Nodes))
	for _, node := range r.Nodes {
		nodes = append(nodes, node)
	}
	return nodes
}

// GetBestNode returns the online node with the highest weight
func (r *Registry) GetBestNode() *NodeInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var best *NodeInfo
	for _, node := range r.Nodes {
		if !node.IsOnline {
			continue
		}
		if best == nil || node.Weight > best.Weight {
			best = node
		}
	}
	return best
}

// MarkOffline marks a node as offline
func (r *Registry) MarkOffline(instanceID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if node, exists := r.Nodes[instanceID]; exists {
		node.IsOnline = false
	}
}
