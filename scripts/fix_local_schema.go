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

	fmt.Println("=== Copying data from snake_case to camelCase columns ===")

	// Copy device_id -> deviceId
	result := db.Exec(`UPDATE registered_devices SET "deviceId" = device_id WHERE "deviceId" IS NULL AND device_id IS NOT NULL`)
	fmt.Printf("Copied device_id -> deviceId: %d rows\n", result.RowsAffected)

	// Copy created_at -> createdAt
	result = db.Exec(`UPDATE registered_devices SET "createdAt" = created_at WHERE "createdAt" IS NULL AND created_at IS NOT NULL`)
	fmt.Printf("Copied created_at -> createdAt: %d rows\n", result.RowsAffected)

	// Copy updated_at -> updatedAt
	result = db.Exec(`UPDATE registered_devices SET "updatedAt" = updated_at WHERE "updatedAt" IS NULL AND updated_at IS NOT NULL`)
	fmt.Printf("Copied updated_at -> updatedAt: %d rows\n", result.RowsAffected)

	// Copy last_seen_at -> lastSeenAt
	result = db.Exec(`UPDATE registered_devices SET "lastSeenAt" = last_seen_at WHERE "lastSeenAt" IS NULL AND last_seen_at IS NOT NULL`)
	fmt.Printf("Copied last_seen_at -> lastSeenAt: %d rows\n", result.RowsAffected)

	// Copy public_key -> publicKey
	result = db.Exec(`UPDATE registered_devices SET "publicKey" = public_key WHERE "publicKey" IS NULL AND public_key IS NOT NULL`)
	fmt.Printf("Copied public_key -> publicKey: %d rows\n", result.RowsAffected)

	fmt.Println("\n=== Updated Device Data ===")
	var devices []map[string]interface{}
	db.Raw(`SELECT "deviceId", name, status, "lastSeenAt", "updatedAt" FROM registered_devices`).Scan(&devices)
	for i, d := range devices {
		fmt.Printf("Device %d: %+v\n", i, d)
	}
}
