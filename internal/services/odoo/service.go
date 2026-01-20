package odoo

import (
	"log"
	"time"

	"github.com/xelth-com/eckwmsgo/internal/database"
	"github.com/xelth-com/eckwmsgo/internal/models"
	"gorm.io/gorm/clause"
)

// SyncService orchestrates synchronization between Odoo and local DB
type SyncService struct {
	client *Client
	db     *database.DB
	cfg    Config
	stop   chan struct{}
}

// Config holds Odoo connection settings
type Config struct {
	URL          string
	Database     string
	Username     string
	Password     string
	SyncInterval int // in minutes
}

// NewSyncService creates a new synchronization service
func NewSyncService(db *database.DB, cfg Config) *SyncService {
	return &SyncService{
		client: NewClient(cfg.URL, cfg.Database, cfg.Username, cfg.Password),
		db:     db,
		cfg:    cfg,
		stop:   make(chan struct{}),
	}
}

// Start begins the background synchronization loop
func (s *SyncService) Start() {
	if s.cfg.URL == "" {
		log.Println("Odoo Sync disabled: ODOO_URL not configured")
		return
	}

	go func() {
		log.Println("📡 Odoo Sync Service started")

		// Authenticate first
		if _, err := s.client.Authenticate(); err != nil {
			log.Printf("❌ Odoo authentication failed: %v", err)
			return
		}

		// Initial sync delay
		time.Sleep(5 * time.Second)
		s.runFullSync()

		interval := time.Duration(s.cfg.SyncInterval) * time.Minute
		if s.cfg.SyncInterval <= 0 {
			interval = 15 * time.Minute
		}

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.runFullSync()
			case <-s.stop:
				log.Println("🛑 Odoo Sync Service stopped")
				return
			}
		}
	}()
}

// Stop halts the service
func (s *SyncService) Stop() {
	close(s.stop)
}

// TriggerManualSync triggers a manual sync immediately
func (s *SyncService) TriggerManualSync() {
	log.Println("🔔 Manual sync triggered")
	s.runFullSync()
}

// runFullSync runs all sync operations
func (s *SyncService) runFullSync() {
	log.Println("🔄 Odoo: Starting full sync...")

	// Order matters: locations first (for hierarchy), then products, then partners, then quants/lots/packages
	s.syncLocations()
	s.syncProducts()
	s.syncPartners() // Sync customer/supplier addresses
	s.syncLots()
	s.syncPackages()
	s.syncQuants()
	s.syncPickings()
	s.syncMoveLines()

	log.Println("✅ Odoo: Full sync completed")
}

// syncProducts pulls product data from Odoo directly into 'product_product' table
func (s *SyncService) syncProducts() {
	log.Println("📦 Odoo: Syncing Products...")

	// 1. Get last write_date from local DB
	var lastProduct models.ProductProduct
	var lastWriteDate string = "2000-01-01 00:00:00"

	result := s.db.Order("write_date DESC").First(&lastProduct)
	if result.Error == nil && !lastProduct.WriteDate.IsZero() {
		lastWriteDate = lastProduct.WriteDate.Format("2006-01-02 15:04:05")
	}

	// 2. Prepare Domain
	domain := []interface{}{
		[]interface{}{"write_date", ">", lastWriteDate},
	}

	// 3. Fetch from Odoo
	var products []models.ProductProduct
	err := s.client.SearchRead("product.product", domain, []string{
		"default_code", "barcode", "name", "type", "list_price", "standard_price", "weight", "volume", "write_date", "active",
	}, 1000, 0, &products)

	if err != nil {
		log.Printf("❌ Odoo Sync Error (Products): %v", err)
		return
	}

	if len(products) == 0 {
		return
	}

	// 4. Save to Local DB
	count := 0
	for _, p := range products {
		p.LastSyncedAt = time.Now()

		// Upsert logic based on ID (Primary Key is Odoo ID)
		if err := s.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			UpdateAll: true,
		}).Create(&p).Error; err != nil {
			log.Printf("Failed to save product %d: %v", p.ID, err)
		} else {
			count++
		}
	}

	log.Printf("✅ Odoo: Updated %d products", count)
}

