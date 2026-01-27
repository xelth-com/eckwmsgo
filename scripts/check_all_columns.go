package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	godotenv.Load()

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("PG_HOST"), os.Getenv("PG_PORT"),
		os.Getenv("PG_USERNAME"), os.Getenv("PG_PASSWORD"),
		os.Getenv("PG_DATABASE"))

	db, _ := gorm.Open(postgres.Open(dsn), &gorm.Config{})

	tables := []string{"registered_devices", "product_aliases", "user_auths", "sync_metadata", "entity_checksums", "stock_location", "stock_picking", "delivery_carrier"}
	for _, table := range tables {
		var cols []string
		db.Raw("SELECT column_name FROM information_schema.columns WHERE table_schema='public' AND table_name=? ORDER BY ordinal_position", table).Scan(&cols)
		fmt.Printf("%s:\n", table)
		for _, col := range cols {
			fmt.Printf("  %s\n", col)
		}
		fmt.Println()
	}
}
