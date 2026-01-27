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

	log.Println("📋 Registered devices table schema:")
	for _, col := range columns {
		log.Printf("  - %s: %s (nullable: %s, default: %s)", col.ColumnName, col.DataType, col.IsNullable, col.ColumnDefault)
	}

	// Count all devices
	var total int64
	db.Table("registered_devices").Where("deleted_at IS NULL").Count(&total)
	log.Printf("\n📊 Total active devices: %d", total)

	// List all devices
	var devices []struct {
		DeviceID   string
		DeviceName string
		Status     string
		CreatedAt  string
		UpdatedAt  string
		DeletedAt  *string
	}

	db.Table("registered_devices").
		Select("device_id, device_name, status, created_at, updated_at, deleted_at").
		Find(&devices)

	if len(devices) == 0 {
		log.Println("\nℹ️  No devices found")
	} else {
		log.Println("\n📱 Devices:")
		for _, d := range devices {
			status := d.Status
			if d.DeletedAt != nil {
				status = fmt.Sprintf("%s (deleted)", status)
			}
			log.Printf("  - ID: %s, Name: %s, Status: %s, Created: %s", d.DeviceID, d.DeviceName, status, d.CreatedAt)
		}
	}
}
