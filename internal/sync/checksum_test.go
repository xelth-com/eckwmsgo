package sync

import (
	"testing"
	"time"

	"github.com/xelth-com/eckwmsgo/internal/models"
)

func TestChecksumCalculator_ComputeChecksum(t *testing.T) {
	calc := NewChecksumCalculator("test-instance")

	// Test with ProductProduct
	product := models.ProductProduct{
		ID:            123,
		DefaultCode:   "TEST-SKU-001",
		Barcode:       "1234567890123",
		Name:          "Test Product",
		Active:        true,
		Type:          "consu",
		ListPrice:     99.99,
		StandardPrice: 79.99,
		Weight:        1.5,
		Volume:        0.5,
		WriteDate:     time.Now(), // Should be excluded from checksum
		LastSyncedAt:  time.Now(), // Should be excluded from checksum
	}

	hash1, err := calc.ComputeChecksum(product)
	if err != nil {
		t.Fatalf("Failed to compute checksum: %v", err)
	}

	if hash1 == "" {
		t.Error("Expected non-empty hash")
	}

	if len(hash1) != 64 {
		t.Errorf("Expected 64-character SHA256 hash, got %d characters", len(hash1))
	}

	// Compute again - should be deterministic
	hash2, err := calc.ComputeChecksum(product)
	if err != nil {
		t.Fatalf("Failed to compute checksum on second attempt: %v", err)
	}

	if hash1 != hash2 {
		t.Error("Checksum should be deterministic")
	}

	// Change a field - hash should change
	product.Name = "Modified Product"
	hash3, err := calc.ComputeChecksum(product)
	if err != nil {
		t.Fatalf("Failed to compute checksum after modification: %v", err)
	}

	if hash1 == hash3 {
		t.Error("Checksum should change when content changes")
	}

	// Change timestamp only - hash should NOT change (timestamps excluded)
	product.Name = "Test Product" // Restore original
	product.LastSyncedAt = time.Now().Add(1 * time.Hour)
	hash4, err := calc.ComputeChecksum(product)
	if err != nil {
		t.Fatalf("Failed to compute checksum after timestamp change: %v", err)
	}

	if hash1 != hash4 {
		t.Error("Checksum should NOT change when only timestamps change")
	}
}

func TestChecksumCalculator_StockLocation(t *testing.T) {
	calc := NewChecksumCalculator("test-instance")

	location := models.StockLocation{
		ID:           456,
		Name:         "WH/Stock",
		CompleteName: "Warehouse/Stock",
		Barcode:      "p-LOC-001",
		Usage:        "internal",
		Active:       true,
		LastSyncedAt: time.Now(),
	}

	hash1, err := calc.ComputeChecksum(location)
	if err != nil {
		t.Fatalf("Failed to compute checksum for location: %v", err)
	}

	if hash1 == "" {
		t.Error("Expected non-empty hash for location")
	}

	// Verify deterministic
	hash2, err := calc.ComputeChecksum(location)
	if err != nil {
		t.Fatalf("Failed to compute checksum on second attempt: %v", err)
	}

	if hash1 != hash2 {
		t.Error("Location checksum should be deterministic")
	}
}

func TestChecksumCalculator_StockQuant(t *testing.T) {
	calc := NewChecksumCalculator("test-instance")

	quant := models.StockQuant{
		ID:          789,
		ProductID:   123,
		LocationID:  456,
		Quantity:    100.0,
		ReservedQty: 25.0,
	}

	hash1, err := calc.ComputeChecksum(quant)
	if err != nil {
		t.Fatalf("Failed to compute checksum for quant: %v", err)
	}

	if hash1 == "" {
		t.Error("Expected non-empty hash for quant")
	}

	// Change quantity - should change hash
	quant.Quantity = 150.0
	hash2, err := calc.ComputeChecksum(quant)
	if err != nil {
		t.Fatalf("Failed to compute checksum after quantity change: %v", err)
	}

	if hash1 == hash2 {
		t.Error("Quant checksum should change when quantity changes")
	}
}
