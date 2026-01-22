package dhl

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
// Note: For DHL, we might use CSV upload instead of individual order creation
func (p *Provider) CreateShipment(ctx context.Context, req *delivery.DeliveryRequest) (*delivery.DeliveryResponse, error) {
	// TODO: Implement DHL shipment creation via Paket & Waren
	return nil, fmt.Errorf("DHL CreateShipment not yet implemented")
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
func (p *Provider) FetchRecentShipments(ctx context.Context, days int) ([]ScrapedShipment, error) {
	// Build path to fetch script
	scriptDir := filepath.Dir(p.config.ScriptPath)
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
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("script failed: %s, stderr: %s", err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("failed to run script: %w", err)
	}

	// Parse JSON output
	var shipments []ScrapedShipment
	if err := json.Unmarshal(output, &shipments); err != nil {
		return nil, fmt.Errorf("failed to parse script output: %w, output: %s", err, string(output))
	}

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
