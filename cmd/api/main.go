package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dmytrosurovtsev/eckwmsgo/internal/config"
	"github.com/dmytrosurovtsev/eckwmsgo/internal/database"
	"github.com/dmytrosurovtsev/eckwmsgo/internal/handlers"
	"github.com/dmytrosurovtsev/eckwmsgo/internal/models"
	"github.com/dmytrosurovtsev/eckwmsgo/internal/services/odoo"
)

func main() {
	// 1. Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// 2. Initialize database (Detects Embedded vs External automatically)
	db, err := database.Connect(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	// Note: db.Close() is called manually in shutdown handler below

	// 3. Auto-Migrate Schema (Critical for Zero-Config)
	log.Println("🚀 Synchronizing database schema...")
	err = db.AutoMigrate(
		&models.UserAuth{},
		&models.RegisteredDevice{},
		// Odoo Core Models (Replacing old WMS models)
		&models.ProductProduct{},
		&models.StockLocation{},
		&models.StockLot{},
		&models.StockPackageType{},
		&models.StockQuantPackage{},
		&models.StockQuant{},
		&models.StockPicking{},
		&models.StockMoveLine{},

		// Legacy support (Keep for now or refactor later if needed)
		&models.Order{}, // Keeping Order for RMA logic for now

		// Sync tables
		&models.EntityChecksum{},
		&models.SyncMetadata{},
		&models.SyncConflict{},
		&models.SyncQueue{},
		&models.SyncRoute{},
		&models.EncryptedSyncPacket{}, // Zero-Knowledge Relay support
		// AI System Models
		&models.AIAgent{},
		&models.AIPermission{},
		&models.AIAuditLog{},
		&models.AIRateLimit{},
	)
	if err != nil {
		log.Printf("⚠️ Migration warning: %v\n", err)
	} else {
		log.Println("✅ Schema synchronized successfully")
	}

	// 4. Set up HTTP router
	router := handlers.NewRouter(db)

	// 5. Start Odoo Sync Service (Background)
	odooService := odoo.NewSyncService(db, odoo.Config{
		URL:          cfg.Odoo.URL,
		Database:     cfg.Odoo.Database,
		Username:     cfg.Odoo.Username,
		Password:     cfg.Odoo.Password,
		SyncInterval: cfg.Odoo.SyncInterval,
	})
	odooService.Start()

	// 6. Start server with graceful shutdown
	port := os.Getenv("PORT")
	if port == "" {
		port = "3001" // Use 3001 as default for Go version
	}

	server := &http.Server{
		Addr:    ":" + port,
		Handler: router.Handler(),
	}

	// Channel to listen for shutdown signals
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	// Start server in goroutine
	go func() {
		log.Printf("🚀 Server starting on port %s\n", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for shutdown signal
	sig := <-shutdown
	log.Printf("\n⚠️  Received signal: %v. Shutting down gracefully...\n", sig)

	// Create context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	// Stop Odoo sync service
	odooService.Stop()

	// Close database (this also stops embedded PostgreSQL)
	log.Println("🛑 Closing database connection...")
	if err := db.Close(); err != nil {
		log.Printf("Database close error: %v", err)
	}

	log.Println("✅ Shutdown complete")
}
