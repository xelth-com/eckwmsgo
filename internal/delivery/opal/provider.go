package opal

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

// Config holds configuration for the OPAL provider
type Config struct {
	ScriptPath string // Path to the Node.js script (create-opal-order.js)
	NodePath   string // Path to node executable (defaults to "node")
	Username   string // OPAL login username
	Password   string // OPAL login password
	URL        string // OPAL URL (defaults to https://opal-kurier.de)
	Headless   bool   // Run browser in headless mode
	Timeout    int    // Timeout in seconds (default: 300)
}

// Provider implements the delivery.ProviderInterface for OPAL Kurier
type Provider struct {
	config Config
}

// NewProvider creates a new OPAL delivery provider
func NewProvider(config Config) (*Provider, error) {
	// Set defaults
	if config.NodePath == "" {
		config.NodePath = "node"
	}
	if config.URL == "" {
		config.URL = "https://opal-kurier.de"
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

	// Check if script exists
	if _, err := os.Stat(config.ScriptPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("script not found at %s", config.ScriptPath)
	}

	return &Provider{
		config: config,
	}, nil
}

// Code returns the provider code
func (p *Provider) Code() string {
	return "opal"
}

// Name returns the provider name
func (p *Provider) Name() string {
	return "OPAL Kurier"
}

// CreateShipment creates a new shipment via OPAL
func (p *Provider) CreateShipment(ctx context.Context, req *delivery.DeliveryRequest) (*delivery.DeliveryResponse, error) {
	// Convert delivery request to OPAL format
	opalData := p.convertToOpalFormat(req)

	// Marshal to JSON
	jsonData, err := json.Marshal(opalData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal order data: %w", err)
	}

	// Create temporary file for order data
	tmpFile, err := os.CreateTemp("", "opal-order-*.json")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write JSON data to temp file
	if _, err := tmpFile.Write(jsonData); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("failed to write order data: %w", err)
	}
	tmpFile.Close()

	// Prepare command
	args := []string{
		p.config.ScriptPath,
		"--data=" + tmpFile.Name(),
		"--json-output", // Tell script to output JSON
	}

	if p.config.Headless {
		args = append(args, "--headless")
	}

	// Create command with timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(p.config.Timeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, p.config.NodePath, args...)

	// Set environment variables
	cmd.Env = append(os.Environ(),
		"OPAL_URL="+p.config.URL,
		"OPAL_USERNAME="+p.config.Username,
		"OPAL_PASSWORD="+p.config.Password,
	)

	// Execute command and capture output (STDOUT only)
	output, err := cmd.Output()
	if err != nil {
		// If the command failed, try to get stderr from the error
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("script execution failed: %w\nStderr: %s", err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("script execution failed: %w", err)
	}

	// Parse JSON response from script
	var result struct {
		Success        bool   `json:"success"`
		TrackingNumber string `json:"trackingNumber"`
		OrderNumber    string `json:"orderNumber"`
		Error          string `json:"error"`
		Message        string `json:"message"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		// If JSON parsing fails, return the raw output
		return nil, fmt.Errorf("failed to parse script output: %w\nOutput: %s", err, string(output))
	}

	if !result.Success {
		return nil, fmt.Errorf("OPAL order creation failed: %s", result.Error)
	}

	// Build response
	response := &delivery.DeliveryResponse{
		TrackingNumber: result.TrackingNumber,
		CreatedAt:      time.Now(),
		RawResponse: map[string]interface{}{
			"orderNumber": result.OrderNumber,
			"message":     result.Message,
		},
	}

	return response, nil
}

// ScrapedOrder represents the JSON structure returned by the Node.js detail page scraper
type ScrapedOrder struct {
	TrackingNumber string `json:"tracking_number"` // OCU number (e.g., "OCU-998-511590")
	HwbNumber      string `json:"hwb_number"`      // GO Barcode (e.g., "041940529157")
	ProductType    string `json:"product_type"`    // Overnight, X-Change, etc.
	Reference      string `json:"reference"`       // Reference number
	CreatedAt      string `json:"created_at"`      // When order was created
	CreatedBy      string `json:"created_by"`      // Who created the order

	// Pickup info
	PickupName     string `json:"pickup_name"`
	PickupName2    string `json:"pickup_name2"`
	PickupContact  string `json:"pickup_contact"`
	PickupPhone    string `json:"pickup_phone"`
	PickupEmail    string `json:"pickup_email"`
	PickupStreet   string `json:"pickup_street"`
	PickupCity     string `json:"pickup_city"`
	PickupZip      string `json:"pickup_zip"`
	PickupCountry  string `json:"pickup_country"`
	PickupNote     string `json:"pickup_note"`
	PickupDate     string `json:"pickup_date"`
	PickupTimeFrom string `json:"pickup_time_from"`
	PickupTimeTo   string `json:"pickup_time_to"`
	PickupVehicle  string `json:"pickup_vehicle"`

	// Delivery info
	DeliveryName     string `json:"delivery_name"`
	DeliveryName2    string `json:"delivery_name2"`
	DeliveryContact  string `json:"delivery_contact"`
	DeliveryPhone    string `json:"delivery_phone"`
	DeliveryEmail    string `json:"delivery_email"`
	DeliveryStreet   string `json:"delivery_street"`
	DeliveryCity     string `json:"delivery_city"`
	DeliveryZip      string `json:"delivery_zip"`
	DeliveryCountry  string `json:"delivery_country"`
	DeliveryNote     string `json:"delivery_note"`
	DeliveryDate     string `json:"delivery_date"`
	DeliveryTimeFrom string `json:"delivery_time_from"`
	DeliveryTimeTo   string `json:"delivery_time_to"`

	// Package info
	PackageCount *int     `json:"package_count"`
	Weight       *float64 `json:"weight"`
	Value        *float64 `json:"value"`
	Description  string   `json:"description"`
	Dimensions   string   `json:"dimensions"`

	// Status
	Status     string `json:"status"`      // Zugestellt, Abgeholt, Storniert, AKTIV
	StatusDate string `json:"status_date"` // When status changed
	StatusTime string `json:"status_time"` // Time of status change
	Receiver   string `json:"receiver"`    // Who received it
}

// FetchRecentOrders executes the Node.js scraper to get recent orders from OPAL
func (p *Provider) FetchRecentOrders(ctx context.Context) ([]ScrapedOrder, error) {
	// Build path to fetch script (same directory as create script)
	scriptDir := filepath.Dir(p.config.ScriptPath)
	fetchScriptPath := filepath.Join(scriptDir, "fetch-opal-orders.js")

	// Check if fetch script exists
	if _, err := os.Stat(fetchScriptPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("fetch script not found at %s", fetchScriptPath)
	}

	// Prepare command
	args := []string{fetchScriptPath, "--json-output"}

	if p.config.Headless {
		args = append(args, "--headless")
	}

	// Create command with timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(p.config.Timeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, p.config.NodePath, args...)

	// Set environment variables
	cmd.Env = append(os.Environ(),
		"OPAL_URL="+p.config.URL,
		"OPAL_USERNAME="+p.config.Username,
		"OPAL_PASSWORD="+p.config.Password,
	)

	// Execute command and capture output (STDOUT only)
	output, err := cmd.Output()
	if err != nil {
		// If the command failed, try to get stderr from the error
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("scraper execution failed: %w\nStderr: %s", err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("scraper execution failed: %w", err)
	}

	// Parse JSON response from script
	var result struct {
		Success bool           `json:"success"`
		Orders  []ScrapedOrder `json:"orders"`
		Error   string         `json:"error"`
		Count   int            `json:"count"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse scraper output: %w\nOutput: %s", err, string(output))
	}

	if !result.Success {
		return nil, fmt.Errorf("scraper returned error: %s", result.Error)
	}

	return result.Orders, nil
}

// CancelShipment cancels an OPAL shipment
func (p *Provider) CancelShipment(ctx context.Context, trackingNumber string) error {
	// OPAL cancellation would require another script or API call
	// For now, return not implemented
	return fmt.Errorf("cancel shipment not implemented for OPAL provider")
}

// GetStatus retrieves the status of an OPAL shipment
func (p *Provider) GetStatus(ctx context.Context, trackingNumber string) (*delivery.TrackingStatus, error) {
	// OPAL status tracking would require web scraping or API
	// For now, return not implemented
	return nil, fmt.Errorf("get status not implemented for OPAL provider")
}

// ValidateAddress validates an address for OPAL
func (p *Provider) ValidateAddress(ctx context.Context, addr *delivery.Address) error {
	// Basic validation - OPAL requires at least name, street, zip, and city
	if addr.Name1 == "" {
		return fmt.Errorf("name is required")
	}
	if addr.Street == "" {
		return fmt.Errorf("street is required")
	}
	if addr.Zip == "" {
		return fmt.Errorf("zip code is required")
	}
	if addr.City == "" {
		return fmt.Errorf("city is required")
	}

	return nil
}

// convertToOpalFormat converts a DeliveryRequest to OPAL's expected format
func (p *Provider) convertToOpalFormat(req *delivery.DeliveryRequest) map[string]interface{} {
	data := make(map[string]interface{})

	// Pickup/sender information
	if req.SenderAddress.Name1 != "" {
		data["pickupName1"] = req.SenderAddress.Name1
	}
	if req.SenderAddress.Name2 != "" {
		data["pickupName2"] = req.SenderAddress.Name2
	}
	if req.SenderAddress.Contact != "" {
		data["pickupContact"] = req.SenderAddress.Contact
	}
	if req.SenderAddress.Street != "" {
		data["pickupStreet"] = req.SenderAddress.Street
	}
	if req.SenderAddress.HouseNumber != "" {
		data["pickupHouseNumber"] = req.SenderAddress.HouseNumber
	}
	if req.SenderAddress.Country != "" {
		data["pickupCountry"] = req.SenderAddress.Country
	}
	if req.SenderAddress.Zip != "" {
		data["pickupZip"] = req.SenderAddress.Zip
	}
	if req.SenderAddress.City != "" {
		data["pickupCity"] = req.SenderAddress.City
	}
	if req.SenderAddress.PhoneCountry != "" {
		data["pickupPhoneCountry"] = req.SenderAddress.PhoneCountry
	}
	if req.SenderAddress.PhoneArea != "" {
		data["pickupPhoneArea"] = req.SenderAddress.PhoneArea
	}
	if req.SenderAddress.PhoneNumber != "" {
		data["pickupPhoneNumber"] = req.SenderAddress.PhoneNumber
	}
	if req.SenderAddress.Email != "" {
		data["pickupEmail"] = req.SenderAddress.Email
	}
	if req.SenderAddress.Notes != "" {
		data["pickupHinweis"] = req.SenderAddress.Notes
	}

	// Delivery/receiver information (required)
	data["deliveryName1"] = req.ReceiverAddress.Name1
	if req.ReceiverAddress.Name2 != "" {
		data["deliveryName2"] = req.ReceiverAddress.Name2
	}
	if req.ReceiverAddress.Contact != "" {
		data["deliveryContact"] = req.ReceiverAddress.Contact
	}
	data["deliveryStreet"] = req.ReceiverAddress.Street
	if req.ReceiverAddress.HouseNumber != "" {
		data["deliveryHouseNumber"] = req.ReceiverAddress.HouseNumber
	}
	if req.ReceiverAddress.Country != "" {
		data["deliveryCountry"] = req.ReceiverAddress.Country
	}
	data["deliveryZip"] = req.ReceiverAddress.Zip
	data["deliveryCity"] = req.ReceiverAddress.City
	if req.ReceiverAddress.PhoneCountry != "" {
		data["deliveryPhoneCountry"] = req.ReceiverAddress.PhoneCountry
	}
	if req.ReceiverAddress.PhoneArea != "" {
		data["deliveryPhoneArea"] = req.ReceiverAddress.PhoneArea
	}
	if req.ReceiverAddress.PhoneNumber != "" {
		data["deliveryPhoneNumber"] = req.ReceiverAddress.PhoneNumber
	}
	if req.ReceiverAddress.Email != "" {
		data["deliveryEmail"] = req.ReceiverAddress.Email
	}
	if req.ReceiverAddress.Notes != "" {
		data["deliveryHinweis"] = req.ReceiverAddress.Notes
	}

	// Time windows
	if req.PickupWindow != nil {
		if req.PickupWindow.Date != "" {
			data["pickupDate"] = req.PickupWindow.Date
		}
		if req.PickupWindow.TimeFrom != "" {
			data["pickupTimeFrom"] = req.PickupWindow.TimeFrom
		}
		if req.PickupWindow.TimeTo != "" {
			data["pickupTimeTo"] = req.PickupWindow.TimeTo
		}
	}

	if req.DeliveryWindow != nil {
		if req.DeliveryWindow.Date != "" {
			data["deliveryDate"] = req.DeliveryWindow.Date
		}
		if req.DeliveryWindow.TimeFrom != "" {
			data["deliveryTimeFrom"] = req.DeliveryWindow.TimeFrom
		}
		if req.DeliveryWindow.TimeTo != "" {
			data["deliveryTimeTo"] = req.DeliveryWindow.TimeTo
		}
	}

	// Package information
	if len(req.Parcels) > 0 {
		pkg := req.Parcels[0] // OPAL uses single package fields
		if pkg.Count > 0 {
			data["packageCount"] = pkg.Count
		}
		if pkg.Weight > 0 {
			data["packageWeight"] = pkg.Weight
		}
		if pkg.Description != "" {
			data["packageDescription"] = pkg.Description
		}
		if pkg.Value > 0 {
			data["shipmentValue"] = pkg.Value
		}
		if pkg.Currency != "" {
			data["shipmentValueCurrency"] = pkg.Currency
		}
	}

	// Additional fields
	if req.RefNumber != "" {
		data["refNumber"] = req.RefNumber
	}
	if req.Notes != "" {
		data["notes"] = req.Notes
	}
	if req.OrderType != "" {
		data["orderType"] = req.OrderType
	}
	if req.VehicleType != "" {
		data["vehicleType"] = req.VehicleType
	}

	return data
}
