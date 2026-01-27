package delivery

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/xelth-com/eckwmsgo/internal/config"
	"github.com/xelth-com/eckwmsgo/internal/database"
	"github.com/xelth-com/eckwmsgo/internal/delivery"
	"github.com/xelth-com/eckwmsgo/internal/delivery/dhl"
	"github.com/xelth-com/eckwmsgo/internal/delivery/opal"
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
			PickingID: &pickingID,
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
			Name1:       partner.Name,
			Street:      string(partner.Street),
			Zip:         string(partner.Zip),
			City:        string(partner.City),
			Country:     s.config.Warehouse.Country, // TODO: Map partner.CountryID to ISO code
			PhoneNumber: string(partner.Phone),
			Email:       string(partner.Email),
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

	// Sort by last_activity_at (most recent first), fall back to created_at
	if err := query.Order("COALESCE(last_activity_at, created_at) DESC").Limit(limit).Find(&shipments).Error; err != nil {
		return nil, err
	}

	return shipments, nil
}

// GetShipment returns a shipment by ID
func (s *Service) GetShipment(id int64) (*models.StockPickingDelivery, error) {
	var shipment models.StockPickingDelivery
	if err := s.db.Preload("Picking").Preload("Carrier").First(&shipment, id).Error; err != nil {
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

// ImportOpalOrders fetches orders from OPAL and updates the database
func (s *Service) ImportOpalOrders(ctx context.Context) error {
	log := func(format string, args ...interface{}) {
		fmt.Printf("[OPAL Import] "+format+"\n", args...)
	}

	// Create sync history record
	history := models.SyncHistory{
		Provider:  "opal",
		Status:    "running",
		StartedAt: time.Now(),
	}
	if err := s.db.Create(&history).Error; err != nil {
		fmt.Printf("Warning: Failed to create sync history: %v\n", err)
	}

	startTime := time.Now()
	log("Starting OPAL order import...")

	// Get the OPAL provider from registry
	// On microservice nodes without OPAL credentials, this provider won't be registered.
	// That's expected - such nodes receive shipment data via Mesh Sync instead.
	providerInt, err := s.registry.Get("opal")
	if err != nil {
		s.updateSyncHistoryErrorWithContext(&history, err, map[string]interface{}{
			"step": "get_provider",
			"provider": "opal",
		})
		return fmt.Errorf("OPAL provider not configured on this node (sync-only mode): %w", err)
	}

	// Type assert to get access to FetchRecentOrders
	opalProvider, ok := providerInt.(*opal.Provider)
	if !ok {
		err := fmt.Errorf("provider is not OPAL")
		s.updateSyncHistoryErrorWithContext(&history, err, map[string]interface{}{
			"step": "type_assertion",
		})
		return err
	}

	// Fetch orders from OPAL
	orders, err := opalProvider.FetchRecentOrders(ctx)
	if err != nil {
		wrappedErr := fmt.Errorf("failed to fetch OPAL orders: %w", err)
		// Save detailed error with context
		s.updateSyncHistoryErrorWithContext(&history, wrappedErr, map[string]interface{}{
			"step":          "fetch_orders",
			"original_error": err.Error(),
		})
		return wrappedErr
	}

	log("Fetched %d orders from OPAL", len(orders))

	// Process each order
	created := 0
	updated := 0
	skipped := 0

	for _, order := range orders {
		// Skip orders without valid identifiers
		if order.TrackingNumber == "" && order.HwbNumber == "" {
			skipped++
			continue
		}

		// Build tracking number (prefer OCU, fallback to HWB)
		trackingNum := order.TrackingNumber
		if trackingNum == "" {
			trackingNum = order.HwbNumber
		}

		// Build raw response JSON with all the scraped data
		rawData := map[string]interface{}{
			"ocu_number":       order.TrackingNumber,
			"hwb_number":       order.HwbNumber,
			"product_type":     order.ProductType,
			"reference":        order.Reference,
			"created_at":       order.CreatedAt,
			"created_by":       order.CreatedBy,
			"pickup_name":      order.PickupName,
			"pickup_name2":     order.PickupName2,
			"pickup_contact":   order.PickupContact,
			"pickup_phone":     order.PickupPhone,
			"pickup_email":     order.PickupEmail,
			"pickup_street":    order.PickupStreet,
			"pickup_city":      order.PickupCity,
			"pickup_zip":       order.PickupZip,
			"pickup_country":   order.PickupCountry,
			"pickup_note":      order.PickupNote,
			"pickup_date":      order.PickupDate,
			"pickup_time":      fmt.Sprintf("%s-%s", order.PickupTimeFrom, order.PickupTimeTo),
			"pickup_vehicle":   order.PickupVehicle,
			"delivery_name":    order.DeliveryName,
			"delivery_name2":   order.DeliveryName2,
			"delivery_contact": order.DeliveryContact,
			"delivery_phone":   order.DeliveryPhone,
			"delivery_email":   order.DeliveryEmail,
			"delivery_street":  order.DeliveryStreet,
			"delivery_city":    order.DeliveryCity,
			"delivery_zip":     order.DeliveryZip,
			"delivery_country": order.DeliveryCountry,
			"delivery_note":    order.DeliveryNote,
			"delivery_date":    order.DeliveryDate,
			"delivery_time":    fmt.Sprintf("%s-%s", order.DeliveryTimeFrom, order.DeliveryTimeTo),
			"description":      order.Description,
			"dimensions":       order.Dimensions,
			"status":           order.Status,
			"status_date":      order.StatusDate,
			"status_time":      order.StatusTime,
			"receiver":         order.Receiver,
		}
		if order.PackageCount != nil {
			rawData["package_count"] = *order.PackageCount
		}
		if order.Weight != nil {
			rawData["weight"] = *order.Weight
		}
		if order.Value != nil {
			rawData["value"] = *order.Value
		}
		rawJSON, _ := json.Marshal(rawData)

		// Parse status date for LastActivityAt (format: "04.11.25 09:03" or "04.11.2025")
		var lastActivity *time.Time
		if order.StatusDate != "" {
			dateStr := order.StatusDate
			if order.StatusTime != "" {
				dateStr += " " + order.StatusTime
			}
			// Try various formats
			formats := []string{
				"02.01.06 15:04",   // 04.11.25 09:03
				"02.01.2006 15:04", // 04.11.2025 09:03
				"02.01.06",         // 04.11.25
				"02.01.2006",       // 04.11.2025
			}
			for _, format := range formats {
				if t, err := time.Parse(format, dateStr); err == nil {
					// Fix year if 2-digit (06 -> 2006)
					if t.Year() < 100 {
						t = t.AddDate(2000, 0, 0)
					}
					lastActivity = &t
					break
				}
			}
		}

		// Determine status - use OPAL status if available, otherwise pending
		status := models.DeliveryStatusPending
		if order.Status == "Zugestellt" || order.Status == "ausgeliefert" || order.Status == "geliefert" {
			// Check if receiver is "Fehlanfahrt" - that means failed delivery, not successful
			if order.Receiver == "Fehlanfahrt" {
				status = models.DeliveryStatusError
			} else {
				status = models.DeliveryStatusDelivered
			}
		} else if order.Status == "Abgeholt" || order.Status == "AKTIV" {
			status = models.DeliveryStatusShipped
		} else if order.Status == "Storniert" || order.Status == "STORNO" {
			status = models.DeliveryStatusCancelled
		} else if order.Status == "Fehlanfahrt" {
			status = models.DeliveryStatusError
		}

		// Search for existing delivery by OCU or HWB tracking number
		var delivery models.StockPickingDelivery
		query := s.db.Where("tracking_number = ?", order.TrackingNumber)
		if order.HwbNumber != "" {
			query = query.Or("tracking_number = ?", order.HwbNumber)
		}
		result := query.First(&delivery)

		if result.Error == nil {
			// Order found - always update with latest data
			oldStatus := delivery.Status
			if status != models.DeliveryStatusPending {
				delivery.Status = status
			}
			delivery.RawResponse = string(rawJSON)
			if lastActivity != nil {
				delivery.LastActivityAt = lastActivity
			}

			if err := s.db.Save(&delivery).Error; err != nil {
				log("Error updating delivery %d: %v", delivery.ID, err)
				continue
			}

			if oldStatus != delivery.Status {
				s.createTrackingEntry(delivery.ID,
					fmt.Sprintf("Status updated from OPAL: %s -> %s (%s)",
						oldStatus, status, order.Receiver),
					status)
				log("Updated: #%d %s -> %s", delivery.ID, oldStatus, status)
			}
			updated++
		} else {
			// Order not found - create new record
			newDelivery := models.StockPickingDelivery{
				TrackingNumber: trackingNum,
				Status:         status,
				RawResponse:    string(rawJSON),
				LastActivityAt: lastActivity,
			}

			if err := s.db.Create(&newDelivery).Error; err != nil {
				log("Error creating delivery: %v", err)
				continue
			}

			s.createTrackingEntry(newDelivery.ID,
				fmt.Sprintf("Imported from OPAL: %s -> %s", order.PickupName, order.DeliveryName),
				status)

			log("Created: #%d OCU=%s HWB=%s Status=%s",
				newDelivery.ID, order.TrackingNumber, order.HwbNumber, status)
			created++
		}
	}

	log("OPAL import completed: %d created, %d updated, %d skipped", created, updated, skipped)

	// Update sync history
	completedAt := time.Now()
	duration := int(time.Since(startTime).Milliseconds())
	history.CompletedAt = &completedAt
	history.Duration = duration
	history.Status = "success"
	history.Created = created
	history.Updated = updated
	history.Skipped = skipped
	if err := s.db.Save(&history).Error; err != nil {
		fmt.Printf("Warning: Failed to update sync history: %v\n", err)
	}

	return nil
}

// updateSyncHistoryErrorWithContext is a helper to mark sync as failed with detailed debug info and context
func (s *Service) updateSyncHistoryErrorWithContext(history *models.SyncHistory, err error, context map[string]interface{}) {
	completedAt := time.Now()
	history.CompletedAt = &completedAt
	history.Status = "error"
	history.Errors = 1
	if err != nil {
		history.ErrorDetail = err.Error()

		// Start with provided context
		debugInfo := make(map[string]interface{})
		for k, v := range context {
			debugInfo[k] = v
		}

		// Add standard debug fields
		debugInfo["error_message"] = err.Error()
		debugInfo["timestamp"] = time.Now().Format(time.RFC3339)
		debugInfo["provider"] = history.Provider

		errorStr := err.Error()
		if len(errorStr) > 100 {
			debugInfo["full_error"] = errorStr
			debugInfo["error_type"] = "detailed"
		}

		// Categorize error for AI analysis
		if strings.Contains(errorStr, "playwright") ||
		   strings.Contains(errorStr, "selector") ||
		   strings.Contains(errorStr, "timeout") ||
		   strings.Contains(errorStr, "navigation") ||
		   strings.Contains(errorStr, "Stderr") {
			debugInfo["error_category"] = "playwright_scraper"
			debugInfo["likely_cause"] = "Frontend changed - selectors may need updating"
			debugInfo["ai_analysis_hint"] = "Check Playwright selectors and page structure changes"
		} else if strings.Contains(errorStr, "connection") ||
		          strings.Contains(errorStr, "network") {
			debugInfo["error_category"] = "network"
			debugInfo["likely_cause"] = "Network connectivity issue"
			debugInfo["ai_analysis_hint"] = "Check network connectivity and API availability"
		} else if strings.Contains(errorStr, "parse") ||
		          strings.Contains(errorStr, "JSON") ||
		          strings.Contains(errorStr, "unmarshal") {
			debugInfo["error_category"] = "parsing"
			debugInfo["likely_cause"] = "Data format changed"
			debugInfo["ai_analysis_hint"] = "Check response structure and parsing logic"
		} else {
			debugInfo["error_category"] = "other"
		}

		// Extract stderr if present (Playwright output)
		if strings.Contains(errorStr, "Stderr:") {
			parts := strings.Split(errorStr, "Stderr:")
			if len(parts) > 1 {
				debugInfo["playwright_stderr"] = strings.TrimSpace(parts[1])
			}
		}

		history.DebugInfo = debugInfo
	}
	if saveErr := s.db.Save(history).Error; saveErr != nil {
		fmt.Printf("Warning: Failed to update sync history: %v\n", saveErr)
	}
}

// updateSyncHistoryError is a helper to mark sync as failed with detailed debug info
func (s *Service) updateSyncHistoryError(history *models.SyncHistory, err error) {
	s.updateSyncHistoryErrorWithContext(history, err, nil)
	completedAt := time.Now()
	history.CompletedAt = &completedAt
	history.Status = "error"
	history.Errors = 1
	if err != nil {
		history.ErrorDetail = err.Error()

		// Save detailed debug info for AI analysis
		debugInfo := map[string]interface{}{
			"error_message": err.Error(),
			"timestamp":     time.Now().Format(time.RFC3339),
			"provider":      history.Provider,
		}

		// Try to extract more context if available
		errorStr := err.Error()
		if len(errorStr) > 100 {
			// Store full error message for detailed analysis
			debugInfo["full_error"] = errorStr
			debugInfo["error_type"] = "detailed"
		}

		// Check if this is a Playwright/scraper error
		if strings.Contains(errorStr, "playwright") ||
		   strings.Contains(errorStr, "selector") ||
		   strings.Contains(errorStr, "timeout") ||
		   strings.Contains(errorStr, "navigation") {
			debugInfo["error_category"] = "playwright_scraper"
			debugInfo["likely_cause"] = "Frontend changed - selectors may need updating"
		} else if strings.Contains(errorStr, "connection") ||
		          strings.Contains(errorStr, "network") {
			debugInfo["error_category"] = "network"
			debugInfo["likely_cause"] = "Network connectivity issue"
		} else {
			debugInfo["error_category"] = "other"
		}

		history.DebugInfo = debugInfo
	}
	if saveErr := s.db.Save(history).Error; saveErr != nil {
		fmt.Printf("Warning: Failed to update sync history: %v\n", saveErr)
	}
}

// ImportDhlOrders fetches orders from DHL and updates the database
func (s *Service) ImportDhlOrders(ctx context.Context) error {
	log := func(format string, args ...interface{}) {
		fmt.Printf("[DHL Import] "+format+"\n", args...)
	}

	log("Starting DHL order import...")

	// Get the DHL provider from registry
	// On microservice nodes without DHL credentials, this provider won't be registered.
	// That's expected - such nodes receive shipment data via Mesh Sync instead.
	providerInt, err := s.registry.Get("dhl")
	if err != nil {
		return fmt.Errorf("DHL provider not configured on this node (sync-only mode): %w", err)
	}

	// Type assert to get access to FetchRecentShipments
	dhlProvider, ok := providerInt.(*dhl.Provider)
	if !ok {
		return fmt.Errorf("provider is not DHL")
	}

	// Fetch shipments from DHL (last 14 days)
	shipments, err := dhlProvider.FetchRecentShipments(ctx, 14)
	if err != nil {
		return fmt.Errorf("failed to fetch DHL shipments: %w", err)
	}

	log("Fetched %d shipments from DHL", len(shipments))

	// Process each shipment
	created := 0
	updated := 0
	skipped := 0

	// Default warehouse/sender info for DHL (since CSV only has recipient)
	defaultSender := s.config.Warehouse.Name
	if defaultSender == "" {
		defaultSender = "InBody Europe B.V."
	}

	for _, shipment := range shipments {
		// Skip orders without valid tracking number
		if shipment.TrackingNumber == "" {
			skipped++
			continue
		}

		// Build raw response JSON with all the scraped data
		// Includes both native DHL fields and standardized delivery_* keys for frontend
		rawData := map[string]interface{}{
			// Standard Identifiers
			"tracking_number": shipment.TrackingNumber,
			"reference":       shipment.Reference,

			// Native DHL Fields (keep for reference)
			"international_number": shipment.InternationalNum,
			"billing_number":       shipment.BillingNumber,
			"recipient_name":       shipment.RecipientName,
			"recipient_street":     shipment.RecipientStreet,
			"recipient_zip":        shipment.RecipientZip,
			"recipient_city":       shipment.RecipientCity,
			"recipient_country":    shipment.RecipientCountry,
			"status":               shipment.Status,
			"status_date":          shipment.StatusDate,
			"note":                 shipment.Note,
			"delivered_to_name":    shipment.DeliveredToName,
			"delivered_to_street":  shipment.DeliveredToStreet,
			"delivered_to_zip":     shipment.DeliveredToZip,
			"delivered_to_city":    shipment.DeliveredToCity,
			"delivered_to_country": shipment.DeliveredToCountry,
			"product":              shipment.Product,
			"services":             shipment.Services,
			"provider":             "dhl",

			// Standardized Delivery Address (Mapped for Frontend)
			"delivery_name":    shipment.RecipientName,
			"delivery_street":  shipment.RecipientStreet,
			"delivery_zip":     shipment.RecipientZip,
			"delivery_city":    shipment.RecipientCity,
			"delivery_country": shipment.RecipientCountry,

			// Standardized Pickup/Sender Address (Defaulted from warehouse config)
			"pickup_name": defaultSender,
			"pickup_city": s.config.Warehouse.City,

			// Description from note field
			"description": shipment.Note,
		}
		rawJSON, _ := json.Marshal(rawData)

		// Map DHL status to internal status
		status := dhl.MapStatus(shipment.Status)

		// Parse status_date for LastActivityAt
		var lastActivity *time.Time
		if shipment.StatusDate != "" {
			// Try parsing ISO format (2026-01-22T11:31:19)
			if t, err := time.Parse("2006-01-02T15:04:05", shipment.StatusDate); err == nil {
				lastActivity = &t
			} else if t, err := time.Parse("2006-01-02T15:04", shipment.StatusDate); err == nil {
				lastActivity = &t
			}
		}

		// Search for existing delivery by tracking number
		var delivery models.StockPickingDelivery
		result := s.db.Where("tracking_number = ?", shipment.TrackingNumber).First(&delivery)

		if result.Error == nil {
			// Order found - always update with latest data
			oldStatus := delivery.Status
			if status != models.DeliveryStatusPending {
				delivery.Status = status
			}
			delivery.RawResponse = string(rawJSON)
			if lastActivity != nil {
				delivery.LastActivityAt = lastActivity
			}

			if err := s.db.Save(&delivery).Error; err != nil {
				log("Error updating delivery %d: %v", delivery.ID, err)
				continue
			}

			if oldStatus != delivery.Status {
				s.createTrackingEntry(delivery.ID,
					fmt.Sprintf("Status updated from DHL: %s -> %s", oldStatus, status),
					status)
				log("Updated: #%d %s -> %s", delivery.ID, oldStatus, status)
			}
			updated++
		} else {
			// Order not found - create new record
			newDelivery := models.StockPickingDelivery{
				TrackingNumber: shipment.TrackingNumber,
				Status:         status,
				RawResponse:    string(rawJSON),
				LastActivityAt: lastActivity,
			}

			if err := s.db.Create(&newDelivery).Error; err != nil {
				log("Error creating delivery: %v", err)
				continue
			}

			s.createTrackingEntry(newDelivery.ID,
				fmt.Sprintf("Imported from DHL: %s (%s)", shipment.RecipientName, shipment.Product),
				status)

			log("Created: #%d Tracking=%s Status=%s", newDelivery.ID, shipment.TrackingNumber, status)
			created++
		}
	}

	log("DHL import completed: %d created, %d updated, %d skipped", created, updated, skipped)
	return nil
}
