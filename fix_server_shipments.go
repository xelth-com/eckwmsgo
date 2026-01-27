package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

func main() {
	// Connect to local DB
	localDB, err := sql.Open("postgres", "host=localhost port=5432 user=openpg password=openpgpwd dbname=eckwms sslmode=disable")
	if err != nil {
		log.Fatal("Local DB:", err)
	}
	defer localDB.Close()

	// Connect to server DB via SSH tunnel or directly if accessible
	// For now, just export SQL statements

	// Get shipments with empty rawResponse on server
	emptyIDs := []int{219, 220, 221, 222, 223, 224}

	fmt.Println("-- SQL statements to fix server shipments")
	fmt.Println("-- Run these on the server database")
	fmt.Println()

	for _, id := range emptyIDs {
		var trackingNum, rawResponse string
		err := localDB.QueryRow("SELECT tracking_number, raw_response FROM stock_picking_delivery WHERE id = $1", id).Scan(&trackingNum, &rawResponse)
		if err != nil {
			log.Printf("Error reading ID %d: %v", id, err)
			continue
		}

		if len(rawResponse) == 0 {
			log.Printf("Warning: ID %d also empty in local DB", id)
			continue
		}

		// Escape single quotes in JSON
		escapedJSON := ""
		for _, ch := range rawResponse {
			if ch == '\'' {
				escapedJSON += "''"
			} else {
				escapedJSON += string(ch)
			}
		}

		fmt.Printf("UPDATE stock_picking_delivery SET raw_response = '%s' WHERE id = %d;\n", escapedJSON, id)
	}
}
