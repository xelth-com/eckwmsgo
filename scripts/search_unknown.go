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

	// Get all table names
	var tables []string
	db.Raw("SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' AND table_type = 'BASE TABLE'").Scan(&tables)

	log.Println("🔍 Searching for 'unknown' values in tables...")

	// Search for 'unknown' in text columns
	for _, table := range tables {
		// Skip metadata tables
		if table == "schema_migrations" || table == "gorm_migrations" {
			continue
		}

		var columns []struct {
			ColumnName string
		}
		db.Raw(`
			SELECT column_name 
			FROM information_schema.columns 
			WHERE table_schema = 'public' 
			AND table_name = ? 
			AND data_type IN ('character varying', 'text', 'varchar')
		`, table).Scan(&columns)

		for _, col := range columns {
			var count int64
			db.Raw(fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s = 'unknown'", table, col.ColumnName)).Scan(&count)

			if count > 0 {
				log.Printf("  Found %d 'unknown' in %s.%s", count, table, col.ColumnName)
			}
		}
	}

	log.Println("✅ Search complete")
}
