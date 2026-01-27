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

	// Find devices with name "unknown"
	var devices []struct {
		DeviceID  string
		Name      string
		Status    string
		CreatedAt string
	}

	result := db.Table("registered_devices").
		Select("device_id, device_name, status, created_at").
		Where("device_name = ?", "unknown").
		Find(&devices)

	if result.Error != nil {
		log.Fatal("Failed to query devices:", result.Error)
	}

	if len(devices) == 0 {
		log.Println("ℹ️  No devices with name 'unknown' found")
		return
	}

	log.Printf("Found %d device(s) with name 'unknown':", len(devices))
	for _, d := range devices {
		log.Printf("  - ID: %s, Status: %s, Created: %s", d.DeviceID, d.Status, d.CreatedAt)
	}

	// Delete all devices with name "unknown" (soft delete)
	deleteResult := db.Table("registered_devices").
		Where("device_name = ?", "unknown").
		Delete(nil)

	if deleteResult.Error != nil {
		log.Fatal("Failed to delete devices:", deleteResult.Error)
	}

	log.Printf("✅ Deleted %d device(s) with name 'unknown'", deleteResult.RowsAffected)
}
