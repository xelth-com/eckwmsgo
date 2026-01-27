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

	type Row struct {
		ID             int
		TrackingNumber string
		Status         string
		RawResponse    string
	}

	var rows []Row
	err = db.Raw("SELECT id, tracking_number, status, raw_response FROM stock_picking_delivery ORDER BY id DESC LIMIT 10").Scan(&rows).Error
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== Local DB ===")
	for _, r := range rows {
		lenStr := "EMPTY"
		if len(r.RawResponse) > 0 {
			lenStr = fmt.Sprintf("%d chars", len(r.RawResponse))
		}
		fmt.Printf("ID=%d | Tracking=%s | Status=%s | Raw=%s\n", r.ID, r.TrackingNumber, r.Status, lenStr)
	}
}
