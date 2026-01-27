package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	db, err := sql.Open("postgres", "host=localhost port=5432 user=openpg password=openpgpwd dbname=eckwms sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Check total shipments
	var total int
	db.QueryRow("SELECT COUNT(*) FROM stock_picking_delivery").Scan(&total)
	fmt.Printf("Total shipments: %d\n", total)

	rows, err := db.Query(`
		SELECT id, picking_id, tracking_number, status, created_at 
		FROM stock_picking_delivery 
		ORDER BY created_at DESC 
		LIMIT 10
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	count := 0
	fmt.Println("\nShipments in database:")
	fmt.Println("ID\tPickingID\tTracking\tStatus\tCreatedAt")
	fmt.Println("--------------------------------------------------------")
	for rows.Next() {
		var id, pickingID sql.NullInt64
		var tracking, status, createdAt string
		err := rows.Scan(&id, &pickingID, &tracking, &status, &createdAt)
		if err != nil {
			log.Fatal(err)
		}
		pickingIDStr := "NULL"
		if pickingID.Valid {
			pickingIDStr = fmt.Sprintf("%d", pickingID.Int64)
		}
		fmt.Printf("%d\t%s\t%s\t%s\t%s\n", id.Int64, pickingIDStr, tracking, status, createdAt)
		count++
	}

	if count == 0 {
		fmt.Println("No shipments found in database")
		os.Exit(1)
	}
}
