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

	// Delete devices with empty name or "unknown" name
	deleteResult := db.Table("registered_devices").
		Where("device_name = '' OR device_name = 'unknown' OR device_name IS NULL").
		Delete(nil)

	if deleteResult.Error != nil {
		log.Fatal("Failed to delete devices:", deleteResult.Error)
	}

	if deleteResult.RowsAffected > 0 {
		log.Printf("✅ Deleted %d device(s) with empty or 'unknown' name", deleteResult.RowsAffected)
	} else {
		log.Println("ℹ️  No devices with empty or 'unknown' name found")
	}

	// Check all remaining devices
	var count int64
	db.Table("registered_devices").Where("deleted_at IS NULL").Count(&count)
	log.Printf("📊 Total active devices: %d", count)
}
