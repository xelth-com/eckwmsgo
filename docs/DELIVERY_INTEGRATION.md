# OPAL Delivery Integration for eckwmsgo

## Overview

This document describes the OPAL Kurier delivery provider integration for eckwmsgo. The integration follows the "Odoo Way" architecture with clean abstractions and modular design.

## Architecture

### Module Structure

```
eckwmsgo/
├── internal/
│   ├── delivery/               # Core delivery abstraction layer
│   │   ├── provider.go        # ProviderInterface definition
│   │   ├── registry.go        # Provider registry
│   │   └── opal/              # OPAL-specific implementation
│   │       └── provider.go    # OPAL provider adapter
│   ├── models/
│   │   └── delivery.go        # Database models
│   └── services/
│       └── delivery/
│           └── service.go     # Business logic service
└── scripts/
    └── delivery/
        ├── create-opal-order.js  # Node.js automation script
        └── package.json           # Node dependencies
```

### Key Components

#### 1. Provider Interface (`internal/delivery/provider.go`)

The core abstraction that all delivery providers must implement:

```go
type ProviderInterface interface {
    Code() string
    Name() string
    CreateShipment(ctx context.Context, req *DeliveryRequest) (*DeliveryResponse, error)
    CancelShipment(ctx context.Context, trackingNumber string) error
    GetStatus(ctx context.Context, trackingNumber string) (*TrackingStatus, error)
    ValidateAddress(ctx context.Context, addr *Address) error
}
```

#### 2. OPAL Provider (`internal/delivery/opal/provider.go`)

Go wrapper that executes the Node.js Playwright script:

- Converts Go structs to JSON format expected by OPAL
- Executes Node.js script via `exec.Command`
- Parses JSON response from script
- Handles errors and timeouts

#### 3. Database Models (`internal/models/delivery.go`)

Three main tables:

- **delivery_carrier**: Provider configurations (OPAL credentials, settings)
- **stock_picking_delivery**: Links pickings to shipments (tracking numbers, status)
- **delivery_tracking**: Tracking event history

#### 4. Delivery Service (`internal/services/delivery/service.go`)

Business logic layer:

- `CreateShipment()`: Queue a shipment for creation
- `ProcessPendingShipments()`: Worker function to process queue
- `GetDeliveryStatus()`: Retrieve shipment status
- `CancelShipment()`: Cancel a shipment

#### 5. Node.js Script (`scripts/delivery/create-opal-order.js`)

Playwright automation script:

- Logs into OPAL Kurier website
- Fills out "Neuer Auftrag" form
- Submits order and extracts tracking number
- Returns structured JSON response

## Setup

### 1. Install Node.js Dependencies

```bash
cd scripts/delivery
npm install
npm run install:playwright
```

### 2. Configure Environment Variables

Add to your `.env` file:

```env
OPAL_URL=https://opal-kurier.de
OPAL_USERNAME=your_username
OPAL_PASSWORD=your_password
```

### 3. Register OPAL Provider

In your `main.go` or initialization code:

```go
import (
    "github.com/dmytrosurovtsev/eckwmsgo/internal/delivery"
    "github.com/dmytrosurovtsev/eckwmsgo/internal/delivery/opal"
)

// Create OPAL provider
opalProvider, err := opal.NewProvider(opal.Config{
    ScriptPath: "./scripts/delivery/create-opal-order.js",
    NodePath:   "node",
    Username:   os.Getenv("OPAL_USERNAME"),
    Password:   os.Getenv("OPAL_PASSWORD"),
    Headless:   true,
    Timeout:    300,
})
if err != nil {
    log.Fatal(err)
}

// Register with global registry
registry := delivery.GetGlobalRegistry()
registry.Register(opalProvider)

// Create delivery service
deliveryService := delivery.NewService(db)
deliveryService.RegisterProvider(opalProvider)
```

### 4. Create Delivery Carrier Record

```sql
INSERT INTO delivery_carrier (name, provider_code, active, config_json)
VALUES ('OPAL Express', 'opal', true, '{}');
```

## Usage

### Creating a Shipment

```go
// Queue shipment for processing
err := deliveryService.CreateShipment(ctx, pickingID, "opal")

// Process pending shipments (call from worker)
err := deliveryService.ProcessPendingShipments(ctx)
```

### Checking Status

```go
delivery, err := deliveryService.GetDeliveryStatus(pickingID)
fmt.Printf("Tracking: %s\n", delivery.TrackingNumber)
fmt.Printf("Status: %s\n", delivery.Status)
```

### Cancelling a Shipment

```go
err := deliveryService.CancelShipment(ctx, pickingID)
```

## API Endpoints (To Be Implemented)

Recommended REST endpoints:

```
POST   /api/delivery/shipments              # Create shipment
GET    /api/delivery/shipments/:id          # Get shipment status
DELETE /api/delivery/shipments/:id          # Cancel shipment
GET    /api/delivery/shipments/:id/tracking # Get tracking history
GET    /api/delivery/carriers               # List available carriers
```

## Background Worker

You should run a background goroutine to process pending shipments:

```go
func StartDeliveryWorker(service *delivery.Service) {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()

    for range ticker.C {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
        if err := service.ProcessPendingShipments(ctx); err != nil {
            log.Printf("Worker error: %v", err)
        }
        cancel()
    }
}

// In main.go
go StartDeliveryWorker(deliveryService)
```

## Extending with Other Providers

To add DHL, UPS, or other providers:

1. Implement `delivery.ProviderInterface`
2. Register with registry
3. Add carrier record to database

Example:

```go
type DHLProvider struct {
    apiKey string
}

func (p *DHLProvider) Code() string { return "dhl" }
func (p *DHLProvider) Name() string { return "DHL Express" }
// ... implement other methods
```

## Security Considerations

1. **Credentials**: Store OPAL credentials securely (env vars or secrets manager)
2. **Script Path**: Validate script path to prevent code injection
3. **Timeouts**: Set reasonable timeouts to prevent hanging
4. **Error Handling**: Never expose sensitive error details to clients
5. **Rate Limiting**: Consider rate limiting to prevent abuse

## Troubleshooting

### Script Fails to Execute

- Verify Node.js is installed: `node --version`
- Check script path is correct
- Run script manually to test: `node scripts/delivery/create-opal-order.js --data=test.json`

### Login Failures

- Verify OPAL credentials in `.env`
- Check if OPAL website structure changed
- Try running with `--headless=false` to see browser

### Timeouts

- Increase timeout in provider config
- Check network connectivity
- OPAL website might be slow

## Future Improvements

1. **Caching**: Cache OPAL authentication cookies
2. **Retry Logic**: Implement exponential backoff for failed shipments
3. **Webhooks**: Support carrier webhooks for status updates
4. **Label Storage**: Store PDF labels in S3/MinIO instead of database
5. **Batch Processing**: Process multiple shipments in parallel
6. **Monitoring**: Add Prometheus metrics for shipment success/failure rates

## References

- Odoo delivery module: https://github.com/odoo/odoo/tree/16.0/addons/delivery
- Playwright documentation: https://playwright.dev/
- OPAL Kurier: https://opal-kurier.de
