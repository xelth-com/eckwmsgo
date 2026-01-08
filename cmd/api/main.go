package main

import (
	"log"
	"net/http"
	"os"

	"github.com/dmytrosurovtsev/eckwmsgo/internal/config"
	"github.com/dmytrosurovtsev/eckwmsgo/internal/database"
	"github.com/dmytrosurovtsev/eckwmsgo/internal/handlers"
	"github.com/dmytrosurovtsev/eckwmsgo/internal/models"
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
	defer db.Close()

	// 3. Auto-Migrate Schema (Critical for Zero-Config)
	log.Println("🚀 Synchronizing database schema...")
	err = db.AutoMigrate(
		&models.UserAuth{},
		&models.Warehouse{},
		&models.WarehouseRack{},
		&models.Place{},
		&models.Item{},
		&models.Box{},
		&models.RmaRequest{},
		&models.RepairOrder{},
		&models.ProductAlias{},
	)
	if err != nil {
		log.Printf("⚠️ Migration warning: %v\n", err)
	} else {
		log.Println("✅ Schema synchronized successfully")
	}

	// 4. Set up HTTP router
	router := handlers.NewRouter(db)

	// 5. Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "3001" // Use 3001 as default for Go version
	}

	log.Printf("🚀 Server starting on port %s\n", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
