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

	// List all devices
	var devices []struct {
		DeviceID  string
		Name      string
		Status    string
		CreatedAt string
		DeletedAt *string
	}

	result := db.Table("registered_devices").
		Select("device_id, device_name, status, created_at, deleted_at").
		Order("created_at DESC").
		Find(&devices)

	if result.Error != nil {
		log.Fatal("Failed to query devices:", result.Error)
	}

	if len(devices) == 0 {
		log.Println("ℹ️  No devices found")
		return
	}

	log.Printf("Found %d device(s):", len(devices))
	for _, d := range devices {
		status := d.Status
		if d.DeletedAt != nil {
			status = fmt.Sprintf("%s (deleted)", status)
		}
		log.Printf("  - ID: %s, Name: %s, Status: %s, Created: %s", d.DeviceID, d.Name, status, d.CreatedAt)
	}
}
