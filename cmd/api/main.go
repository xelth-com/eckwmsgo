package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/xelth-com/eckwmsgo/internal/ai"
	"github.com/xelth-com/eckwmsgo/internal/config"
	"github.com/xelth-com/eckwmsgo/internal/database"
	"github.com/xelth-com/eckwmsgo/internal/delivery"
	"github.com/xelth-com/eckwmsgo/internal/delivery/dhl"
	"github.com/xelth-com/eckwmsgo/internal/delivery/opal"
	"github.com/xelth-com/eckwmsgo/internal/handlers"
	"github.com/xelth-com/eckwmsgo/internal/mesh"
	"github.com/xelth-com/eckwmsgo/internal/models"
	deliveryService "github.com/xelth-com/eckwmsgo/internal/services/delivery"
	"github.com/xelth-com/eckwmsgo/internal/services/odoo"
	"github.com/xelth-com/eckwmsgo/internal/sync"
	"github.com/xelth-com/eckwmsgo/internal/utils"
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
		&models.ResPartner{}, // Customer/Supplier addresses

		// Delivery Models (OPAL integration)
		&models.DeliveryCarrier{},
		&models.StockPickingDelivery{},
		&models.DeliveryTracking{},

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

	// Start Mesh Discovery
	mesh.StartDiscovery(cfg)

	// Report to global server for discovery (non-blocking)
	go utils.ReportToGlobalServer()

	// --- MESH SYNC ENGINE INIT ---
	log.Println("🔄 Initializing Mesh Sync Engine...")
	syncCfg := config.LoadSyncConfig()
	syncCfg.Role = string(cfg.NodeRole) // Use node role from main config

	syncEngine := sync.NewSyncEngine(db, syncCfg, &sync.MeshConfig{
		InstanceID: cfg.InstanceID,
		MeshSecret: cfg.MeshSecret,
		BaseURL:    cfg.BaseURL,
		NodeRole:   string(cfg.NodeRole),
	})

	if syncCfg.Enabled {
		if err := syncEngine.Start(); err != nil {
			log.Printf("⚠️ Sync Engine: Failed to start: %v", err)
		} else {
			log.Println("✅ Sync Engine: Started successfully")
		}
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

	// Register Odoo service with router for API endpoints
	router.SetOdooService(odooService)

	// Register Sync Engine with router for mesh sync endpoints
	router.SetSyncEngine(syncEngine)

	// --- DELIVERY SYSTEM INIT ---
	log.Println("📦 Initializing Delivery System...")

	// Get absolute path to scripts directory
	// Try to find scripts relative to executable first, then fall back to working directory
	opalScriptPath := "./scripts/delivery/create-opal-order.js"
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		candidatePath := filepath.Join(exeDir, "scripts", "delivery", "create-opal-order.js")
		if _, statErr := os.Stat(candidatePath); statErr == nil {
			opalScriptPath = candidatePath
			log.Printf("📦 Using script path relative to executable: %s", opalScriptPath)
		}
	}
	// Also check if .env specifies a custom path
	if envPath := os.Getenv("OPAL_SCRIPT_PATH"); envPath != "" {
		opalScriptPath = envPath
		log.Printf("📦 Using script path from OPAL_SCRIPT_PATH: %s", opalScriptPath)
	}

	// Create OPAL provider
	opalProvider, err := opal.NewProvider(opal.Config{
		ScriptPath: opalScriptPath,
		NodePath:   "node",
		Username:   os.Getenv("OPAL_USERNAME"),
		Password:   os.Getenv("OPAL_PASSWORD"),
		URL:        os.Getenv("OPAL_URL"),
		Headless:   true,
		Timeout:    300,
	})
	if err != nil {
		log.Printf("⚠️ Delivery: Failed to init OPAL provider: %v", err)
	} else {
		if err := delivery.GetGlobalRegistry().Register(opalProvider); err != nil {
			log.Printf("⚠️ Delivery: Failed to register OPAL: %v", err)
		} else {
			log.Println("✅ Delivery: OPAL provider registered")
		}
	}

	// Create DHL provider
	dhlScriptPath := filepath.Join(filepath.Dir(opalScriptPath), "fetch-dhl-orders.js")
	dhlProvider, err := dhl.NewProvider(dhl.Config{
		ScriptPath: dhlScriptPath,
		NodePath:   "node",
		Username:   os.Getenv("DHL_USERNAME"),
		Password:   os.Getenv("DHL_PASSWORD"),
		URL:        os.Getenv("DHL_URL"),
		Headless:   true,
		Timeout:    300,
	})
	if err != nil {
		log.Printf("⚠️ Delivery: Failed to init DHL provider: %v", err)
	} else {
		if err := delivery.GetGlobalRegistry().Register(dhlProvider); err != nil {
			log.Printf("⚠️ Delivery: Failed to register DHL: %v", err)
		} else {
			log.Println("✅ Delivery: DHL provider registered")
		}
	}

	// Create and register delivery service
	delSvc := deliveryService.NewService(db, cfg)
	router.SetDeliveryService(delSvc)

	// Start background worker for processing shipments
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		for range ticker.C {
			if err := delSvc.ProcessPendingShipments(context.Background()); err != nil {
				log.Printf("Delivery Worker Error: %v", err)
			}
		}
	}()
	log.Println("✅ Delivery: Background worker started")

	// --- AI INITIALIZATION ---
	var aiClient *ai.GeminiClient
	if cfg.AI.GeminiKey != "" {
		log.Printf("🧠 Initializing Gemini AI (Primary: %s, Fallback: %s)...", cfg.AI.Model, cfg.AI.ModelFallback)
		c, err := ai.NewGeminiClient(context.Background(), cfg.AI.GeminiKey, cfg.AI.Model, cfg.AI.ModelFallback)
		if err != nil {
			log.Printf("⚠️ Failed to init AI: %v", err)
		} else {
			aiClient = c
			defer aiClient.Close()
			router.SetAIClient(aiClient)
			log.Println("✅ AI Client initialized")
		}
	} else {
		log.Println("⚠️ GEMINI_API_KEY not found. AI features will be disabled.")
	}

	// Start OPAL import scheduler (every hour)
	go func() {
		// Wait for system startup
		time.Sleep(1 * time.Minute)

		// Run initial import
		log.Println("⏰ Running initial OPAL import...")
		if err := delSvc.ImportOpalOrders(context.Background()); err != nil {
			log.Printf("❌ OPAL Import (initial) failed: %v", err)
		} else {
			log.Println("✅ OPAL Import (initial) completed")
		}

		// Schedule regular imports
		opalTicker := time.NewTicker(1 * time.Hour)
		for range opalTicker.C {
			log.Println("⏰ Starting scheduled OPAL import...")
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			if err := delSvc.ImportOpalOrders(ctx); err != nil {
				log.Printf("❌ OPAL Import failed: %v", err)
			} else {
				log.Println("✅ OPAL Import completed")
			}
			cancel()
		}
	}()
	log.Println("✅ Delivery: OPAL import scheduler started (hourly)")

	// 6. Start server with graceful shutdown
	port := os.Getenv("PORT")
	if port == "" {
		port = "3210" // Standard eckWMS port
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
		log.Printf("🚀 Server (%s) starting on port %s [Prefix: '%s']\n", cfg.NodeRole, port, cfg.PathPrefix)
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

	// Stop Mesh sync engine
	if syncEngine != nil {
		syncEngine.Stop()
	}

	// Close database (this also stops embedded PostgreSQL)
	log.Println("🛑 Closing database connection...")
	if err := db.Close(); err != nil {
		log.Printf("Database close error: %v", err)
	}

	log.Println("✅ Shutdown complete")
}
