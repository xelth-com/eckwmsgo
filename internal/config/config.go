package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// NodeRole represents the role of a node in the mesh network
type NodeRole string

const (
	RoleMaster     NodeRole = "master"
	RolePeer       NodeRole = "peer"
	RoleEdge       NodeRole = "edge"
	RoleBlindRelay NodeRole = "blind_relay"
)

// Config holds all application configuration
type Config struct {
	NodeEnv     string
	Port        string
	PathPrefix  string
	JWTSecret   string
	EncKey      string

	// Mesh Configuration
	InstanceID     string
	NodeRole       NodeRole
	NodeWeight     int
	BaseURL        string
	MeshSecret     string
	BootstrapNodes []string

	Database    DatabaseConfig
	Translation TranslationConfig
	Server      ServerConfig
	Odoo        OdooConfig
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	Database string
	Alter    bool
}

// TranslationConfig holds translation configuration
type TranslationConfig struct {
	DefaultLanguage    string
	TranslationDomain  string
	OpenAIAPIKey       string
}

// ServerConfig holds server configuration
type ServerConfig struct {
	LocalPort             string
	GlobalPort            string
	GlobalURL             string
	LocalInternalURL      string
	GlobalAPIEndpoint     string
	GlobalAPIKey          string
	InstanceID            string
	ServerPublicKey       string
	ServerPrivateKey      string
	GlobalRegisterURL     string
}

// OdooConfig holds Odoo ERP connection settings
type OdooConfig struct {
	URL          string
	Database     string
	Username     string
	Password     string
	SyncInterval int // in minutes
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	pathPrefix := os.Getenv("HTTP_PATH_PREFIX")
	if pathPrefix != "" && !strings.HasPrefix(pathPrefix, "/") {
		pathPrefix = "/" + pathPrefix
	}
	pathPrefix = strings.TrimRight(pathPrefix, "/")

	// Support DATABASE_URL format
	databaseURL := os.Getenv("DATABASE_URL")

	return &Config{
		NodeEnv:    getEnv("NODE_ENV", "development"),
		Port:       getEnv("PORT", "3210"),
		PathPrefix: pathPrefix,
		JWTSecret:  jwtSecret,
		EncKey:     os.Getenv("ENC_KEY"),

		// Mesh Configuration
		InstanceID:     getEnv("INSTANCE_ID", "unknown_instance"),
		NodeRole:       NodeRole(getEnv("NODE_ROLE", "edge")),
		NodeWeight:     getIntEnv("NODE_WEIGHT", 10),
		BaseURL:        strings.TrimRight(getEnv("BASE_URL", "http://localhost:3210"), "/"),
		MeshSecret:     getEnv("MESH_SECRET", "change_me_to_something_secure"),
		BootstrapNodes: parseBootstrapNodes(os.Getenv("BOOTSTRAP_NODES")),

		Database: func() DatabaseConfig {
			if databaseURL != "" {
				return DatabaseConfig{
					Host:     getEnv("PG_HOST", "localhost"),
					Port:     getEnv("PG_PORT", "5432"),
					Username: getEnv("PG_USERNAME", "postgres"),
					Password: os.Getenv("PG_PASSWORD"),
					Database: getEnv("PG_DATABASE", "eckwms"),
					Alter:    getEnv("DB_ALTER", "false") == "true",
				}
			}
			return DatabaseConfig{
				Host:     getEnv("PG_HOST", "localhost"),
				Port:     getEnv("PG_PORT", "5432"),
				Username: getEnv("PG_USERNAME", "postgres"),
				Password: os.Getenv("PG_PASSWORD"),
				Database: getEnv("PG_DATABASE", "eckwms"),
				Alter:    getEnv("DB_ALTER", "false") == "true",
			}
		}(),
		Translation: TranslationConfig{
			DefaultLanguage:   getEnv("DEFAULT_LANGUAGE", "en"),
			TranslationDomain: os.Getenv("TRANSLATION_DOMAIN"),
			OpenAIAPIKey:      os.Getenv("OPENAI_API_KEY"),
		},
		Server: ServerConfig{
			LocalPort:         getEnv("LOCAL_SERVER_PORT", "3210"),
			GlobalPort:        getEnv("GLOBAL_SERVER_PORT", "8080"),
			GlobalURL:         os.Getenv("GLOBAL_SERVER_URL"),
			LocalInternalURL:  os.Getenv("LOCAL_SERVER_INTERNAL_URL"),
			GlobalAPIEndpoint: os.Getenv("GLOBAL_SERVER_API_ENDPOINT"),
			GlobalAPIKey:      os.Getenv("GLOBAL_SERVER_API_KEY"),
			InstanceID:        getEnv("INSTANCE_ID", ""),
			ServerPublicKey:   os.Getenv("SERVER_PUBLIC_KEY"),
			ServerPrivateKey:  os.Getenv("SERVER_PRIVATE_KEY"),
			GlobalRegisterURL: os.Getenv("GLOBAL_SERVER_REGISTER_URL"),
		},
		Odoo: OdooConfig{
			URL:          os.Getenv("ODOO_URL"),
			Database:     os.Getenv("ODOO_DB"),
			Username:     os.Getenv("ODOO_USER"),
			Password:     os.Getenv("ODOO_PASSWORD"),
			SyncInterval: getIntEnv("ODOO_SYNC_INTERVAL", 15),
		},
	}, nil
}

// getEnv gets environment variable with default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// parseBootstrapNodes parses comma-separated bootstrap node URLs
func parseBootstrapNodes(s string) []string {
	if s == "" {
		return nil
	}
	nodes := strings.Split(s, ",")
	result := make([]string, 0, len(nodes))
	for _, n := range nodes {
		n = strings.TrimSpace(n)
		if n != "" {
			result = append(result, n)
		}
	}
	return result
}
