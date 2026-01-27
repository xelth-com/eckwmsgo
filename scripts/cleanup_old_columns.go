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

	log.Println("🔄 Final cleanup: Removing old camelCase columns...")

	// Список старых camelCase колонок для удаления
	oldColumns := []struct {
		table   string
		column  string
		comment string
	}{
		// registered_devices
		{"registered_devices", "device_name", "Old snake_case column (replaced by name)"},
		{"registered_devices", "deviceName", "Old camelCase column (replaced by name)"},
		{"registered_devices", "instance_id", "Old snake_case column (not used)"},
		{"registered_devices", "is_active", "Old snake_case column (replaced by status)"},
		{"registered_devices", "role_id", "Old snake_case column (not used)"},

		// product_aliases
		{"product_aliases", "created_context", "Old snake_case column (replaced by createdContext)"},
		{"product_aliases", "createdContext", "Old camelCase column (replaced by createdContext)"},

		// user_auths
		// (если есть старые колонки, они будут здесь)
	}

	for _, col := range oldColumns {
		// Проверяем существует ли колонка
		var colExists bool
		db.Raw(`
			SELECT EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_schema = 'public'
				AND table_name = ?
				AND column_name = ?
			)
		`, col.table, col.column).Scan(&colExists)

		if colExists {
			dropSQL := fmt.Sprintf(`ALTER TABLE %s DROP COLUMN IF EXISTS "%s"`, col.table, col.column)
			result := db.Exec(dropSQL)
			if result.Error != nil {
				log.Printf("  ⚠️  Ошибка удаления %s.%s: %v", col.table, col.column, result.Error)
			} else {
				log.Printf("  ✅ Удалена старая колонка: %s.%s (%s)", col.table, col.column, col.comment)
			}
		}
	}

	log.Println("\n✅ Cleanup completed!")
	log.Println("\n📊 Database is now clean and standardized to snake_case")
	log.Println("   - Go models use PascalCase")
	log.Println("   - DB columns use snake_case")
	log.Println("   - JSON API uses camelCase")
}
