package database

import (
	"fmt"
	"log"
	"time"

	"github.com/dmytrosurovtsev/eckwmsgo/internal/config"
	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB wraps gorm.DB and includes a reference to an embedded process if active
type DB struct {
	*gorm.DB
	embedded *embeddedpostgres.EmbeddedPostgres
}

// Connect establishes a connection to a PostgreSQL database (external or embedded)
func Connect(cfg config.DatabaseConfig) (*DB, error) {
	var embedded *embeddedpostgres.EmbeddedPostgres

	// Logic for Embedded Mode: Localhost and No Password
	isEmbedded := cfg.Host == "localhost" && cfg.Password == ""

	var embeddedPassword string
	if isEmbedded {
		log.Println("📦 Mode: [Embedded PostgreSQL] - Initializing internal database...")

		// Setup embedded configuration
		embeddedCfg := embeddedpostgres.DefaultConfig().
			DataPath("./db_data"). // Persistent data folder in the app directory
			Port(5433).            // Use custom port for embedded mode
			Database(cfg.Database).
			Username(cfg.Username).
			Password("postgres") // Set password for embedded user

		embedded = embeddedpostgres.NewDatabase(embeddedCfg)

		if err := embedded.Start(); err != nil {
			return nil, fmt.Errorf("failed to start embedded database: %w", err)
		}

		// Update connection parameters to point to the embedded instance
		cfg.Port = "5433"
		embeddedPassword = "postgres"
		log.Println("✅ Embedded PostgreSQL process started on port 5433")
	} else {
		log.Printf("🌐 Mode: [External PostgreSQL] - Connecting to %s:%s\n", cfg.Host, cfg.Port)
		embeddedPassword = cfg.Password
	}

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host,
		cfg.Port,
		cfg.Username,
		embeddedPassword,
		cfg.Database,
	)

	// Configure GORM
	logLevel := logger.Info
	if cfg.Alter {
		logLevel = logger.Silent
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		// Clean up embedded process if GORM connection fails
		if embedded != nil {
			_ = embedded.Stop()
		}
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err == nil {
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetMaxOpenConns(100)
		sqlDB.SetConnMaxLifetime(time.Hour)
	}

	log.Println("✅ Database connection established")

	return &DB{
		DB:       db,
		embedded: embedded,
	}, nil
}

// Close ensures the database connection and embedded process are shut down
func (db *DB) Close() error {
	if db.embedded != nil {
		log.Println("🛑 Stopping Embedded PostgreSQL process...")
		_ = db.embedded.Stop()
	}

	sqlDB, err := db.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// AutoMigrate triggers GORM schema synchronization
func (db *DB) AutoMigrate(models ...interface{}) error {
	return db.DB.AutoMigrate(models...)
}
