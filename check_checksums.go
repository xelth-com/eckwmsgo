package main

import (
	"fmt"
	"log"

	"github.com/xelth-com/eckwmsgo/internal/config"
	"github.com/xelth-com/eckwmsgo/internal/database"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	db, err := database.Connect(cfg.Database)
	if err != nil {
		log.Fatal(err)
	}

	var counts []struct {
		EntityType string
		Count      int
	}
	db.Raw("SELECT entity_type, COUNT(*) as count FROM entity_checksums GROUP BY entity_type ORDER BY entity_type").Scan(&counts)

	fmt.Println("=== Local Checksums ===")
	total := 0
	for _, c := range counts {
		fmt.Printf("%s: %d\n", c.EntityType, c.Count)
		total += c.Count
	}
	fmt.Printf("\nTotal: %d checksums\n", total)
}
