package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	godotenv.Load()
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("PG_HOST"), os.Getenv("PG_PORT"), os.Getenv("PG_USERNAME"), os.Getenv("PG_PASSWORD"), os.Getenv("PG_DATABASE"))
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		fmt.Println("DB Error:", err)
		return
	}

	fmt.Println("=== LOCAL INVENTORY ===")
	var count int64

	db.Table("product_product").Count(&count)
	fmt.Printf("product_product: %d\n", count)

	db.Table("stock_quant").Count(&count)
	fmt.Printf("stock_quant: %d\n", count)

	db.Table("stock_location").Count(&count)
	fmt.Printf("stock_location: %d\n", count)

	fmt.Println("\n=== PRODUCTS ===")
	var products []map[string]interface{}
	db.Table("product_product").Select("id, name, default_code").Find(&products)
	for _, p := range products {
		fmt.Printf("  ID=%v, Code=%v, Name=%v\n", p["id"], p["default_code"], p["name"])
	}
}