// syncPartners pulls partner (customer/supplier) data from Odoo
func (s *SyncService) syncPartners() {
	log.Println("👥 Odoo: Syncing Partners (Customers/Suppliers)...")

	// Fetch only companies and contacts (not addresses which are child records)
	domain := []interface{}{
		[]interface{}{"active", "=", true},
		// Optionally filter by type: "contact", "invoice", "delivery", "other"
		// For delivery, we mainly need "contact" and "delivery" types
	}

	var partners []models.ResPartner
	err := s.client.SearchRead("res.partner", domain, []string{
		"name", "street", "street2", "zip", "city", "state_id", "country_id",
		"phone", "email", "vat", "company_type", "is_company",
	}, 1000, 0, &partners)

	if err != nil {
		log.Printf("❌ Odoo Sync Error (Partners): %v", err)
		return
	}

	if len(partners) == 0 {
		return
	}

	// Save to local DB
	count := 0
	for _, partner := range partners {
		// Upsert based on Odoo ID
		if err := s.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			UpdateAll: true,
		}).Create(&partner).Error; err != nil {
			log.Printf("Failed to save partner %d: %v", partner.ID, err)
		} else {
			count++
		}
	}

	log.Printf("✅ Odoo: Updated %d partners", count)
}

// syncLocations pulls location data from Odoo directly into 'stock_location' table
func (s *SyncService) syncLocations() {
	log.Println("📍 Odoo: Syncing Locations...")

	// Simple approach: fetch all active locations (or use write_date if available)
	domain := []interface{}{
		[]interface{}{"active", "=", true},
	}

	var locations []models.StockLocation
	err := s.client.SearchRead("stock.location", domain, []string{
		"name", "complete_name", "barcode", "usage", "location_id", "active",
	}, 1000, 0, &locations)

	if err != nil {
		log.Printf("❌ Odoo Sync Error (Locations): %v", err)
		return
	}

	if len(locations) == 0 {
		return
	}

	count := 0
	for _, l := range locations {
		l.LastSyncedAt = time.Now()

		if err := s.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			UpdateAll: true,
		}).Create(&l).Error; err != nil {
			log.Printf("Failed to save location %d: %v", l.ID, err)
		} else {
			count++
		}
	}

	log.Printf("✅ Odoo: Updated %d locations", count)
}

// syncLots pulls lot data from Odoo directly into 'stock_lot' table
func (s *SyncService) syncLots() {
	log.Println("🏷️ Odoo: Syncing Lots...")

	domain := []interface{}{}

	var lots []models.StockLot
	err := s.client.SearchRead("stock.lot", domain, []string{
		"name", "product_id", "ref", "create_date",
	}, 1000, 0, &lots)

	if err != nil {
		log.Printf("❌ Odoo Sync Error (Lots): %v", err)
		return
	}

	if len(lots) == 0 {
		return
	}

	count := 0
	for _, lot := range lots {
		if err := s.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			UpdateAll: true,
		}).Create(&lot).Error; err != nil {
			log.Printf("Failed to save lot %d: %v", lot.ID, err)
		} else {
			count++
		}
	}

	log.Printf("✅ Odoo: Updated %d lots", count)
}

// syncPackages pulls package data from Odoo directly into 'stock_quant_package' table
func (s *SyncService) syncPackages() {
	log.Println("📦 Odoo: Syncing Packages...")

	domain := []interface{}{}

	var packages []models.StockQuantPackage
	err := s.client.SearchRead("stock.quant.package", domain, []string{
		"name", "pack_date", "location_id",
	}, 1000, 0, &packages)

	if err != nil {
		log.Printf("❌ Odoo Sync Error (Packages): %v", err)
		return
	}

	if len(packages) == 0 {
		return
	}

	count := 0
	for _, pkg := range packages {
		if err := s.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			UpdateAll: true,
		}).Create(&pkg).Error; err != nil {
			log.Printf("Failed to save package %d: %v", pkg.ID, err)
		} else {
			count++
		}
	}

	log.Printf("✅ Odoo: Updated %d packages", count)
}

