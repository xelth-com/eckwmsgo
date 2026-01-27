package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

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

	log.Println("🔄 Сканирование базы данных на camelCase колонки...")

	// 1. Получить все таблицы
	var tables []string
	db.Raw(`
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = 'public'
		AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`).Scan(&tables)

	log.Printf("\n📋 Найдено таблиц: %d\n", len(tables))

	// Регулярное выражение для camelCase
	camelCaseRegex := regexp.MustCompile(`^[a-z][a-zA-Z0-9]*[A-Z][a-zA-Z0-9]*$`)

	// 2. Проверяем каждую таблицу
	totalCamelCase := 0
	totalFixed := 0

	for _, table := range tables {
		// Пропускаем системные таблицы
		if strings.HasPrefix(table, "gorm_") || strings.HasPrefix(table, "schema_") {
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
			ORDER BY ordinal_position
		`, table).Scan(&columns)

		camelCaseCols := []string{}
		for _, col := range columns {
			if camelCaseRegex.MatchString(col.ColumnName) {
				camelCaseCols = append(camelCaseCols, col.ColumnName)
			}
		}

		if len(camelCaseCols) > 0 {
			log.Printf("📱 Таблица %s: найдено camelCase колонок: %d", table, len(camelCaseCols))
			for _, camelCol := range camelCaseCols {
				log.Printf("   - %s", camelCol)

				// Конвертируем в snake_case
				snakeCol := camelToSnake(camelCol)
				log.Printf("     → snake_case: %s", snakeCol)

				// Проверяем существует ли snake_case версия
				var snakeExists bool
				db.Raw(`
					SELECT EXISTS (
						SELECT 1
						FROM information_schema.columns
						WHERE table_schema = 'public'
						AND table_name = ?
						AND column_name = ?
					)
				`, table, snakeCol).Scan(&snakeExists)

				if snakeExists {
					// Объединяем данные и удаляем camelCase
					log.Printf("     ✅ Объединение данных: %s → %s", camelCol, snakeCol)

					mergeSQL := fmt.Sprintf(`
						UPDATE %s
						SET %s = COALESCE(%s, %s)
						WHERE %s IS NULL OR %s = ''
					`, table, snakeCol, snakeCol, camelCol, snakeCol, snakeCol)

					result := db.Exec(mergeSQL)
					if result.Error != nil {
						log.Printf("     ⚠️  Ошибка объединения: %v", result.Error)
					} else {
						log.Printf("     ✅ Объединено, строк: %d", result.RowsAffected)
					}

					// Удаляем camelCase колонку
					dropSQL := fmt.Sprintf(`ALTER TABLE %s DROP COLUMN IF EXISTS "%s"`, table, camelCol)
					result = db.Exec(dropSQL)
					if result.Error != nil {
						log.Printf("     ⚠️  Ошибка удаления: %v", result.Error)
					} else {
						log.Printf("     ✅ Удалена camelCase колонка")
						totalFixed++
					}
				} else {
					// Переименовываем camelCase в snake_case
					renameSQL := fmt.Sprintf(`ALTER TABLE %s RENAME COLUMN "%s" TO %s`, table, camelCol, snakeCol)
					result := db.Exec(renameSQL)
					if result.Error != nil {
						log.Printf("     ❌ Ошибка переименования: %v", result.Error)
					} else {
						log.Printf("     ✅ Переименовано")
						totalFixed++
					}
				}
			}
			totalCamelCase += len(camelCaseCols)
		}
	}

	log.Println("\n📊 Итого:")
	log.Printf("   Найдено camelCase колонок: %d", totalCamelCase)
	log.Printf("   Исправлено колонок: %d", totalFixed)

	if totalCamelCase == 0 {
		log.Println("\n✅ Все колонки уже в snake_case!")
	} else {
		log.Println("\n✅ Миграция завершена!")
	}
}

// camelToSnake конвертирует camelCase в snake_case
func camelToSnake(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, r)
	}
	return strings.ToLower(string(result))
}
