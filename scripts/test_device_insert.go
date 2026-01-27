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

	// Get table schema
	var columns []struct {
		ColumnName    string
		DataType      string
		IsNullable    string
		ColumnDefault string
	}

	db.Raw(`
		SELECT column_name, data_type, is_nullable, column_default
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'registered_devices'
		ORDER BY ordinal_position
	`).Scan(&columns)

	log.Println("📋 Current registered_devices table schema:")
	for _, col := range columns {
		log.Printf("  - %s: %s", col.ColumnName, col.DataType)
	}

	// Try to insert a test device
	log.Println("\n🧪 Testing insert with camelCase columns...")
	testDevice := struct {
		DeviceID   string
		DeviceName string
		PublicKey  string
		Status     string
		LastSeenAt string
	}{
		DeviceID:   "test_device_id",
		DeviceName: "Test Device",
		PublicKey:  "test_public_key",
		Status:     "pending",
		LastSeenAt: "2026-01-27T00:00:00Z",
	}

	insertResult := db.Exec(`
		INSERT INTO registered_devices (deviceId, deviceName, public_key, status, last_seen_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, NOW(), NOW())
	`, testDevice.DeviceID, testDevice.DeviceName, testDevice.PublicKey, testDevice.Status, testDevice.LastSeenAt)

	if insertResult.Error != nil {
		log.Printf("❌ Insert failed: %v", insertResult.Error)
	} else {
		log.Printf("✅ Insert successful! Rows affected: %d", insertResult.RowsAffected)

		// Clean up
		db.Exec("DELETE FROM registered_devices WHERE deviceId = ?", testDevice.DeviceID)
		log.Println("🧹 Test record deleted")
	}
}
