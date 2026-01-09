package printer

import (
	"bytes"
	"fmt"
	"hash/crc32"
	"math"
	"os"

	"github.com/dmytrosurovtsev/eckwmsgo/internal/utils"
	"github.com/jung-kurt/gofpdf"
	"github.com/skip2/go-qrcode"
)

// ElementConfig defines position and scale for a label element
type ElementConfig struct {
	X     float64 `json:"x"`     // Position in % of label width (from left)
	Y     float64 `json:"y"`     // Position in % of label height (from bottom)
	Scale float64 `json:"scale"` // Scale relative to min(W, H)
}

// ContentConfig holds configuration for all visible elements
type ContentConfig struct {
	QR1      *ElementConfig `json:"qr1,omitempty"`
	QR2      *ElementConfig `json:"qr2,omitempty"`
	QR3      *ElementConfig `json:"qr3,omitempty"`
	Checksum *ElementConfig `json:"checksum,omitempty"`
	Serial   *ElementConfig `json:"serial,omitempty"`
}

// RegalConfig defines a rack structure for place calculation
type RegalConfig struct {
	Index      int `json:"index"`       // 1-based regal index
	Columns    int `json:"columns"`     // Number of columns in this regal
	Rows       int `json:"rows"`        // Number of rows in this regal
	StartIndex int `json:"start_index"` // Starting place index for this regal
}

// WarehouseConfig holds the full warehouse structure
type WarehouseConfig struct {
	Regals []RegalConfig `json:"regals"`
}

// LabelConfig holds configuration for PDF generation
type LabelConfig struct {
	Type            string           `json:"type"`            // i, b, p, l
	StartNumber     int              `json:"startNumber"`     // Starting serial number
	Count           int              `json:"count"`           // How many labels
	Cols            int              `json:"cols"`            // Columns per page
	Rows            int              `json:"rows"`            // Rows per page
	MarginTop       float64          `json:"marginTop"`       // Top margin in mm
	MarginLeft      float64          `json:"marginLeft"`      // Left margin in mm
	MarginRight     float64          `json:"marginRight"`     // Right margin in mm
	MarginBottom    float64          `json:"marginBottom"`    // Bottom margin in mm
	GapX            float64          `json:"gapX"`            // Horizontal gap between labels in mm
	GapY            float64          `json:"gapY"`            // Vertical gap between labels in mm
	IsTightMode     bool             `json:"isTightMode"`     // Overlap mode (tight=true means labels touch)
	SerialDigits    int              `json:"serialDigits"`    // How many digits to show (0 = full 18)
	ContentConfig   *ContentConfig   `json:"contentConfig"`   // Custom element positions
	WarehouseConfig *WarehouseConfig `json:"warehouseConfig"` // Warehouse structure for places
}

// calculateWarehouseLocation maps a place index to regal coordinates
func calculateWarehouseLocation(placeIndex int, config *WarehouseConfig) (regal, col, row int, found bool) {
	if config == nil || len(config.Regals) == 0 {
		return 0, 0, 0, false
	}

	for _, r := range config.Regals {
		placesInRegal := r.Columns * r.Rows
		endIdx := r.StartIndex + placesInRegal - 1

		if placeIndex >= r.StartIndex && placeIndex <= endIdx {
			indexInRegal := placeIndex - r.StartIndex
			column := indexInRegal / r.Rows
			row := indexInRegal % r.Rows
			return r.Index, column + 1, row + 1, true
		}
	}
	return 0, 0, 0, false
}

// formatSerial formats the serial number with prefix
func formatSerial(num int, prefix string, serialDigits int) string {
	padded := fmt.Sprintf("%018d", num)
	if serialDigits > 0 && serialDigits < 18 {
		return prefix + padded[18-serialDigits:]
	}
	return prefix + padded
}

