package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type Shipment struct {
	ID             int64     `json:"id"`
	PickingID      *int64    `json:"pickingId"`
	TrackingNumber string    `json:"trackingNumber"`
	Status         string    `json:"status"`
	CreatedAt      string    `json:"createdAt"`
	RawResponse    string    `json:"rawResponse"`
}

func main() {
	resp, err := http.Get("http://localhost:3210/E/api/delivery/shipments")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var shipments []Shipment
	if err := json.Unmarshal(body, &shipments); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("API returned %d shipments:\n", len(shipments))
	fmt.Println("ID\tPickingID\tTracking\tStatus\tCreatedAt")
	fmt.Println("--------------------------------------------------------")
	for _, s := range shipments {
		pickingIDStr := "NULL"
		if s.PickingID != nil {
			pickingIDStr = fmt.Sprintf("%d", *s.PickingID)
		}
		fmt.Printf("%d\t%s\t%s\t%s\t%s\n", s.ID, pickingIDStr, s.TrackingNumber, s.Status, s.CreatedAt)
	}
}
