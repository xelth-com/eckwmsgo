package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/xelth-com/eckwmsgo/internal/config"
	"github.com/xelth-com/eckwmsgo/internal/mesh"
	"github.com/gorilla/mux"
)

// handleHandshake processes mesh node handshake requests
func (r *Router) handleHandshake(w http.ResponseWriter, req *http.Request) {
	authHeader := req.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "Missing Authorization Header", http.StatusUnauthorized)
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	cfg, _ := config.Load()

	// Validate Token
	peerInfo, err := mesh.ValidateNodeToken(tokenString, cfg.MeshSecret)
	if err != nil {
		http.Error(w, "Invalid Token: "+err.Error(), http.StatusForbidden)
		return
	}

	// Register Peer
	mesh.GlobalRegistry.RegisterNode(*peerInfo)

	// Respond with My Info
	myInfo := mesh.NodeInfo{
		InstanceID: cfg.InstanceID,
		Role:       string(cfg.NodeRole),
		BaseURL:    cfg.BaseURL,
		Weight:     cfg.NodeWeight,
		IsOnline:   true,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(myInfo)
}

// listMeshNodes returns the list of known mesh nodes
func (r *Router) listMeshNodes(w http.ResponseWriter, req *http.Request) {
	nodes := mesh.GlobalRegistry.GetNodes()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(nodes)
}

// getMeshNodeStatus returns status of a specific node
func (r *Router) getMeshNodeStatus(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	instanceID := vars["id"]

	nodes := mesh.GlobalRegistry.GetNodes()
	for _, node := range nodes {
		if node.InstanceID == instanceID {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(node)
			return
		}
	}
	http.Error(w, "Node not found", http.StatusNotFound)
}
