package database

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/xelth-com/eckwmsgo/internal/config"
	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	embeddedDataPath = "./db_data"
	embeddedPort     = 5433
)

// DB wraps gorm.DB and includes a reference to an embedded process if active
type DB struct {
	*gorm.DB
	embedded *embeddedpostgres.EmbeddedPostgres
}

// cleanupStaleEmbeddedPostgres cleans up leftover processes from a previous crash
func cleanupStaleEmbeddedPostgres() {
	pidFile := filepath.Join(embeddedDataPath, "postmaster.pid")

	// Check if postmaster.pid exists
	data, err := os.ReadFile(pidFile)
	if err != nil {
		// No pid file = clean state
		return
	}

	// Parse PID from first line of postmaster.pid
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	if !scanner.Scan() {
		return
	}
	pid, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
	if err != nil {
		log.Printf("⚠️  Could not parse PID from postmaster.pid: %v", err)
		return
	}

	// Check if process is still running
	process, err := os.FindProcess(pid)
	if err != nil {
		// Process doesn't exist, clean up pid file
		log.Printf("🧹 Cleaning up stale postmaster.pid (PID %d not found)", pid)
		os.Remove(pidFile)
		return
	}

	// On Unix, FindProcess always succeeds, so we need to send signal 0 to check
	err = process.Signal(syscall.Signal(0))
	if err != nil {
		// Process is not running, clean up pid file
		log.Printf("🧹 Cleaning up stale postmaster.pid (PID %d not running)", pid)
		os.Remove(pidFile)
		return
	}

	// Process is running - try to stop it gracefully
	log.Printf("⚠️  Found orphaned PostgreSQL process (PID %d), attempting to stop...", pid)

	// Send SIGTERM for graceful shutdown
	if err := process.Signal(syscall.SIGTERM); err != nil {
		log.Printf("⚠️  Could not send SIGTERM to PID %d: %v", pid, err)
	}

	// Wait up to 5 seconds for process to stop
	for i := 0; i < 10; i++ {
		time.Sleep(500 * time.Millisecond)
		if err := process.Signal(syscall.Signal(0)); err != nil {
			log.Printf("✅ Orphaned PostgreSQL process stopped")
			os.Remove(pidFile)
			return
		}
	}

	// If still running, force kill
	log.Printf("⚠️  Process did not stop gracefully, sending SIGKILL...")
	process.Kill()
	time.Sleep(500 * time.Millisecond)
	os.Remove(pidFile)
}

// isPortInUse checks if a port is already in use
func isPortInUse(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// Connect establishes a connection to a PostgreSQL database (external or embedded)
func Connect(cfg config.DatabaseConfig) (*DB, error) {
	var embedded *embeddedpostgres.EmbeddedPostgres

	// Logic for Embedded Mode: Localhost and No Password
	isEmbedded := cfg.Host == "localhost" && cfg.Password == ""

	var embeddedPassword string
	if isEmbedded {
		log.Println("📦 Mode: [Embedded PostgreSQL] - Initializing internal database...")

		// Cleanup any stale processes from previous crash
		cleanupStaleEmbeddedPostgres()

		// Additional check: if port is still in use after cleanup, wait a bit
		if isPortInUse(embeddedPort) {
			log.Printf("⚠️  Port %d still in use, waiting for release...", embeddedPort)
			for i := 0; i < 6; i++ {
				time.Sleep(500 * time.Millisecond)
				if !isPortInUse(embeddedPort) {
					break
				}
			}
			if isPortInUse(embeddedPort) {
				return nil, fmt.Errorf("port %d is still in use by another process", embeddedPort)
			}
		}

		// Setup embedded configuration
		embeddedCfg := embeddedpostgres.DefaultConfig().
			DataPath(embeddedDataPath).
			Port(uint32(embeddedPort)).
			Database(cfg.Database).
			Username(cfg.Username).
			Password("postgres") // Set password for embedded user

		embedded = embeddedpostgres.NewDatabase(embeddedCfg)

		if err := embedded.Start(); err != nil {
			return nil, fmt.Errorf("failed to start embedded database: %w", err)
		}

		// Update connection parameters to point to the embedded instance
		cfg.Port = strconv.Itoa(embeddedPort)
		embeddedPassword = "postgres"
		log.Printf("✅ Embedded PostgreSQL process started on port %d", embeddedPort)
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
