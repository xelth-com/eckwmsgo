package utils

import (
	"testing"
	"time"
)

func TestSmartItemCode(t *testing.T) {
	// Test with EAN-13 (13 chars)
	serial := "ABC123"
	ean := "1234567890123"

	code := GenerateSmartItem(serial, ean)
	t.Logf("Generated Item Code: %s", code)

	decoded, err := DecodeSmartItem(code)
	if err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	if decoded.Serial != serial {
		t.Errorf("Serial mismatch: got %s, want %s", decoded.Serial, serial)
	}

	if decoded.RefID != ean {
		t.Errorf("RefID mismatch: got %s, want %s", decoded.RefID, ean)
	}

	t.Logf("Decoded: Serial=%s, RefID=%s", decoded.Serial, decoded.RefID)
}

func TestSmartBoxCode(t *testing.T) {
	box := SmartBoxData{
		Length: 40,
		Width:  30,
		Height: 25,
		Weight: 5.5, // 5.5kg
		Type:   "B",
		Serial: 12345,
	}

	code, err := GenerateSmartBox(box)
	if err != nil {
		t.Fatalf("Failed to generate: %v", err)
	}

	t.Logf("Generated Box Code: %s (len=%d)", code, len(code))

	decoded, err := DecodeSmartBox(code)
	if err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	if decoded.Length != box.Length {
		t.Errorf("Length mismatch: got %d, want %d", decoded.Length, box.Length)
	}
	if decoded.Width != box.Width {
		t.Errorf("Width mismatch: got %d, want %d", decoded.Width, box.Width)
	}
	if decoded.Height != box.Height {
		t.Errorf("Height mismatch: got %d, want %d", decoded.Height, box.Height)
	}

	// Weight might have small precision loss due to tiers
	weightDiff := decoded.Weight - box.Weight
	if weightDiff < -0.01 || weightDiff > 0.01 {
		t.Errorf("Weight mismatch: got %.2f, want %.2f", decoded.Weight, box.Weight)
	}

	t.Logf("Decoded: L=%d W=%d H=%d Weight=%.2fkg Type=%s Serial=%d",
		decoded.Length, decoded.Width, decoded.Height, decoded.Weight, decoded.Type, decoded.Serial)
}

func TestSmartLabelCode(t *testing.T) {
	label := SmartLabelData{
		Date:    time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
		Type:    "A",
		Payload: "TESTPAYLOAD123",
	}

	code, err := GenerateSmartLabel(label)
	if err != nil {
		t.Fatalf("Failed to generate: %v", err)
	}

	t.Logf("Generated Label Code: %s (len=%d)", code, len(code))

	decoded, err := DecodeSmartLabel(code)
	if err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	if !decoded.Date.Equal(label.Date) {
		t.Errorf("Date mismatch: got %s, want %s", decoded.Date, label.Date)
	}
	if decoded.Type != label.Type {
		t.Errorf("Type mismatch: got %s, want %s", decoded.Type, label.Type)
	}

	t.Logf("Decoded: Date=%s Type=%s Payload=%s",
		decoded.Date.Format("2006-01-02"), decoded.Type, decoded.Payload)
}

func TestWeightEncoding(t *testing.T) {
	testCases := []float64{
		0.5,    // Tier 1: 10g precision
		10.0,   // Tier 1
		19.99,  // Tier 1 boundary
		50.0,   // Tier 2: 100g precision
		500.0,  // Tier 2
		999.9,  // Tier 2 boundary
		1500.0, // Tier 3: 1kg precision
		10000.0, // Tier 3
	}

	for _, weight := range testCases {
		encoded, err := encodeWeight(weight)
		if err != nil {
			t.Errorf("Failed to encode weight %.2f: %v", weight, err)
			continue
		}

		decoded := decodeWeight(encoded)

		// Check precision based on tier
		var tolerance float64
		if weight <= Tier1Limit {
			tolerance = 0.01
		} else if weight <= Tier2Limit {
			tolerance = 0.1
		} else {
			tolerance = 1.0
		}

		diff := decoded - weight
		if diff < -tolerance || diff > tolerance {
			t.Errorf("Weight %.2fkg: encoded=%d, decoded=%.2fkg (diff=%.3f, tolerance=%.3f)",
				weight, encoded, decoded, diff, tolerance)
		} else {
			t.Logf("Weight %.2fkg -> %d -> %.2fkg ✓", weight, encoded, decoded)
		}
	}
}

func TestBase32Conversion(t *testing.T) {
	testCases := []struct {
		num   int
		width int
		want  string
	}{
		{0, 1, "0"},
		{10, 1, "A"},
		{35, 1, "Z"},
		{36, 2, "10"},
		{100, 2, "2S"},
		{1023, 2, "SF"}, // 36^2 - 1 would be "ZZ" (1295), so 1023 = 28*36 + 15 = SF
		{46655, 3, "ZZZ"}, // 36^3 - 1
	}

	for _, tc := range testCases {
		got := intToBase32(tc.num, tc.width)
		if got != tc.want {
			t.Errorf("intToBase32(%d, %d) = %s, want %s", tc.num, tc.width, got, tc.want)
		}

		// Test round-trip
		decoded := base32ToInt(got)
		if decoded != tc.num {
			t.Errorf("Round-trip failed: %d -> %s -> %d", tc.num, got, decoded)
		}
	}
}
