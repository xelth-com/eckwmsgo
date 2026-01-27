package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

// SyncConfig holds synchronization configuration
type SyncConfig struct {
	// ============ BASIC SETTINGS ============
	Enabled      bool   `json:"enabled"`
	Role         string `json:"role"`          // master, peer, edge, blind_relay
	Mode         string `json:"mode"`          // full, incremental, selective, cache, master
	CacheProfile string `json:"cache_profile"` // full, extended, standard, light, minimal
	Direction    string `json:"direction"`     // bidirectional, pull_only, push_only

	// ============ SCHEDULING ============
	AutoSyncEnabled  bool   `json:"auto_sync_enabled"`
	AutoSyncInterval int    `json:"auto_sync_interval"` // seconds
	SyncOnStartup    bool   `json:"sync_on_startup"`
	Schedule         string `json:"schedule"` // cron format

	// ============ LIMITS ============
	MaxSyncSize int `json:"max_sync_size"` // MB
	SyncTimeout int `json:"sync_timeout"`  // seconds
	MaxRetries  int `json:"max_retries"`
	BatchSize   int `json:"batch_size"`

	// ============ ENTITIES ============
	Entities map[string]EntitySyncConfig `json:"entities"`

	// ============ REALTIME ============
	Realtime RealtimeConfig `json:"realtime"`

	// ============ CONFLICTS ============
	ConflictResolution string `json:"conflict_resolution"` // server_wins, client_wins, last_write_wins, manual, priority_based

	// ============ OPTIMIZATION ============
	CompressionEnabled bool `json:"compression_enabled"`
	DeltaSyncEnabled   bool `json:"delta_sync_enabled"`
	EagerLoadRelations bool `json:"eager_load_relations"`
	ParallelSync       bool `json:"parallel_sync"`
	ParallelWorkers    int  `json:"parallel_workers"`

	// ============ CACHE ============
	LocalCacheEnabled   bool   `json:"local_cache_enabled"`
	CacheTTL            int    `json:"cache_ttl"`             // seconds, 0 = infinite
	MaxCacheSize        int    `json:"max_cache_size"`        // MB
	CacheEvictionPolicy string `json:"cache_eviction_policy"` // lru, fifo, lfu

	// ============ SECURITY ============
	EncryptionEnabled     bool `json:"encryption_enabled"`
	SignatureVerification bool `json:"signature_verification"`

	// ============ ROUTES ============
	Routes []SyncRouteConfig `json:"routes"`
}

// EntitySyncConfig holds sync configuration for a specific entity type
type EntitySyncConfig struct {
	Enabled      bool         `json:"enabled"`
	Strategy     string       `json:"strategy"` // full, active_only, time_window, filtered, metadata_only
	Filters      []SyncFilter `json:"filters"`
	HistoryDepth int          `json:"history_depth"` // days, 0 = all history
	MaxRecords   int          `json:"max_records"`   // 0 = no limit
	SyncInterval int          `json:"sync_interval"` // seconds
	Priority     int          `json:"priority"`      // 1-10, where 10 = highest
}

// SyncFilter represents a filter for selective sync
type SyncFilter struct {
	Field         string      `json:"field"`
	Operator      string      `json:"operator"` // eq, ne, gt, lt, gte, lte, in, not_in, like
	Value         interface{} `json:"value"`
	LogicOperator string      `json:"logic_operator"` // AND, OR
}

// RealtimeConfig holds realtime sync configuration
type RealtimeConfig struct {
	Enabled             bool     `json:"enabled"`
	Events              []string `json:"events"`      // create, update, delete
	Entities            []string `json:"entities"`    // which entities to track
	BufferTime          int      `json:"buffer_time"` // ms
	AutoSyncOnReconnect bool     `json:"auto_sync_on_reconnect"`
}

// SyncRouteConfig represents a sync route
type SyncRouteConfig struct {
	URL      string `json:"url"`
	Type     string `json:"type"`     // primary, fallback, web
	Timeout  int    `json:"timeout"`  // seconds
	Priority int    `json:"priority"` // lower = higher priority
}

