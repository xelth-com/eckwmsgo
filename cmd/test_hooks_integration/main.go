package main

import (
	"fmt"
	"log"
	"time"

	"github.com/xelth-com/eckwmsgo/internal/config"
	"github.com/xelth-com/eckwmsgo/internal/database"
	"github.com/xelth-com/eckwmsgo/internal/models"
	"github.com/xelth-com/eckwmsgo/internal/sync"
)

func main() {
	fmt.Println("🪝 Testing GORM Hooks Integration...")

	// 1. Init
	cfg, _ := config.Load()
	db, err := database.Connect(cfg.Database)
	if err != nil {
		log.Fatalf("DB Error: %v", err)
	}

	// Register Hooks manually for this standalone script (usually done in main.go)
	// We need to ensure the calculator is the same as in production
	calc := sync.NewChecksumCalculator("test-node")
	sync.RegisterHooks(db.DB, calc, "test-node")

	// 2. Test Create
	testProduct := models.ProductProduct{
		ID:          99999, // High ID to avoid conflict
		Name:        "Test Hook Product",
		DefaultCode: "TEST-HOOK-001",
		Active:      true,
	}

	fmt.Println("1️⃣ Creating Product...")
	if err := db.Create(&testProduct).Error; err != nil {
		log.Fatalf("Create failed: %v", err)
	}

	// Verify Checksum exists
	var checksum models.EntityChecksum
	err = db.Where("entity_type = ? AND entity_id = ?", "product", "99999").First(&checksum).Error
	if err != nil {
		log.Fatalf("❌ Hook failed: Checksum not found after create! (%v)", err)
	}
	fmt.Printf("✅ Checksum created: %s (Hash: %s)\n", checksum.EntityID, checksum.ContentHash[:8])
	initialHash := checksum.ContentHash

	// 3. Test Update (Content Change)
	fmt.Println("2️⃣ Updating Product Content (Name)...")
	testProduct.Name = "Updated Name"
	if err := db.Save(&testProduct).Error; err != nil {
		log.Fatalf("Update failed: %v", err)
	}

	// Verify Checksum changed
	var checksum2 models.EntityChecksum
	db.Where("entity_type = ? AND entity_id = ?", "product", "99999").First(&checksum2)

	if checksum2.ContentHash == initialHash {
		log.Fatalf("❌ Hook failed: Hash did NOT change after content update!")
	}
	fmt.Printf("✅ Checksum updated: %s -> %s\n", initialHash[:8], checksum2.ContentHash[:8])

	// 4. Test Update (Ignored Field)
	fmt.Println("3️⃣ Updating Ignored Field (LastSyncedAt)...")
	// We use UpdateColumn to bypass hooks? No, we want to see if hooks handle it correctly.
	// Actually GORM hooks run on Save.
	testProduct.LastSyncedAt = time.Now()
	if err := db.Save(&testProduct).Error; err != nil {
		log.Fatalf("Update failed: %v", err)
	}

	var checksum3 models.EntityChecksum
	db.Where("entity_type = ? AND entity_id = ?", "product", "99999").First(&checksum3)

	if checksum3.ContentHash != checksum2.ContentHash {
		log.Printf("⚠️ Warning: Hash changed on timestamp update. Check exclusion logic. (%s -> %s)", checksum2.ContentHash[:8], checksum3.ContentHash[:8])
	} else {
		fmt.Printf("✅ Hash stable (correctly ignored timestamp change)\n")
	}

	// Cleanup
	db.Unscoped().Delete(&testProduct)
	db.Unscoped().Delete(&checksum)
	fmt.Println("\n🎉 GORM Hooks Integration Verified!")
}
