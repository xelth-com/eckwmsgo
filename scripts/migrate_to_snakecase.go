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

	log.Println("🔄 Миграция базы данных на snake_case...")

	// Миграция таблицы registered_devices
	log.Println("\n📱 Таблица: registered_devices")

	// 1. Переименовать camelCase колонки в snake_case
	migrations := []struct {
		table   string
		oldCol  string
		newCol  string
		example string
	}{
		// registered_devices
		{"registered_devices", "deviceId", "device_id", "ID устройства"},
		{"registered_devices", "deviceName", "device_name", "Имя устройства"},
		{"registered_devices", "publicKey", "public_key", "Публичный ключ"},
		{"registered_devices", "lastSeenAt", "last_seen_at", "Дата последнего визита"},
		{"registered_devices", "createdAt", "created_at", "Дата создания"},
		{"registered_devices", "updatedAt", "updated_at", "Дата обновления"},

		// product_aliases
		{"product_aliases", "externalCode", "external_code", "Внешний код"},
		{"product_aliases", "internalId", "internal_id", "Внутренний ID"},
		{"product_aliases", "isVerified", "is_verified", "Проверено"},
		{"product_aliases", "confidenceScore", "confidence_score", "Уверенность"},
		{"product_aliases", "createdContext", "created_context", "Контекст создания"},
		{"product_aliases", "createdAt", "created_at", "Дата создания"},
		{"product_aliases", "updatedAt", "updated_at", "Дата обновления"},

		// user_auths
		{"user_auths", "userType", "user_type", "Тип пользователя"},
		{"user_auths", "googleId", "google_id", "Google ID"},
		{"user_auths", "isActive", "is_active", "Активен"},
		{"user_auths", "lastLogin", "last_login", "Последний вход"},
		{"user_auths", "failedLoginAttempts", "failed_login_attempts", "Неудачные попытки"},
		{"user_auths", "preferredLanguage", "preferred_language", "Предпочитаемый язык"},
		{"user_auths", "createdAt", "created_at", "Дата создания"},
		{"user_auths", "updatedAt", "updated_at", "Дата обновления"},
	}

	for _, m := range migrations {
		// Проверяем существует ли старая колонка
		var colExists bool
		db.Raw(`
			SELECT EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_schema = 'public'
				AND table_name = ?
				AND column_name = ?
			)
		`, m.table, m.oldCol).Scan(&colExists)

		if colExists {
			// Проверяем существует ли новая колонка
			var newColExists bool
			db.Raw(`
				SELECT EXISTS (
					SELECT 1
					FROM information_schema.columns
					WHERE table_schema = 'public'
					AND table_name = ?
					AND column_name = ?
				)
			`, m.table, m.newCol).Scan(&newColExists)

			if newColExists {
				// Новая колонка уже есть - скопируем данные и удалим старую
				log.Printf("  📋 Объединение данных: %s.%s → %s", m.table, m.oldCol, m.newCol)

				// Копируем данные если они есть
				mergeSQL := fmt.Sprintf(`
					UPDATE %s
					SET %s = COALESCE(%s, %s)
					WHERE %s IS NULL OR %s = ''
				`, m.table, m.newCol, m.newCol, m.oldCol, m.newCol, m.newCol)

				result := db.Exec(mergeSQL)
				if result.Error != nil {
					log.Printf("  ⚠️  Ошибка объединения: %v", result.Error)
				} else {
					log.Printf("  ✅ Данные объединены, затронуто строк: %d", result.RowsAffected)
				}

				// Удаляем старую колонку
				dropSQL := fmt.Sprintf(`ALTER TABLE %s DROP COLUMN IF EXISTS "%s"`, m.table, m.oldCol)
				result = db.Exec(dropSQL)
				if result.Error != nil {
					log.Printf("  ⚠️  Ошибка удаления колонки: %v", result.Error)
				} else {
					log.Printf("  ✅ Удалена старая колонка: %s", m.oldCol)
				}
			} else {
				// Новой колонки нет - переименовываем
				renameSQL := fmt.Sprintf(`ALTER TABLE %s RENAME COLUMN "%s" TO %s`, m.table, m.oldCol, m.newCol)
				result := db.Exec(renameSQL)
				if result.Error != nil {
					log.Printf("  ❌ Ошибка переименования %s → %s: %v", m.oldCol, m.newCol, result.Error)
				} else {
					log.Printf("  ✅ Переименовано: %s → %s (%s)", m.oldCol, m.newCol, m.example)
				}
			}
		}
	}

	log.Println("\n✅ Миграция завершена!")
	log.Println("\n📊 Следующие шаги:")
	log.Println("   1. Проверьте что все колонки теперь в snake_case")
	log.Println("   2. Перекомпилируйте приложение: go build")
	log.Println("   3. Перезапустите сервер")
	log.Println("   4. Протестируйте регистрацию устройства")
}
