package delivery

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/xelth-com/eckwmsgo/internal/config"
	"github.com/xelth-com/eckwmsgo/internal/database"
	"github.com/xelth-com/eckwmsgo/internal/delivery"
	"github.com/xelth-com/eckwmsgo/internal/models"
)

// Service handles delivery operations
type Service struct {
	db       *database.DB
	registry *delivery.Registry
	config   *config.Config
}

// NewService creates a new delivery service
func NewService(db *database.DB, cfg *config.Config) *Service {
	return &Service{
		db:       db,
		registry: delivery.GetGlobalRegistry(),
		config:   cfg,
	}
}

// RegisterProvider registers a new delivery provider
func (s *Service) RegisterProvider(provider delivery.ProviderInterface) error {
	return s.registry.Register(provider)
}

// CreateShipment creates a shipment for a stock picking
func (s *Service) CreateShipment(ctx context.Context, pickingID int64, providerCode string) error {
	// 1. Get the picking
	var picking models.StockPicking
	if err := s.db.First(&picking, pickingID).Error; err != nil {
		return fmt.Errorf("picking not found: %w", err)
	}

	// 2. Get or create delivery record
	var deliveryRecord models.StockPickingDelivery
	result := s.db.Where("picking_id = ?", pickingID).First(&deliveryRecord)

	if result.Error != nil {
		// Create new delivery record
		var carrier models.DeliveryCarrier
		if err := s.db.Where("provider_code = ? AND active = ?", providerCode, true).First(&carrier).Error; err != nil {
			return fmt.Errorf("carrier not found or inactive: %w", err)
		}

		deliveryRecord = models.StockPickingDelivery{
			PickingID: pickingID,
			CarrierID: &carrier.ID,
			Status:    models.DeliveryStatusPending,
		}

		if err := s.db.Create(&deliveryRecord).Error; err != nil {
			return fmt.Errorf("failed to create delivery record: %w", err)
		}
	}

	// 3. Update status to pending (will be processed by worker)
	deliveryRecord.Status = models.DeliveryStatusPending
	if err := s.db.Save(&deliveryRecord).Error; err != nil {
		return fmt.Errorf("failed to update delivery status: %w", err)
	}

	return nil
}

// ProcessPendingShipments processes all pending shipments
// This should be called by a background worker
func (s *Service) ProcessPendingShipments(ctx context.Context) error {
	var pendingDeliveries []models.StockPickingDelivery

	// Get all pending deliveries
	if err := s.db.
		Preload("Picking").
		Preload("Carrier").
		Where("status = ?", models.DeliveryStatusPending).
		Find(&pendingDeliveries).Error; err != nil {
		return fmt.Errorf("failed to fetch pending deliveries: %w", err)
	}

	for _, deliveryRecord := range pendingDeliveries {
		// Process each shipment
		if err := s.processShipment(ctx, &deliveryRecord); err != nil {
			// Log error but continue with next shipment
			fmt.Printf("Error processing shipment %d: %v\n", deliveryRecord.ID, err)
		}
	}

	return nil
}

// processShipment processes a single shipment
func (s *Service) processShipment(ctx context.Context, deliveryRecord *models.StockPickingDelivery) error {
	// Get carrier
	var carrier models.DeliveryCarrier
	if err := s.db.First(&carrier, deliveryRecord.CarrierID).Error; err != nil {
		return s.markShipmentError(deliveryRecord, fmt.Sprintf("carrier not found: %v", err))
	}

	// Get provider
	provider, err := s.registry.Get(carrier.ProviderCode)
	if err != nil {
		return s.markShipmentError(deliveryRecord, fmt.Sprintf("provider not found: %v", err))
	}

	// Build delivery request from picking
	req, err := s.buildDeliveryRequest(deliveryRecord.Picking)
	if err != nil {
		return s.markShipmentError(deliveryRecord, fmt.Sprintf("failed to build request: %v", err))
	}

	// Create shipment with provider
	resp, err := provider.CreateShipment(ctx, req)
	if err != nil {
		return s.markShipmentError(deliveryRecord, fmt.Sprintf("provider error: %v", err))
	}

	// Update delivery record with response
	now := time.Now()
	deliveryRecord.TrackingNumber = resp.TrackingNumber
	deliveryRecord.CarrierPrice = resp.Price
	deliveryRecord.Currency = resp.Currency
	deliveryRecord.Status = models.DeliveryStatusShipped
	deliveryRecord.ShippedAt = &now
	deliveryRecord.LabelURL = resp.LabelURL

	if len(resp.LabelPDF) > 0 {
		deliveryRecord.LabelData = resp.LabelPDF
	}

	if resp.RawResponse != nil {
		rawJSON, _ := json.Marshal(resp.RawResponse)
		deliveryRecord.RawResponse = string(rawJSON)
	}

	if err := s.db.Save(deliveryRecord).Error; err != nil {
		return fmt.Errorf("failed to save delivery record: %w", err)
	}

	// Create tracking entry
	s.createTrackingEntry(deliveryRecord.ID, "Shipment created", models.DeliveryStatusShipped)

	return nil
}

