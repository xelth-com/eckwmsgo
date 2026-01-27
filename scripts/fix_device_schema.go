package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Load .env
	godotenv.Load()

	// Build connection string
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("PG_HOST"),
		os.Getenv("PG_PORT"),
		os.Getenv("PG_USERNAME"),
		os.Getenv("PG_PASSWORD"),
		os.Getenv("PG_DATABASE"),
	)

	// Connect to database
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Check if there are any records
	var count int64
	db.Table("registered_devices").Count(&count)
	log.Printf("📊 Total records (including deleted): %d", count)

	if count > 0 {
		// Show data from snake_case columns
		var snakeRecords []map[string]interface{}
		db.Table("registered_devices").
			Select("device_id, device_name, status").
			Find(&snakeRecords)

		log.Println("\n📝 Data from snake_case columns:")
		for _, r := range snakeRecords {
			log.Printf("  - device_id: %v, device_name: %v, status: %v",
				r["device_id"], r["device_name"], r["status"])
		}

		// Show data from camelCase columns
		var camelRecords []map[string]interface{}
		db.Table("registered_devices").
			Select("deviceId, deviceName, status").
			Find(&camelRecords)

		log.Println("\n📝 Data from camelCase columns:")
		for _, r := range camelRecords {
			log.Printf("  - deviceId: %v, deviceName: %v, status: %v",
				r["deviceId"], r["deviceName"], r["status"])
		}
	}

	// Drop old snake_case columns and migrate data
	log.Println("\n🔧 Attempting to fix schema...")

	// Migrate data from snake_case to camelCase if needed
	migrateResult := db.Exec(`
		UPDATE registered_devices 
		SET deviceId = device_id, 
		    deviceName = device_name 
		WHERE (deviceId IS NULL OR deviceId = '') AND (device_id IS NOT NULL AND device_id != '')
	`)
	log.Printf("  - Data migration: %d rows affected, error: %v", migrateResult.RowsAffected, migrateResult.Error)

	// Drop old snake_case columns
	dropResult := db.Exec(`
		ALTER TABLE registered_devices 
		DROP COLUMN IF EXISTS device_id,
		DROP COLUMN IF EXISTS device_name
	`)
	log.Printf("  - Drop old columns: error: %v", dropResult.Error)

	if dropResult.Error == nil {
		log.Println("✅ Schema fixed successfully!")
	} else {
		log.Println("❌ Error dropping columns:", dropResult.Error)
	}
}