// GenerateLabelsPDF creates a PDF with complex QR puzzle layouts
func GenerateLabelsPDF(cfg LabelConfig) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(0, 0, 0)
	pdf.SetAutoPageBreak(false, 0)

	// A4 dimensions in mm
	pageWidth, pageHeight := 210.0, 297.0

	// Calculate effective margins based on TightMode (Overlap)
	extraX := 0.0
	extraY := 0.0
	if !cfg.IsTightMode {
		extraX = cfg.GapX / 2.0
		extraY = cfg.GapY / 2.0
	}

	effMarginLeft := cfg.MarginLeft + extraX
	effMarginRight := cfg.MarginRight + extraX
	effMarginTop := cfg.MarginTop + extraY
	effMarginBottom := cfg.MarginBottom + extraY

	// Available space for labels
	availW := pageWidth - effMarginLeft - effMarginRight
	availH := pageHeight - effMarginTop - effMarginBottom

	// Label dimensions
	totalGapX := float64(cfg.Cols-1) * cfg.GapX
	totalGapY := float64(cfg.Rows-1) * cfg.GapY
	labelW := (availW - totalGapX) / float64(cfg.Cols)
	labelH := (availH - totalGapY) / float64(cfg.Rows)

	instanceSuffix := os.Getenv("INSTANCE_SUFFIX")
	if instanceSuffix == "" {
		instanceSuffix = "IB"
	}

	labelsPerPage := cfg.Cols * cfg.Rows

	for i := 0; i < cfg.Count; i++ {
		if i%labelsPerPage == 0 {
			pdf.AddPage()
		}

		indexOnPage := i % labelsPerPage
		colIdx := indexOnPage % cfg.Cols
		rowIdx := indexOnPage / cfg.Cols

		// Top-Left of current label box
		originX := effMarginLeft + float64(colIdx)*(labelW+cfg.GapX)
		originY := effMarginTop + float64(rowIdx)*(labelH+cfg.GapY)

		// Data Preparation
		currentID := cfg.StartNumber + i
		idString := fmt.Sprintf("%s%018d", cfg.Type, currentID)

		// Generate encrypted code for QR
		encryptedCode, err := utils.EckURLEncrypt(idString)
		if err != nil {
			// Fallback to plain ID if encryption fails
			encryptedCode = idString
		}

		var field1, field2 string

		// Format Serial (Field 1)
		switch cfg.Type {
		case "i":
			field1 = formatSerial(currentID, "!", cfg.SerialDigits)
		case "b":
			field1 = formatSerial(currentID, "#", cfg.SerialDigits)
		case "p":
			field1 = formatSerial(currentID, "_", cfg.SerialDigits)
		case "l":
			field1 = formatSerial(currentID, "*", cfg.SerialDigits)
		default:
			field1 = formatSerial(currentID, "", cfg.SerialDigits)
		}

		// Format Checksum / Location (Field 2)
		if cfg.Type == "p" && cfg.WarehouseConfig != nil {
			r, c, row, found := calculateWarehouseLocation(currentID, cfg.WarehouseConfig)
			if found {
				field2 = fmt.Sprintf("%s%s%s",
					utils.ToBase32Char(r),
					utils.ToBase32Char(c),
					utils.ToBase32Char(row))
			} else {
				field2 = "???"
			}
		} else {
			// Standard CRC-based 2-char checksum
			temp := crc32.ChecksumIEEE([]byte(fmt.Sprintf("%d", currentID))) & 1023
			field2 = string(utils.Base32Chars[temp>>5]) + string(utils.Base32Chars[temp&31])
		}

		// Drawing Helper
		minSide := math.Min(labelW, labelH)

		drawQR := func(prefix string, elCfg *ElementConfig) {
			if elCfg == nil {
				return
			}
			qrData := fmt.Sprintf("%s/%s%s", prefix, encryptedCode, instanceSuffix)
			qrPng, err := qrcode.Encode(qrData, qrcode.Low, 256)
			if err != nil {
				return
			}

			size := minSide * elCfg.Scale
			posX := originX + (elCfg.X * labelW / 100.0)
			posY := originY + labelH - (elCfg.Y * labelH / 100.0) - size

			imgName := fmt.Sprintf("qr_%d_%s", i, prefix)
			pdf.RegisterImageOptionsReader(imgName, gofpdf.ImageOptions{ImageType: "PNG"}, bytes.NewReader(qrPng))
			pdf.ImageOptions(imgName, posX, posY, size, size, false, gofpdf.ImageOptions{ImageType: "PNG"}, 0, "")
		}

		drawText := func(text string, elCfg *ElementConfig, fontName string, gray float64) {
			if elCfg == nil {
				return
			}
			size := minSide * elCfg.Scale
			posX := originX + (elCfg.X * labelW / 100.0)
			posY := originY + labelH - (elCfg.Y * labelH / 100.0) - (size * 0.8)

			pdf.SetFont(fontName, "B", size*2.5)
			pdf.SetTextColor(int(gray*255), int(gray*255), int(gray*255))
			pdf.SetXY(posX, posY)
			pdf.CellFormat(0, size, text, "", 0, "L", false, 0, "")
		}

		// If no ContentConfig provided, use the default "Master QR Puzzle" layout
		if cfg.ContentConfig == nil {
			// Default layout: Large QR1 left, Checksum center, Serial bottom, QR2+QR3 right
			qr1Scale := 0.85
			qr1Size := labelH * qr1Scale

			// QR1 (Large, left side)
			qr1Data := fmt.Sprintf("ECK1.COM/%s%s", encryptedCode, instanceSuffix)
			qr1Png, _ := qrcode.Encode(qr1Data, qrcode.Low, 256)
			imgName1 := fmt.Sprintf("qr1_%d", i)
			pdf.RegisterImageOptionsReader(imgName1, gofpdf.ImageOptions{ImageType: "PNG"}, bytes.NewReader(qr1Png))
			pdf.ImageOptions(imgName1, originX+2, originY+(labelH-qr1Size)/2, qr1Size, qr1Size, false, gofpdf.ImageOptions{ImageType: "PNG"}, 0, "")

			// Checksum (Large, center-right)
			csScale := 0.45
			csSize := labelH * csScale
			pdf.SetFont("Arial", "B", csSize)
			pdf.SetTextColor(0, 0, 0)
			pdf.SetXY(originX+qr1Size+8, originY+(labelH/2)-(csSize/4))
			pdf.CellFormat(0, csSize/2, field2, "", 0, "L", false, 0, "")

			// Get checksum text width for positioning QR2/QR3
			csWidth := pdf.GetStringWidth(field2)

			// Serial (Small, below checksum)
			sScale := 0.12
			sSize := labelH * sScale
			pdf.SetFont("Courier", "B", sSize)
			pdf.SetTextColor(77, 77, 77)
			pdf.SetXY(originX+qr1Size+8, originY+labelH*0.75)
			pdf.CellFormat(0, sSize/2, field1, "", 0, "L", false, 0, "")

			// QR2 & QR3 (Small, right side)
			sQrScale := 0.32
			sQrSize := labelH * sQrScale
			rightX := originX + qr1Size + csWidth + 16

			if rightX+sQrSize < originX+labelW {
				// QR2 (top right)
				qr2Data := fmt.Sprintf("ECK2.COM/%s%s", encryptedCode, instanceSuffix)
				qr2Png, _ := qrcode.Encode(qr2Data, qrcode.Low, 256)
				imgName2 := fmt.Sprintf("qr2_%d", i)
				pdf.RegisterImageOptionsReader(imgName2, gofpdf.ImageOptions{ImageType: "PNG"}, bytes.NewReader(qr2Png))
				pdf.ImageOptions(imgName2, rightX, originY+3, sQrSize, sQrSize, false, gofpdf.ImageOptions{ImageType: "PNG"}, 0, "")

				// QR3 (bottom right)
				qr3Data := fmt.Sprintf("ECK3.COM/%s%s", encryptedCode, instanceSuffix)
				qr3Png, _ := qrcode.Encode(qr3Data, qrcode.Low, 256)
				imgName3 := fmt.Sprintf("qr3_%d", i)
				pdf.RegisterImageOptionsReader(imgName3, gofpdf.ImageOptions{ImageType: "PNG"}, bytes.NewReader(qr3Png))
				pdf.ImageOptions(imgName3, rightX, originY+labelH-sQrSize-3, sQrSize, sQrSize, false, gofpdf.ImageOptions{ImageType: "PNG"}, 0, "")
			}
		} else {
			// Custom configuration from UI
			if cfg.ContentConfig.QR1 != nil {
				drawQR("ECK1.COM", cfg.ContentConfig.QR1)
			}
			if cfg.ContentConfig.QR2 != nil {
				drawQR("ECK2.COM", cfg.ContentConfig.QR2)
			}
			if cfg.ContentConfig.QR3 != nil {
				drawQR("ECK3.COM", cfg.ContentConfig.QR3)
			}
			if cfg.ContentConfig.Checksum != nil {
				drawText(field2, cfg.ContentConfig.Checksum, "Arial", 0)
			}
			if cfg.ContentConfig.Serial != nil {
				drawText(field1, cfg.ContentConfig.Serial, "Courier", 0.3)
			}
		}

		// Debug border (uncomment to see label boundaries)
		// pdf.SetDrawColor(200, 200, 200)
		// pdf.Rect(originX, originY, labelW, labelH, "D")
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
