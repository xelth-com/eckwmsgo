package dhl

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/xelth-com/eckwmsgo/internal/delivery"
)

// Config holds configuration for the DHL provider
type Config struct {
	ScriptPath string // Path to the Node.js script (fetch-dhl-orders.js)
	NodePath   string // Path to node executable (defaults to "node")
	Username   string // DHL login username (email)
	Password   string // DHL login password
	URL        string // DHL URL (defaults to https://geschaeftskunden.dhl.de)
	Headless   bool   // Run browser in headless mode
	Timeout    int    // Timeout in seconds (default: 300)
}

// Provider implements the delivery.ProviderInterface for DHL
type Provider struct {
	config Config
}

// ScrapedShipment represents a shipment scraped from DHL portal
type ScrapedShipment struct {
	TrackingNumber     string `json:"tracking_number"`
	Reference          string `json:"reference"`
	InternationalNum   string `json:"international_number"`
	BillingNumber      string `json:"billing_number"`
	RecipientName      string `json:"recipient_name"`
	RecipientStreet    string `json:"recipient_street"`
	RecipientZip       string `json:"recipient_zip"`
	RecipientCity      string `json:"recipient_city"`
	RecipientCountry   string `json:"recipient_country"`
	Status             string `json:"status"`
	StatusDate         string `json:"status_date"`
	Note               string `json:"note"`
	DeliveredToName    string `json:"delivered_to_name"`
	DeliveredToStreet  string `json:"delivered_to_street"`
	DeliveredToZip     string `json:"delivered_to_zip"`
	DeliveredToCity    string `json:"delivered_to_city"`
	DeliveredToCountry string `json:"delivered_to_country"`
	Product            string `json:"product"`
	Services           string `json:"services"`
}

// NewProvider creates a new DHL delivery provider
func NewProvider(config Config) (*Provider, error) {
	// Set defaults
	if config.NodePath == "" {
		config.NodePath = "node"
	}
	if config.URL == "" {
		config.URL = "https://geschaeftskunden.dhl.de"
	}
	if config.Timeout == 0 {
		config.Timeout = 300 // 5 minutes default
	}

	// Validate required fields
	if config.ScriptPath == "" {
		return nil, fmt.Errorf("script path is required")
	}
	if config.Username == "" {
		return nil, fmt.Errorf("username is required")
	}
	if config.Password == "" {
		return nil, fmt.Errorf("password is required")
	}

	return &Provider{
		config: config,
	}, nil
}

// Code returns the provider code
func (p *Provider) Code() string {
	return "dhl"
}

// Name returns the provider name
func (p *Provider) Name() string {
	return "DHL Geschäftskunden"
}

