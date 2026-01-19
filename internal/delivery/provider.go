package delivery

import (
	"context"
	"time"
)

// Address represents a physical address for pickup or delivery
type Address struct {
	Name1        string `json:"name1"`        // Company or person name (line 1)
	Name2        string `json:"name2"`        // Additional name info (line 2)
	Contact      string `json:"contact"`      // Contact person
	Street       string `json:"street"`       // Street name
	HouseNumber  string `json:"houseNumber"`  // House number
	Country      string `json:"country"`      // Country code (e.g., "DE")
	Zip          string `json:"zip"`          // Postal code
	City         string `json:"city"`         // City name
	PhoneCountry string `json:"phoneCountry"` // Phone country code
	PhoneArea    string `json:"phoneArea"`    // Phone area code
	PhoneNumber  string `json:"phoneNumber"`  // Phone number
	Email        string `json:"email"`        // Email address
	Notes        string `json:"notes"`        // Additional notes/instructions
}

// Package represents a single package/parcel in a shipment
type Package struct {
	Count       int     `json:"count"`       // Number of packages
	Weight      float64 `json:"weight"`      // Weight in kg
	Description string  `json:"description"` // Content description
	Value       float64 `json:"value"`       // Declared value
	Currency    string  `json:"currency"`    // Currency code (e.g., "EUR")
}

// TimeWindow represents a time window for pickup or delivery
type TimeWindow struct {
	Date     string `json:"date"`     // Date in format DD.MM.YYYY
	TimeFrom string `json:"timeFrom"` // Start time HH:MM
	TimeTo   string `json:"timeTo"`   // End time HH:MM
}

// DeliveryRequest contains all data needed to create a shipment
type DeliveryRequest struct {
	OrderNumber     string      `json:"orderNumber"`
	SenderAddress   Address     `json:"senderAddress"`
	ReceiverAddress Address     `json:"receiverAddress"`
	Parcels         []Package   `json:"parcels"`
	PickupWindow    *TimeWindow `json:"pickupWindow,omitempty"`
	DeliveryWindow  *TimeWindow `json:"deliveryWindow,omitempty"`
	RefNumber       string      `json:"refNumber"`       // Customer reference number
	Notes           string      `json:"notes"`           // Additional shipment notes
	OrderType       string      `json:"orderType"`       // Order type (provider-specific)
	VehicleType     string      `json:"vehicleType"`     // Vehicle type (provider-specific)
}

// DeliveryResponse contains the result from the delivery provider
type DeliveryResponse struct {
	TrackingNumber string                 `json:"trackingNumber"`
	LabelPDF       []byte                 `json:"labelPDF,omitempty"` // PDF label data
	LabelURL       string                 `json:"labelURL,omitempty"` // URL to download label
	Price          float64                `json:"price"`
	Currency       string                 `json:"currency"`
	RawResponse    map[string]interface{} `json:"rawResponse,omitempty"` // Original provider response
	CreatedAt      time.Time              `json:"createdAt"`
}

// TrackingStatus represents the current status of a shipment
type TrackingStatus struct {
	TrackingNumber string                 `json:"trackingNumber"`
	Status         string                 `json:"status"`         // Current status
	StatusCode     string                 `json:"statusCode"`     // Provider-specific status code
	Location       string                 `json:"location"`       // Current location
	UpdatedAt      time.Time              `json:"updatedAt"`      // Last update time
	Events         []TrackingEvent        `json:"events"`         // History of tracking events
	RawResponse    map[string]interface{} `json:"rawResponse,omitempty"`
}

// TrackingEvent represents a single tracking event in the shipment history
type TrackingEvent struct {
	Timestamp   time.Time `json:"timestamp"`
	Status      string    `json:"status"`
	StatusCode  string    `json:"statusCode"`
	Location    string    `json:"location"`
	Description string    `json:"description"`
}

// ProviderInterface defines the contract for all delivery providers
// This is the "Odoo Way" - a clean abstraction that allows plugging in different carriers
type ProviderInterface interface {
	// Code returns the unique code for this provider (e.g., "opal", "dhl", "ups")
	Code() string

	// Name returns the human-readable name of the provider
	Name() string

	// CreateShipment creates a new shipment and returns tracking information
	CreateShipment(ctx context.Context, req *DeliveryRequest) (*DeliveryResponse, error)

	// CancelShipment cancels an existing shipment
	CancelShipment(ctx context.Context, trackingNumber string) error

	// GetStatus retrieves the current status of a shipment
	GetStatus(ctx context.Context, trackingNumber string) (*TrackingStatus, error)

	// ValidateAddress validates an address for this carrier
	// Returns nil if valid, error if invalid
	ValidateAddress(ctx context.Context, addr *Address) error
}