// syncQuants pulls quant data from Odoo directly into 'stock_quant' table
func (s *SyncService) syncQuants() {
	log.Println("📊 Odoo: Syncing Quants...")

	domain := []interface{}{
		[]interface{}{"quantity", ">", 0},
	}

	var quants []models.StockQuant
	err := s.client.SearchRead("stock.quant", domain, []string{
		"product_id", "location_id", "lot_id", "package_id", "quantity", "reserved_quantity", "inventory_date",
	}, 1000, 0, &quants)

	if err != nil {
		log.Printf("❌ Odoo Sync Error (Quants): %v", err)
		return
	}

	if len(quants) == 0 {
		return
	}

	count := 0
	for _, q := range quants {
		if err := s.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			UpdateAll: true,
		}).Create(&q).Error; err != nil {
			log.Printf("Failed to save quant %d: %v", q.ID, err)
		} else {
			count++
		}
	}

	log.Printf("✅ Odoo: Updated %d quants", count)
}

// syncPickings pulls picking (transfer order) data from Odoo
func (s *SyncService) syncPickings() {
	log.Println("📋 Odoo: Syncing Pickings (Transfer Orders)...")

	var lastPicking models.StockPicking
	var lastWriteDate string = "2000-01-01 00:00:00"

	result := s.db.Order("scheduled_date DESC").First(&lastPicking)
	if result.Error == nil && !lastPicking.ScheduledDate.IsZero() {
		lastWriteDate = lastPicking.ScheduledDate.Format("2006-01-02 15:04:05")
	}

	domain := []interface{}{
		"&",
		[]interface{}{"scheduled_date", ">", lastWriteDate},
		[]interface{}{"state", "in", []string{"draft", "waiting", "confirmed", "assigned", "done"}},
	}

	var pickings []models.StockPicking
	err := s.client.SearchRead("stock.picking", domain, []string{
		"name", "state", "location_id", "location_dest_id", "scheduled_date",
		"origin", "priority", "picking_type_id", "partner_id", "date_done",
	}, 1000, 0, &pickings)

	if err != nil {
		log.Printf("❌ Odoo Sync Error (Pickings): %v", err)
		return
	}

	if len(pickings) == 0 {
		return
	}

	count := 0
	for _, p := range pickings {
		if err := s.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			UpdateAll: true,
		}).Create(&p).Error; err != nil {
			log.Printf("Failed to save picking %d: %v", p.ID, err)
		} else {
			count++
		}
	}

	log.Printf("✅ Odoo: Updated %d pickings", count)
}

// syncMoveLines pulls move line data from Odoo
func (s *SyncService) syncMoveLines() {
	log.Println("📝 Odoo: Syncing Move Lines...")

	domain := []interface{}{}

	var moveLines []models.StockMoveLine
	err := s.client.SearchRead("stock.move.line", domain, []string{
		"picking_id", "product_id", "quantity", "location_id", "location_dest_id", // Odoo 19: qty_done -> quantity
		"package_id", "result_package_id", "lot_id", "state",
	}, 1000, 0, &moveLines)

	if err != nil {
		log.Printf("❌ Odoo Sync Error (Move Lines): %v", err)
		return
	}

	if len(moveLines) == 0 {
		return
	}

	count := 0
	for _, ml := range moveLines {
		if err := s.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			UpdateAll: true,
		}).Create(&ml).Error; err != nil {
			log.Printf("Failed to save move line %d: %v", ml.ID, err)
		} else {
			count++
		}
	}

	log.Printf("✅ Odoo: Updated %d move lines", count)
}