// CreateShipment creates a new shipment via DHL
func (p *Provider) CreateShipment(ctx context.Context, req *delivery.DeliveryRequest) (*delivery.DeliveryResponse, error) {
	// 1. Prepare data for the script
	data := map[string]interface{}{
		"delivery_name":   req.ReceiverAddress.Name1,
		"delivery_street": req.ReceiverAddress.Street,
		"delivery_zip":    req.ReceiverAddress.Zip,
		"delivery_city":   req.ReceiverAddress.City,
		"weight":          1.0, // Default weight
		"reference":       req.RefNumber,
	}

	// Add house number if available
	if req.ReceiverAddress.HouseNumber != "" {
		data["delivery_house_number"] = req.ReceiverAddress.HouseNumber
	}

	// Use actual weight if provided
	if len(req.Parcels) > 0 && req.Parcels[0].Weight > 0 {
		data["weight"] = req.Parcels[0].Weight
	}

	// 2. Create temp file with order data
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "dhl-order-*.json")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(jsonData); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("failed to write temp file: %w", err)
	}
	tmpFile.Close()

	// 3. Execute script
	scriptDir := filepath.Dir(p.config.ScriptPath)
	createScript := filepath.Join(scriptDir, "create-dhl-order.js")

	// Check if script exists
	if _, err := os.Stat(createScript); os.IsNotExist(err) {
		return nil, fmt.Errorf("create script not found at %s", createScript)
	}

	args := []string{
		createScript,
		"--data=" + tmpFile.Name(),
		"--json-output",
	}

	if p.config.Headless {
		args = append(args, "--headless")
	}

	// Create context with timeout
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(p.config.Timeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(execCtx, p.config.NodePath, args...)
	cmd.Dir = scriptDir

	// Pass environment variables for login
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("DHL_USERNAME=%s", p.config.Username),
		fmt.Sprintf("DHL_PASSWORD=%s", p.config.Password),
		fmt.Sprintf("DHL_URL=%s", p.config.URL),
	)

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("script failed: %s, stderr: %s", err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("script execution failed: %w", err)
	}

	// 4. Parse response
	var result struct {
		Success        bool   `json:"success"`
		TrackingNumber string `json:"trackingNumber"`
		OrderNumber    string `json:"orderNumber"`
		Error          string `json:"error"`
		Message        string `json:"message"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse script response: %w, output: %s", err, string(output))
	}

	if !result.Success {
		return nil, fmt.Errorf("DHL creation failed: %s", result.Error)
	}

	return &delivery.DeliveryResponse{
		TrackingNumber: result.TrackingNumber,
		CreatedAt:      time.Now(),
		RawResponse:    map[string]interface{}{"provider": "dhl", "message": result.Message},
	}, nil
}

// CancelShipment cancels an existing DHL shipment
func (p *Provider) CancelShipment(ctx context.Context, trackingNumber string) error {
	// TODO: Implement DHL shipment cancellation
	return fmt.Errorf("DHL CancelShipment not yet implemented")
}

// GetStatus retrieves the current status of a DHL shipment
func (p *Provider) GetStatus(ctx context.Context, trackingNumber string) (*delivery.TrackingStatus, error) {
	// TODO: Implement individual tracking lookup
	return nil, fmt.Errorf("DHL GetStatus not yet implemented")
}

// ValidateAddress validates an address for DHL
func (p *Provider) ValidateAddress(ctx context.Context, addr *delivery.Address) error {
	// Basic validation for DHL
	if addr.Zip == "" {
		return fmt.Errorf("postal code is required for DHL")
	}
	if addr.City == "" {
		return fmt.Errorf("city is required for DHL")
	}
	if addr.Street == "" {
		return fmt.Errorf("street is required for DHL")
	}
	return nil
}

// FetchRecentShipments fetches recent shipments from DHL portal
// Falls back to cached CSV if scraper fails
func (p *Provider) FetchRecentShipments(ctx context.Context, days int) ([]ScrapedShipment, error) {
	// Build path to fetch script
	scriptDir := filepath.Dir(p.config.ScriptPath)
	cachedCSV := filepath.Join(scriptDir, "..", "..", "data", "dhl-shipments.csv")
	fetchScriptPath := filepath.Join(scriptDir, "fetch-dhl-orders.js")

	// Check if script exists
	if _, err := os.Stat(fetchScriptPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("fetch script not found at %s", fetchScriptPath)
	}

	// Build command arguments
	args := []string{
		fetchScriptPath,
		"--username", p.config.Username,
		"--password", p.config.Password,
		"--output", "json",
	}

	if days > 0 {
		args = append(args, "--days", fmt.Sprintf("%d", days))
	}

	if p.config.Headless {
		args = append(args, "--headless")
	}

	// Create context with timeout
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(p.config.Timeout)*time.Second)
	defer cancel()

	// Run the script
	cmd := exec.CommandContext(execCtx, p.config.NodePath, args...)
	cmd.Dir = scriptDir

	// Set environment variables
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("DHL_USERNAME=%s", p.config.Username),
		fmt.Sprintf("DHL_PASSWORD=%s", p.config.Password),
		fmt.Sprintf("DHL_URL=%s", p.config.URL),
	)

	output, err := cmd.Output()
	if err != nil {
		// Scraper failed - try to use cached CSV as fallback
		fmt.Printf("[DHL] Scraper failed, trying cached CSV at %s\n", cachedCSV)
		return p.readCachedCSV(cachedCSV)
	}

	// Parse JSON output
	var shipments []ScrapedShipment
	if err := json.Unmarshal(output, &shipments); err != nil {
		return nil, fmt.Errorf("failed to parse script output: %w, output: %s", err, string(output))
	}

	return shipments, nil
}

// readCachedCSV reads shipments from a cached CSV file
func (p *Provider) readCachedCSV(csvPath string) ([]ScrapedShipment, error) {
	content, err := os.ReadFile(csvPath)
	if err != nil {
		return nil, fmt.Errorf("no cached CSV available: %w", err)
	}

	// Parse CSV (semicolon-separated, German headers)
	lines := strings.Split(strings.ReplaceAll(string(content), "\r", ""), "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("cached CSV is empty")
	}

	headers := strings.Split(lines[0], ";")
	headerMap := map[string]string{
		"Sendungsnummer":                       "tracking_number",
		"Sendungsreferenz":                     "reference",
		"internationale Sendungsnummer":       "international_number",
		"Abrechnungsnummer":                   "billing_number",
		"Empfängername":                        "recipient_name",
		"Empfängerstraße (inkl. Hausnummer)": "recipient_street",
		"Empfänger-PLZ":                        "recipient_zip",
		"Empfänger-Ort":                        "recipient_city",
		"Empfänger-Land":                       "recipient_country",
		"Status":                               "status",
		"Datum Status":                         "status_date",
		"Hinweis":                              "note",
		"Zugestellt an - Name":                 "delivered_to_name",
		"Zugestellt an - Straße (inkl. Hausnummer)": "delivered_to_street",
		"Zugestellt an - PLZ":                  "delivered_to_zip",
		"Zugestellt an - Ort":                  "delivered_to_city",
		"Zugestellt an - Land":                 "delivered_to_country",
		"Produkt":                              "product",
		"Services":                             "services",
	}

	var shipments []ScrapedShipment
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "" {
			continue
		}
		values := strings.Split(lines[i], ";")
		if len(values) < len(headers) {
			continue
		}

		data := make(map[string]string)
		for j, header := range headers {
			key := headerMap[header]
			if key == "" {
				key = header
			}
			data[key] = strings.TrimSpace(values[j])
		}

		if data["tracking_number"] == "" {
			continue
		}

		shipments = append(shipments, ScrapedShipment{
			TrackingNumber:     data["tracking_number"],
			Reference:          data["reference"],
			InternationalNum:   data["international_number"],
			BillingNumber:      data["billing_number"],
			RecipientName:      data["recipient_name"],
			RecipientStreet:    data["recipient_street"],
			RecipientZip:       data["recipient_zip"],
			RecipientCity:      data["recipient_city"],
			RecipientCountry:   data["recipient_country"],
			Status:             data["status"],
			StatusDate:         data["status_date"],
			Note:               data["note"],
			DeliveredToName:    data["delivered_to_name"],
			DeliveredToStreet:  data["delivered_to_street"],
			DeliveredToZip:     data["delivered_to_zip"],
			DeliveredToCity:    data["delivered_to_city"],
			DeliveredToCountry: data["delivered_to_country"],
			Product:            data["product"],
			Services:           data["services"],
		})
	}

	fmt.Printf("[DHL] Loaded %d shipments from cached CSV\n", len(shipments))
	return shipments, nil
}

// MapStatus converts DHL status to standard status
func MapStatus(dhlStatus string) string {
	switch dhlStatus {
	case "Zugestellt":
		return "delivered"
	case "Transport", "in Zustellung":
		return "shipped"
	case "Abholung erfolgreich":
		return "shipped"
	case "Storniert":
		return "cancelled"
	case "Auftragsdaten übermittelt":
		return "pending"
	default:
		return "pending"
	}
}
