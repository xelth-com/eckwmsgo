package printer

import (
	"bytes"
	"fmt"
	"os"

	"github.com/jung-kurt/gofpdf"
	"github.com/skip2/go-qrcode"
)

// LabelConfig holds configuration for PDF generation
type LabelConfig struct {
	Type        string  `json:"type"`        // i, b, p, l
	StartNumber int     `json:"startNumber"` // Starting serial number
	Count       int     `json:"count"`       // How many labels
	Cols        int     `json:"cols"`
	Rows        int     `json:"rows"`
	MarginTop   float64 `json:"marginTop"`
	MarginLeft  float64 `json:"marginLeft"`
	GapX        float64 `json:"gapX"`
	GapY        float64 `json:"gapY"`
}

// GenerateLabelsPDF creates a PDF with QR codes
func GenerateLabelsPDF(cfg LabelConfig) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(0, 0, 0)
	pdf.SetAutoPageBreak(false, 0)

	// Default font
	pdf.SetFont("Arial", "B", 10)

	// A4 dimensions
	pageWidth, pageHeight := 210.0, 297.0

	// Calculate label size
	totalGapX := float64(cfg.Cols-1) * cfg.GapX
	totalGapY := float64(cfg.Rows-1) * cfg.GapY

	// Available space
	availW := pageWidth - (cfg.MarginLeft * 2) // Assuming symmetric horizontal margins
	availH := pageHeight - (cfg.MarginTop * 2) // Assuming symmetric vertical margins

	labelW := (availW - totalGapX) / float64(cfg.Cols)
	labelH := (availH - totalGapY) / float64(cfg.Rows)

	instanceSuffix := os.Getenv("INSTANCE_SUFFIX")
	if instanceSuffix == "" {
		instanceSuffix = "IB" // Default fallback
	}

	labelsPerPage := cfg.Cols * cfg.Rows

	for i := 0; i < cfg.Count; i++ {
		// New page logic
		if i%labelsPerPage == 0 {
			pdf.AddPage()
		}

		indexOnPage := i % labelsPerPage
		col := indexOnPage % cfg.Cols
		row := indexOnPage / cfg.Cols

		// Calculate Position (Top-Left of label)
		x := cfg.MarginLeft + float64(col)*(labelW+cfg.GapX)
		y := cfg.MarginTop + float64(row)*(labelH+cfg.GapY)

		// Generate ID and QR Content
		currentID := cfg.StartNumber + i
		idString := fmt.Sprintf("%s%018d", cfg.Type, currentID) // e.g., i000000000000000001

		// In a real scenario, you might encrypt this ID. For now, we use plain ID + suffix protocol
		// Protocol: ECK1.COM/{ID}{SUFFIX}
		qrContent := fmt.Sprintf("ECK1.COM/%s%s", idString, instanceSuffix)

		// Generate QR Image
		qrPng, err := qrcode.Encode(qrContent, qrcode.Low, 256)
		if err != nil {
			return nil, err
		}

		// Embed Image into PDF
		imgName := fmt.Sprintf("qr_%d", i)
		imgOptions := gofpdf.ImageOptions{
			ImageType: "PNG",
			ReadDpi:   true,
		}

		// Load image from buffer
		reader := bytes.NewReader(qrPng)
		_ = pdf.RegisterImageOptionsReader(imgName, imgOptions, reader)

		// Draw QR Code (Centered in label, taking up 70% height)
		qrSize := labelH * 0.7
		if qrSize > labelW {
			qrSize = labelW * 0.9
		}

		qrX := x + (labelW-qrSize)/2
		qrY := y + (labelH-qrSize)/2 - 2 // Shift up slightly for text space

		pdf.ImageOptions(imgName, qrX, qrY, qrSize, qrSize, false, imgOptions, 0, "")

		// Draw Text (Serial Number) below QR
		pdf.SetXY(x, y+labelH-6)
		pdf.SetFontSize(8)
		pdf.CellFormat(labelW, 5, idString, "", 0, "C", false, 0, "")

		// Draw Text (Type) top right
		pdf.SetXY(x, y+1)
		pdf.SetFontSize(6)
		pdf.CellFormat(labelW, 3, cfg.Type, "", 0, "R", false, 0, "")

		// Optional: Draw border for debugging or cutting
		// pdf.Rect(x, y, labelW, labelH, "D")
	}

	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
