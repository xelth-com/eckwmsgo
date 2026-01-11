package sync

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// RouteType represents the type of sync route
type RouteType string

const (
	RouteTypePrimary  RouteType = "primary"
	RouteTypeFallback RouteType = "fallback"
	RouteTypeWeb      RouteType = "web"
)

// SyncRouteConfig represents a configured sync route
type SyncRouteConfig struct {
	URL          string    `json:"url"`
	Type         RouteType `json:"type"`
	Timeout      int       `json:"timeout"` // seconds
	Priority     int       `json:"priority"` // lower = higher priority
}

// RouteSwitch tracks when routes are switched
type RouteSwitch struct {
	FromRoute string
	ToRoute   string
	Reason    string
	Timestamp time.Time
}

// RouteStatus tracks the health of a route
type RouteStatus struct {
	URL           string
	IsAvailable   bool
	LastCheck     time.Time
	LastSuccess   *time.Time
	LastFailure   *time.Time
	SuccessCount  int
	FailureCount  int
	AvgLatency    time.Duration
	LatencySum    time.Duration
	LatencyCount  int
}

// ConnectionManager manages sync connections with fallback support
type ConnectionManager struct {
	mu sync.RWMutex

	// Configuration
	instanceID string
	routes     []SyncRouteConfig

	// Current state
	currentRoute  string
	routeStatuses map[string]*RouteStatus
	routeHistory  []RouteSwitch
	isOnline      bool

	// Health check
	healthCheckInterval time.Duration
	healthCheckRunning  bool
	stopHealthCheck     chan struct{}

	// HTTP client
	httpClient *http.Client
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager(instanceID string, routes []SyncRouteConfig) *ConnectionManager {
	cm := &ConnectionManager{
		instanceID:          instanceID,
		routes:              routes,
		routeStatuses:       make(map[string]*RouteStatus),
		routeHistory:        make([]RouteSwitch, 0),
		isOnline:            false,
		healthCheckInterval: 30 * time.Second,
		stopHealthCheck:     make(chan struct{}),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	// Initialize route statuses
	for _, route := range routes {
		cm.routeStatuses[route.URL] = &RouteStatus{
			URL:         route.URL,
			IsAvailable: false,
			LastCheck:   time.Time{},
		}
	}

	return cm
}

// Start begins health checking
func (cm *ConnectionManager) Start() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.healthCheckRunning {
		return
	}

	cm.healthCheckRunning = true
	go cm.healthCheckLoop()
}

// Stop stops health checking
func (cm *ConnectionManager) Stop() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if !cm.healthCheckRunning {
		return
	}

	cm.healthCheckRunning = false
	close(cm.stopHealthCheck)
}

// SelectRoute selects the best available route
func (cm *ConnectionManager) SelectRoute() string {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Try routes in priority order
	for _, route := range cm.routes {
		if cm.testConnection(route.URL, route.Timeout) {
			if cm.currentRoute != route.URL {
				cm.logRouteSwitch(cm.currentRoute, route.URL, "route_available")
				cm.currentRoute = route.URL
			}
			cm.isOnline = true
			return route.URL
		}
	}

	// No routes available
	cm.isOnline = false
	if cm.currentRoute != "offline" {
		cm.logRouteSwitch(cm.currentRoute, "offline", "all_routes_unavailable")
		cm.currentRoute = "offline"
	}
	return "offline"
}

// GetCurrentRoute returns the currently selected route
func (cm *ConnectionManager) GetCurrentRoute() string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.currentRoute
}

// IsOnline returns whether any route is available
func (cm *ConnectionManager) IsOnline() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.isOnline
}

// GetRouteStatus returns the status of a specific route
func (cm *ConnectionManager) GetRouteStatus(url string) *RouteStatus {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.routeStatuses[url]
}

// GetAllRouteStatuses returns all route statuses
func (cm *ConnectionManager) GetAllRouteStatuses() map[string]*RouteStatus {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	result := make(map[string]*RouteStatus)
	for k, v := range cm.routeStatuses {
		result[k] = v
	}
	return result
}

// GetRouteHistory returns the route switch history
func (cm *ConnectionManager) GetRouteHistory() []RouteSwitch {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.routeHistory
}