// markShipmentError marks a shipment as failed
func (s *Service) markShipmentError(deliveryRecord *models.StockPickingDelivery, errorMsg string) error {
	deliveryRecord.Status = models.DeliveryStatusError
	deliveryRecord.ErrorMessage = errorMsg

	if err := s.db.Save(deliveryRecord).Error; err != nil {
		return fmt.Errorf("failed to mark shipment as error: %w", err)
	}

	// Create tracking entry
	s.createTrackingEntry(deliveryRecord.ID, errorMsg, models.DeliveryStatusError)

	return fmt.Errorf(errorMsg)
}

// createTrackingEntry creates a tracking history entry
func (s *Service) createTrackingEntry(pickingDeliveryID int64, description, status string) {
	tracking := models.DeliveryTracking{
		PickingDeliveryID: pickingDeliveryID,
		Timestamp:         time.Now(),
		Status:            status,
		Description:       description,
	}

	s.db.Create(&tracking)
}

// buildDeliveryRequest builds a delivery request from a stock picking
func (s *Service) buildDeliveryRequest(picking *models.StockPicking) (*delivery.DeliveryRequest, error) {
	// Validate warehouse configuration
	if s.config.Warehouse.Street == "" || s.config.Warehouse.Zip == "" || s.config.Warehouse.City == "" {
		return nil, fmt.Errorf("warehouse address not configured - set WAREHOUSE_STREET, WAREHOUSE_ZIP, WAREHOUSE_CITY environment variables")
	}

	// Validate partner ID (customer)
	if picking.PartnerID == nil || *picking.PartnerID == 0 {
		return nil, fmt.Errorf("picking %s has no partner_id - cannot determine customer address", picking.Name)
	}

	// Fetch partner (customer) address from res_partner table
	var partner models.ResPartner
	if err := s.db.First(&partner, *picking.PartnerID).Error; err != nil {
		return nil, fmt.Errorf("partner %d not found in database - ensure Odoo partner sync is running: %w", *picking.PartnerID, err)
	}

	// Validate partner has required address fields
	if partner.Street == "" || partner.Zip == "" || partner.City == "" {
		return nil, fmt.Errorf("partner %d (%s) has incomplete address (missing street/zip/city)", partner.ID, partner.Name)
	}

	// Build delivery request
	req := &delivery.DeliveryRequest{
		OrderNumber: picking.Name,
		SenderAddress: delivery.Address{
			Name1:   s.config.Warehouse.Name,
			Street:  s.config.Warehouse.Street,
			Zip:     s.config.Warehouse.Zip,
			City:    s.config.Warehouse.City,
			Country: s.config.Warehouse.Country,
		},
		ReceiverAddress: delivery.Address{
			Name1:   partner.Name,
			Street:  partner.Street,
			Zip:     partner.Zip,
			City:    partner.City,
			Country: s.config.Warehouse.Country, // TODO: Map partner.CountryID to ISO code
			PhoneNumber: partner.Phone,
			Email:   partner.Email,
		},
		Parcels: []delivery.Package{
			{
				Count:       1,
				Weight:      10.0, // TODO: Calculate from move lines
				Description: "Order " + picking.Name,
				Value:       0, // TODO: Calculate from order
				Currency:    "EUR",
			},
		},
	}

	return req, nil
}

