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

	// Reset last_sync_at for production node (shipments_push entity)
	result := db.Exec("UPDATE sync_metadata SET last_sync_at = '2020-01-01' WHERE instance_id = 'production_pda_repair' AND entity_type = 'shipments_push'")
	if result.Error != nil {
		log.Fatal("Failed to update:", result.Error)
	}

	log.Printf("✅ Reset shipments_push sync timestamp. Rows affected: %d", result.RowsAffected)

	// If no rows affected, maybe the metadata doesn't exist yet - that's OK
	if result.RowsAffected == 0 {
		log.Println("⚠️  No existing sync_metadata found for shipments_push - this is normal for first sync")
	}
	log.Println("Now trigger mesh sync: curl -X POST http://localhost:3210/api/mesh/trigger")
}