// LoadSyncConfig loads sync configuration from environment or file
func LoadSyncConfig() *SyncConfig {
	// Try to load from file first
	if configPath := os.Getenv("SYNC_CONFIG_PATH"); configPath != "" {
		if cfg, err := loadSyncConfigFromFile(configPath); err == nil {
			return cfg
		}
	}

	// Otherwise use defaults
	return getDefaultSyncConfig()
}

// loadSyncConfigFromFile loads sync config from JSON file
func loadSyncConfigFromFile(path string) (*SyncConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg SyncConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// getDefaultSyncConfig returns default sync configuration
func getDefaultSyncConfig() *SyncConfig {
	return &SyncConfig{
		Enabled:      getBoolEnv("SYNC_ENABLED", true),
		Role:         getEnv("SYNC_ROLE", "peer"), // Default to peer (trusted node)
		Mode:         getEnv("SYNC_MODE", "incremental"),
		CacheProfile: getEnv("SYNC_CACHE_PROFILE", "standard"),
		Direction:    getEnv("SYNC_DIRECTION", "bidirectional"),

		AutoSyncEnabled:  getBoolEnv("SYNC_AUTO_ENABLED", true),
		AutoSyncInterval: getIntEnv("SYNC_AUTO_INTERVAL", 300),
		SyncOnStartup:    getBoolEnv("SYNC_ON_STARTUP", true),

		MaxSyncSize: getIntEnv("SYNC_MAX_SIZE", 50),
		SyncTimeout: getIntEnv("SYNC_TIMEOUT", 300),
		MaxRetries:  getIntEnv("SYNC_MAX_RETRIES", 3),
		BatchSize:   getIntEnv("SYNC_BATCH_SIZE", 100),

		Entities: getDefaultEntityConfigs(),

		Realtime: RealtimeConfig{
			Enabled:             getBoolEnv("SYNC_REALTIME_ENABLED", true),
			Events:              []string{"create", "update", "delete"},
			Entities:            []string{"items", "boxes", "places", "orders"},
			BufferTime:          getIntEnv("SYNC_REALTIME_BUFFER", 1000),
			AutoSyncOnReconnect: true,
		},

		ConflictResolution: getEnv("SYNC_CONFLICT_RESOLUTION", "priority_based"),

		CompressionEnabled: getBoolEnv("SYNC_COMPRESSION", true),
		DeltaSyncEnabled:   getBoolEnv("SYNC_DELTA", true),
		EagerLoadRelations: getBoolEnv("SYNC_EAGER_LOAD", false),
		ParallelSync:       getBoolEnv("SYNC_PARALLEL", true),
		ParallelWorkers:    getIntEnv("SYNC_WORKERS", 2),

		LocalCacheEnabled:   getBoolEnv("SYNC_CACHE_ENABLED", true),
		CacheTTL:            getIntEnv("SYNC_CACHE_TTL", 3600),
		MaxCacheSize:        getIntEnv("SYNC_CACHE_SIZE", 100),
		CacheEvictionPolicy: getEnv("SYNC_CACHE_POLICY", "lru"),

		EncryptionEnabled:     getBoolEnv("SYNC_ENCRYPTION", false),
		SignatureVerification: getBoolEnv("SYNC_SIGNATURE_VERIFY", true),

		Routes: getDefaultRoutes(),
	}
}

// getDefaultEntityConfigs returns default entity sync configs
func getDefaultEntityConfigs() map[string]EntitySyncConfig {
	return map[string]EntitySyncConfig{
		// New Odoo-aligned entity types
		"products": {
			Enabled:      true,
			Strategy:     "checksum", // Use checksum-based sync
			HistoryDepth: 0,
			MaxRecords:   0,
			SyncInterval: 60,
			Priority:     10,
		},
		"locations": {
			Enabled:      true,
			Strategy:     "checksum", // Use checksum-based sync
			HistoryDepth: 0,
			MaxRecords:   0,
			SyncInterval: 300,
			Priority:     8,
		},
		"quants": {
			Enabled:      true,
			Strategy:     "checksum", // Use checksum-based sync
			HistoryDepth: 0,
			MaxRecords:   0,
			SyncInterval: 60,
			Priority:     9,
		},
		"lots": {
			Enabled:      true,
			Strategy:     "checksum",
			HistoryDepth: 0,
			MaxRecords:   0,
			SyncInterval: 120,
			Priority:     7,
		},
		"packages": {
			Enabled:      true,
			Strategy:     "checksum",
			HistoryDepth: 0,
			MaxRecords:   0,
			SyncInterval: 60,
			Priority:     8,
		},
		"pickings": {
			Enabled:      true,
			Strategy:     "checksum",
			HistoryDepth: 0,
			MaxRecords:   0,
			SyncInterval: 120,
			Priority:     9,
		},
		"partners": {
			Enabled:      true,
			Strategy:     "checksum",
			HistoryDepth: 0,
			MaxRecords:   0,
			SyncInterval: 300,
			Priority:     6,
		},

		// Legacy entity types (kept for backward compatibility, disabled by default)
		"items": {
			Enabled:      false, // Deprecated, use "products"
			Strategy:     "active_only",
			HistoryDepth: 0,
			MaxRecords:   0,
			SyncInterval: 60,
			Priority:     10,
		},
		"boxes": {
			Enabled:      false, // Deprecated, use "packages"
			Strategy:     "active_only",
			HistoryDepth: 0,
			MaxRecords:   0,
			SyncInterval: 60,
			Priority:     9,
		},
		"places": {
			Enabled:      false, // Deprecated, use "locations"
			Strategy:     "full",
			HistoryDepth: 0,
			MaxRecords:   0,
			SyncInterval: 300,
			Priority:     8,
		},
		"racks": {
			Enabled:      false,
			Strategy:     "full",
			HistoryDepth: 0,
			MaxRecords:   0,
			SyncInterval: 600,
			Priority:     7,
		},
		"warehouses": {
			Enabled:      false,
			Strategy:     "full",
			HistoryDepth: 0,
			MaxRecords:   0,
			SyncInterval: 600,
			Priority:     7,
		},
		"orders": {
			Enabled:      false,
			Strategy:     "time_window",
			HistoryDepth: 90,
			MaxRecords:   0,
			SyncInterval: 120,
			Priority:     10,
		},
		"users": {
			Enabled:      false,
			Strategy:     "active_only",
			HistoryDepth: 0,
			MaxRecords:   0,
			SyncInterval: 300,
			Priority:     5,
		},
		"devices": {
			Enabled:      false,
			Strategy:     "active_only",
			HistoryDepth: 0,
			MaxRecords:   0,
			SyncInterval: 300,
			Priority:     6,
		},
		"shipments": {
			Enabled:      true, // ENABLED
			Strategy:     "checksum", // Changed from time_window for robust sync
			HistoryDepth: 30,
			MaxRecords:   0,
			SyncInterval: 60,
			Priority:     9,
		},
		"tracking": {
			Enabled:      true, // ENABLED
			Strategy:     "checksum", // Changed from time_window for robust sync
			HistoryDepth: 30,
			MaxRecords:   0,
			SyncInterval: 60,
			Priority:     8,
		},
	}
}

// getDefaultRoutes returns default sync routes
func getDefaultRoutes() []SyncRouteConfig {
	routes := []SyncRouteConfig{}

	// Primary local server
	if localURL := os.Getenv("LOCAL_SERVER_INTERNAL_URL"); localURL != "" {
		log.Printf("🔗 Adding primary sync route: %s", localURL)
		routes = append(routes, SyncRouteConfig{
			URL:      localURL,
			Type:     "primary",
			Timeout:  10,
			Priority: 1,
		})
	}

	// Fallback web server
	if globalURL := os.Getenv("GLOBAL_SERVER_URL"); globalURL != "" {
		log.Printf("🔗 Adding web sync route: %s", globalURL)
		routes = append(routes, SyncRouteConfig{
			URL:      globalURL,
			Type:     "web",
			Timeout:  15,
			Priority: 2,
		})
	}

	if len(routes) == 0 {
		log.Println("⚠️ No sync routes configured (LOCAL_SERVER_INTERNAL_URL and GLOBAL_SERVER_URL not set)")
	}

	return routes
}

// Helper functions for environment variables

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1" || value == "yes"
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var result int
		if _, err := fmt.Sscanf(value, "%d", &result); err == nil {
			return result
		}
	}
	return defaultValue
}
