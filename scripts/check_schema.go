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
		os.Getenv("PG_HOST"), os.Getenv("PG_PORT"), os.Getenv("PG_USERNAME"), os.Getenv("PG_PASSWORD"), os.Getenv("PG_DATABASE"))
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		fmt.Println("DB Error:", err)
		return
	}

	fmt.Println("=== LOCAL DB: registered_devices columns ===")
	var result []map[string]interface{}
	db.Raw("SELECT column_name FROM information_schema.columns WHERE table_name = 'registered_devices' ORDER BY ordinal_position").Scan(&result)
	for _, r := range result {
		fmt.Println(" -", r["column_name"])
	}

	fmt.Println("\n=== Device Data ===")
	var devices []map[string]interface{}
	db.Raw("SELECT * FROM registered_devices").Scan(&devices)
	for i, d := range devices {
		fmt.Printf("Device %d: %+v\n", i, d)
	}
}