// GetDeliveryStatus retrieves the delivery status for a picking
func (s *Service) GetDeliveryStatus(pickingID int64) (*models.StockPickingDelivery, error) {
	var deliveryRecord models.StockPickingDelivery

	if err := s.db.
		Preload("Carrier").
		Preload("Picking").
		Where("picking_id = ?", pickingID).
		First(&deliveryRecord).Error; err != nil {
		return nil, fmt.Errorf("delivery record not found: %w", err)
	}

	return &deliveryRecord, nil
}

// GetTrackingHistory retrieves tracking history for a delivery
func (s *Service) GetTrackingHistory(pickingDeliveryID int64) ([]models.DeliveryTracking, error) {
	var tracking []models.DeliveryTracking

	if err := s.db.
		Where("picking_delivery_id = ?", pickingDeliveryID).
		Order("timestamp DESC").
		Find(&tracking).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch tracking history: %w", err)
	}

	return tracking, nil
}

// CancelShipment cancels a shipment
func (s *Service) CancelShipment(ctx context.Context, pickingID int64) error {
	// Get delivery record
	deliveryRecord, err := s.GetDeliveryStatus(pickingID)
	if err != nil {
		return err
	}

	// Get carrier
	var carrier models.DeliveryCarrier
	if err := s.db.First(&carrier, deliveryRecord.CarrierID).Error; err != nil {
		return fmt.Errorf("carrier not found: %w", err)
	}

	// Get provider
	provider, err := s.registry.Get(carrier.ProviderCode)
	if err != nil {
		return fmt.Errorf("provider not found: %w", err)
	}

	// Cancel with provider
	if err := provider.CancelShipment(ctx, deliveryRecord.TrackingNumber); err != nil {
		return fmt.Errorf("provider cancellation failed: %w", err)
	}

	// Update status
	deliveryRecord.Status = models.DeliveryStatusCancelled
	if err := s.db.Save(deliveryRecord).Error; err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	// Create tracking entry
	s.createTrackingEntry(deliveryRecord.ID, "Shipment cancelled", models.DeliveryStatusCancelled)

	return nil
}

// ListShipments returns all shipments with optional state filter
func (s *Service) ListShipments(state string, limit int) ([]models.StockPickingDelivery, error) {
	var shipments []models.StockPickingDelivery

	query := s.db.Preload("Picking").Preload("Carrier")
	if state != "" {
		query = query.Where("status = ?", state)
	}

	if err := query.Order("created_at DESC").Limit(limit).Find(&shipments).Error; err != nil {
		return nil, err
	}

	return shipments, nil
}

// GetShipment returns a shipment by ID
func (s *Service) GetShipment(id int64) (*models.StockPickingDelivery, error) {
	var shipment models.StockPickingDelivery
	if err := s.db.Preload("Picking").Preload("Carrier").First(& shipment, id).Error; err != nil {
		return nil, err
	}
	return &shipment, nil
}

// ListCarriers returns all delivery carriers
func (s *Service) ListCarriers() ([]models.DeliveryCarrier, error) {
	var carriers []models.DeliveryCarrier
	if err := s.db.Order("name ASC").Find(&carriers).Error; err != nil {
		return nil, err
	}
	return carriers, nil
}

// CreateCarrier creates a new delivery carrier
func (s *Service) CreateCarrier(name, providerCode, configJSON string) (*models.DeliveryCarrier, error) {
	carrier := models.DeliveryCarrier{
		Name:         name,
		ProviderCode: providerCode,
		ConfigJSON:   configJSON,
		Active:       true,
	}

	if err := s.db.Create(&carrier).Error; err != nil {
		return nil, err
	}

	return &carrier, nil
}

// GetCarrier returns a carrier by ID
func (s *Service) GetCarrier(id int64) (*models.DeliveryCarrier, error) {
	var carrier models.DeliveryCarrier
	if err := s.db.First(&carrier, id).Error; err != nil {
		return nil, err
	}
	return &carrier, nil
}

// ToggleCarrier toggles carrier active status
func (s *Service) ToggleCarrier(id int64) error {
	carrier, err := s.GetCarrier(id)
	if err != nil {
		return err
	}

	carrier.Active = !carrier.Active
	if err := s.db.Save(carrier).Error; err != nil {
		return err
	}

	return nil
}
