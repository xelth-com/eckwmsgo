package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config holds all application configuration
type Config struct {
	NodeEnv     string
	Port        string
	JWTSecret   string
	EncKey      string
	Database    DatabaseConfig
	Translation TranslationConfig
	Server      ServerConfig
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

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	// Support DATABASE_URL format
	databaseURL := os.Getenv("DATABASE_URL")

	return &Config{
		NodeEnv:   getEnv("NODE_ENV", "development"),
		Port:      getEnv("PORT", "3001"),
		JWTSecret: jwtSecret,
		EncKey:    os.Getenv("ENC_KEY"),
		Database: func() DatabaseConfig {
			if databaseURL != "" {
				// Parse DATABASE_URL (format: postgresql://user:pass@host:port/dbname?sslmode=xxx)
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
			LocalPort:         getEnv("LOCAL_SERVER_PORT", "3000"),
			GlobalPort:        getEnv("GLOBAL_SERVER_PORT", "8080"),
			GlobalURL:         os.Getenv("GLOBAL_SERVER_URL"),
			LocalInternalURL:  os.Getenv("LOCAL_SERVER_INTERNAL_URL"),
			GlobalAPIEndpoint: os.Getenv("GLOBAL_SERVER_API_ENDPOINT"),
			GlobalAPIKey:      os.Getenv("GLOBAL_SERVER_API_KEY"),
			InstanceID:        os.Getenv("INSTANCE_ID"),
			ServerPublicKey:   os.Getenv("SERVER_PUBLIC_KEY"),
			ServerPrivateKey:  os.Getenv("SERVER_PRIVATE_KEY"),
			GlobalRegisterURL: os.Getenv("GLOBAL_SERVER_REGISTER_URL"),
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
