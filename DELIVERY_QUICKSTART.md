# OPAL Delivery Integration - Quick Start

## Prerequisites

- Node.js 16+ installed
- Go 1.21+ installed
- PostgreSQL database
- OPAL Kurier account with credentials

## Installation Steps

### 1. Install Node.js Dependencies

```bash
cd scripts/delivery
npm install
npx playwright install chromium
```

### 2. Configure Environment

Add to your `.env` file in the project root:

```env
# OPAL Kurier Configuration
OPAL_URL=https://opal-kurier.de
OPAL_USERNAME=your_username_here
OPAL_PASSWORD=your_password_here
```

### 3. Run Database Migrations

The delivery tables will be created automatically on server start:

```bash
# Start the server (migrations run automatically)
go run cmd/api/main.go
```

Or manually trigger migrations:

```bash
# The tables created:
# - delivery_carrier
# - stock_picking_delivery
# - delivery_tracking
```

### 4. Register OPAL Provider (Example)

Create a file `cmd/api/delivery_init.go`:

```go
package main

import (
    "log"
    "os"
    "path/filepath"

    "github.com/dmytrosurovtsev/eckwmsgo/internal/delivery"
    "github.com/dmytrosurovtsev/eckwmsgo/internal/delivery/opal"
    "github.com/dmytrosurovtsev/eckwmsgo/internal/database"
    "github.com/dmytrosurovtsev/eckwmsgo/internal/models"
)

func InitializeDeliveryProviders(db *database.DB) error {
    // Get script path (relative to project root)
    scriptPath, _ := filepath.Abs("./scripts/delivery/create-opal-order.js")

    // Create OPAL provider
    opalProvider, err := opal.NewProvider(opal.Config{
        ScriptPath: scriptPath,
        NodePath:   "node",
        Username:   os.Getenv("OPAL_USERNAME"),
        Password:   os.Getenv("OPAL_PASSWORD"),
        URL:        os.Getenv("OPAL_URL"),
        Headless:   true,
        Timeout:    300, // 5 minutes
    })
    if err != nil {
        return err
    }

    // Register with global registry
    registry := delivery.GetGlobalRegistry()
    if err := registry.Register(opalProvider); err != nil {
        return err
    }

    // Create or update OPAL carrier in database
    carrier := models.DeliveryCarrier{
        Name:         "OPAL Express",
        ProviderCode: "opal",
        Active:       true,
        ConfigJSON:   "{}",
    }

    // Upsert carrier
    var existing models.DeliveryCarrier
    result := db.Where("provider_code = ?", "opal").First(&existing)
    if result.Error != nil {
        // Create new
        if err := db.Create(&carrier).Error; err != nil {
            return err
        }
        log.Println("✅ Created OPAL carrier")
    } else {
        // Update existing
        existing.Active = true
        if err := db.Save(&existing).Error; err != nil {
            return err
        }
        log.Println("✅ Updated OPAL carrier")
    }

    log.Println("✅ OPAL delivery provider initialized")
    return nil
}
```

Update `cmd/api/main.go` to call this function:

```go
// After database migration
if err := InitializeDeliveryProviders(db); err != nil {
    log.Printf("⚠️ Failed to initialize delivery providers: %v", err)
}
```

### 5. Test the Integration

Create a test file `test_opal.go`:

```go
package main

import (
    "context"
    "log"

    "github.com/dmytrosurovtsev/eckwmsgo/internal/delivery"
    "github.com/dmytrosurovtsev/eckwmsgo/internal/services/delivery"
)

func testOpalShipment() {
    // Initialize database and service
    db := /* your DB connection */
    deliveryService := delivery.NewService(db)

    // Create test shipment
    pickingID := int64(1) // Use actual picking ID
    err := deliveryService.CreateShipment(context.Background(), pickingID, "opal")
    if err != nil {
        log.Fatalf("Failed to create shipment: %v", err)
    }

    log.Println("✅ Shipment queued for processing")

    // Process immediately (normally done by worker)
    err = deliveryService.ProcessPendingShipments(context.Background())
    if err != nil {
        log.Fatalf("Failed to process shipment: %v", err)
    }

    // Check status
    status, err := deliveryService.GetDeliveryStatus(pickingID)
    if err != nil {
        log.Fatalf("Failed to get status: %v", err)
    }

    log.Printf("Status: %s", status.Status)
    log.Printf("Tracking: %s", status.TrackingNumber)
}
```

## Testing the Script Manually

Test the Node.js script independently:

```bash
cd scripts/delivery

# Create test data file
cat > test-order.json <<EOF
{
  "deliveryName1": "Test Customer",
  "deliveryStreet": "Teststraße 123",
  "deliveryZip": "12345",
  "deliveryCity": "Berlin",
  "deliveryCountry": "DE",
  "packageCount": 1,
  "packageWeight": 5.0,
  "packageDescription": "Test Package"
}
EOF

# Run script
node create-opal-order.js --data=test-order.json --json-output
```

Expected output (JSON):

```json
{
  "success": true,
  "trackingNumber": "OPAL123456",
  "orderNumber": "ORD-001",
  "message": "Order created successfully"
}
```

## Production Deployment

### 1. Background Worker

Add to your `main.go`:

```go
import (
    "time"
    deliveryservice "github.com/dmytrosurovtsev/eckwmsgo/internal/services/delivery"
)

func startDeliveryWorker(service *deliveryservice.Service) {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()

    for range ticker.C {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
        if err := service.ProcessPendingShipments(ctx); err != nil {
            log.Printf("⚠️ Delivery worker error: %v", err)
        }
        cancel()
    }
}

// In main()
deliveryService := deliveryservice.NewService(db)
go startDeliveryWorker(deliveryService)
```

### 2. Docker Support

If using Docker, add to your `Dockerfile`:

```dockerfile
# Install Node.js
FROM golang:1.21 AS builder
RUN curl -fsSL https://deb.nodesource.com/setup_20.x | bash -
RUN apt-get install -y nodejs

# Copy and install Node dependencies
COPY scripts/delivery/package.json scripts/delivery/package-lock.json ./scripts/delivery/
WORKDIR /app/scripts/delivery
RUN npm install
RUN npx playwright install --with-deps chromium
WORKDIR /app

# Rest of your Dockerfile...
```

### 3. Monitoring

Add logging and metrics:

```go
log.Printf("📦 Processing %d pending shipments", count)
log.Printf("✅ Shipment created: %s (Tracking: %s)", picking.Name, tracking)
log.Printf("❌ Shipment failed: %s (Error: %s)", picking.Name, err)
```

## Troubleshooting

### "node: command not found"

```bash
# Check Node.js installation
node --version
npm --version

# If not installed, install Node.js
# Ubuntu/Debian:
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt-get install -y nodejs

# macOS:
brew install node

# Windows:
# Download from https://nodejs.org
```

### "Playwright browser not found"

```bash
cd scripts/delivery
npx playwright install chromium
```

### "Login failed"

- Verify credentials in `.env`
- Test login manually with `--headless=false`
- Check OPAL website for changes

### "Script timeout"

- Increase timeout in provider config (default: 300s)
- Check OPAL website performance
- Verify network connectivity

## Next Steps

- Read full documentation: [DELIVERY_INTEGRATION.md](docs/DELIVERY_INTEGRATION.md)
- Implement API endpoints for frontend integration
- Set up monitoring and alerting
- Configure retry logic for failed shipments
- Add more delivery providers (DHL, UPS, etc.)

## Support

- GitHub Issues: https://github.com/dmytrosurovtsev/eckwmsgo/issues
- OPAL Support: Check OPAL Kurier website for contact details
