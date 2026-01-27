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

	// Получить все таблицы
	var tables []string
	db.Raw("SELECT table_name FROM information_schema.tables WHERE table_schema='public' AND table_type='BASE TABLE' ORDER BY table_name").Scan(&tables)

	fmt.Printf("Found %d tables\n\n", len(tables))

	// Проверяем каждую таблицу на camelCase колонки
	for _, table := range tables {
		var cols []struct {
			Name string
		}

		db.Raw("SELECT column_name FROM information_schema.columns WHERE table_schema='public' AND table_name=? ORDER BY ordinal_position", table).Scan(&cols)

		// Ищем camelCase колонки (содержит заглавную букву кроме первой)
		camelCaseFound := false
		for _, col := range cols {
			// Простейшая проверка: если есть заглавная буква в имени
			// (snake_case обычно все нижний регистр)
			for i := 1; i < len(col.Name); i++ {
				if col.Name[i] >= 'A' && col.Name[i] <= 'Z' {
					// camelCase найден
					if !camelCaseFound {
						fmt.Printf("Table: %s\n", table)
						camelCaseFound = true
					}
					fmt.Printf("  - %s (possible camelCase)\n", col.Name)
					break
				}
			}
		}

		if camelCaseFound {
			fmt.Println()
		}
	}
}