// testConnection tests if a route is available
func (cm *ConnectionManager) testConnection(url string, timeout int) bool {
	// Create custom client with specific timeout
	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	status := cm.routeStatuses[url]
	status.LastCheck = time.Now()

	start := time.Now()
	resp, err := client.Get(url + "/health")
	latency := time.Since(start)

	if err != nil {
		// Connection failed
		status.IsAvailable = false
		status.FailureCount++
		now := time.Now()
		status.LastFailure = &now
		log.Printf("Route %s failed: %v", url, err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		// Connection successful
		status.IsAvailable = true
		status.SuccessCount++
		now := time.Now()
		status.LastSuccess = &now
		status.FailureCount = 0 // Reset failure count on success

		// Update latency statistics
		status.LatencySum += latency
		status.LatencyCount++
		status.AvgLatency = status.LatencySum / time.Duration(status.LatencyCount)

		return true
	}

	// Non-200 status code
	status.IsAvailable = false
	status.FailureCount++
	now := time.Now()
	status.LastFailure = &now
	log.Printf("Route %s returned status %d", url, resp.StatusCode)
	return false
}

// logRouteSwitch logs a route switch
func (cm *ConnectionManager) logRouteSwitch(fromRoute, toRoute, reason string) {
	if fromRoute == toRoute {
		return
	}

	switchEvent := RouteSwitch{
		FromRoute: fromRoute,
		ToRoute:   toRoute,
		Reason:    reason,
		Timestamp: time.Now(),
	}

	cm.routeHistory = append(cm.routeHistory, switchEvent)

	// Keep only last 100 switches
	if len(cm.routeHistory) > 100 {
		cm.routeHistory = cm.routeHistory[len(cm.routeHistory)-100:]
	}

	log.Printf("Route switched: %s -> %s (reason: %s)", fromRoute, toRoute, reason)
}

// healthCheckLoop periodically checks route health
func (cm *ConnectionManager) healthCheckLoop() {
	ticker := time.NewTicker(cm.healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cm.checkAllRoutes()
		case <-cm.stopHealthCheck:
			return
		}
	}
}

// checkAllRoutes checks health of all routes
func (cm *ConnectionManager) checkAllRoutes() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// If we're offline, try to reconnect to primary
	if !cm.isOnline || cm.currentRoute == "offline" {
		for _, route := range cm.routes {
			if cm.testConnection(route.URL, route.Timeout) {
				cm.logRouteSwitch(cm.currentRoute, route.URL, "health_check_reconnect")
				cm.currentRoute = route.URL
				cm.isOnline = true
				return
			}
		}
		return
	}

	// If we're using fallback, check if primary is back
	if cm.currentRoute != "" && cm.currentRoute != "offline" {
		for _, route := range cm.routes {
			// Check if there's a higher priority route available
			if route.Priority < cm.getCurrentRoutePriority() {
				if cm.testConnection(route.URL, route.Timeout) {
					cm.logRouteSwitch(cm.currentRoute, route.URL, "primary_restored")
					cm.currentRoute = route.URL
					return
				}
			}
		}
	}

	// Check current route health
	currentRouteConfig := cm.getRouteConfig(cm.currentRoute)
	if currentRouteConfig != nil {
		if !cm.testConnection(cm.currentRoute, currentRouteConfig.Timeout) {
			// Current route failed, select new one
			newRoute := cm.selectBestRoute()
			if newRoute != cm.currentRoute {
				cm.logRouteSwitch(cm.currentRoute, newRoute, "current_route_failed")
				cm.currentRoute = newRoute
				if newRoute == "offline" {
					cm.isOnline = false
				}
			}
		}
	}
}

// getCurrentRoutePriority returns the priority of the current route
func (cm *ConnectionManager) getCurrentRoutePriority() int {
	for _, route := range cm.routes {
		if route.URL == cm.currentRoute {
			return route.Priority
		}
	}
	return 999 // Very low priority if not found
}

// getRouteConfig gets the config for a specific route URL
func (cm *ConnectionManager) getRouteConfig(url string) *SyncRouteConfig {
	for _, route := range cm.routes {
		if route.URL == url {
			return &route
		}
	}
	return nil
}

// selectBestRoute selects the best available route
func (cm *ConnectionManager) selectBestRoute() string {
	for _, route := range cm.routes {
		status := cm.routeStatuses[route.URL]
		if status != nil && status.IsAvailable {
			return route.URL
		}
	}
	return "offline"
}

// SendWithRetry sends a request with retry logic
func (cm *ConnectionManager) SendWithRetry(path string, data interface{}, maxRetries int) error {
	retryCount := 0
	baseDelay := 2 * time.Second

	for retryCount <= maxRetries {
		route := cm.SelectRoute()
		if route == "offline" {
			return fmt.Errorf("no routes available")
		}

		// TODO: Actually send the request
		// For now, just simulate
		fullURL := route + path
		log.Printf("Sending to %s (attempt %d/%d)", fullURL, retryCount+1, maxRetries+1)

		// Simulate request
		// err := cm.sendRequest(fullURL, data)
		// if err == nil {
		//     return nil
		// }

		// If failed, wait before retry with exponential backoff
		if retryCount < maxRetries {
			delay := baseDelay * time.Duration(1<<uint(retryCount)) // 2s, 4s, 8s, 16s
			log.Printf("Request failed, retrying in %v...", delay)
			time.Sleep(delay)
		}

		retryCount++
	}

	return fmt.Errorf("max retries exceeded")
}

// SetHealthCheckInterval sets the health check interval
func (cm *ConnectionManager) SetHealthCheckInterval(interval time.Duration) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.healthCheckInterval = interval
}
