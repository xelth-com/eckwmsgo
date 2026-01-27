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

	// Check user_auths for 'unknown'
	var userCount int64
	db.Table("user_auths").Where("username = 'unknown' OR email = 'unknown'").Count(&userCount)
	if userCount > 0 {
		log.Printf("👤 Found %d user(s) with 'unknown' in username/email", userCount)

		// Delete them
		result := db.Table("user_auths").Where("username = 'unknown' OR email = 'unknown'").Delete(nil)
		log.Printf("✅ Deleted %d user(s)", result.RowsAffected)
	} else {
		log.Println("ℹ️  No users with 'unknown' found")
	}

	// Check other common tables for 'unknown'
	tables := []string{"registered_devices", "user_auths", "items", "locations", "shipments", "boxes"}

	for _, table := range tables {
		// Try common text columns
		conditions := []string{
			"id = 'unknown'",
			"device_id = 'unknown'",
			"device_id::text LIKE '%unknown%'",
			"username LIKE '%unknown%'",
			"name LIKE '%unknown%'",
			"email LIKE '%unknown%'",
		}

		for _, cond := range conditions {
			var count int64
			if err := db.Table(table).Where(cond).Count(&count).Error; err == nil && count > 0 {
				log.Printf("  Found %d record(s) in %s with %s", count, table, cond)
			}
		}
	}

	log.Println("✅ Check complete")
}
