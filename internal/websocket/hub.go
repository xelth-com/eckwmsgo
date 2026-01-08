package websocket

import (
	"encoding/json"
	"log"
	"sync"
)

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	// Registered clients map: DeviceID -> Client
	clients map[string]*Client

	// Register requests
	register chan *Client

	// Unregister requests
	unregister chan *Client

	// Mutex for thread-safe access to clients map
	mu sync.RWMutex
}

// NewHub creates a new Hub instance
func NewHub() *Hub {
	return &Hub{
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[string]*Client),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if client.DeviceID != "" {
				// If device connects again, close old connection
				if old, ok := h.clients[client.DeviceID]; ok {
					close(old.send)
					delete(h.clients, client.DeviceID)
				}
				h.clients[client.DeviceID] = client
				log.Printf("📱 Device connected: %s", client.DeviceID)
			}
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if client.DeviceID != "" {
				if _, ok := h.clients[client.DeviceID]; ok {
					delete(h.clients, client.DeviceID)
					close(client.send)
					log.Printf("📴 Device disconnected: %s", client.DeviceID)
				}
			}
			h.mu.Unlock()
		}
	}
}

// SendToDevice sends a message to a specific device
func (h *Hub) SendToDevice(deviceID string, message interface{}) bool {
	h.mu.RLock()
	client, ok := h.clients[deviceID]
	h.mu.RUnlock()

	if !ok {
		return false
	}

	jsonMsg, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return false
	}

	select {
	case client.send <- jsonMsg:
		return true
	default:
		// Buffer full or client dead
		return false
	}
}
