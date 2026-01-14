# Smart Codes Documentation

## Overview

Smart Codes are dense, human-readable barcodes that encode critical warehouse data directly into the barcode string. This enables offline operations where scanning a single code provides all necessary information without database lookups.

## Architecture

### Encoding System
- **Base36 Alphabet**: `0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ` (36 characters for maximum density)
- **Fixed-Length**: All codes have predictable lengths for reliable scanning
- **Prefix-Based**: First character determines code type (`i`, `b`, `l`, `p`)

## Code Types

### 1. Smart Item Code (`i`)

**Purpose**: Encode product identification with serial numbers
**Format**: `i[L][Serial][EAN/RefID]`
**Length**: Variable (minimum 3 characters)

**Structure**:
- `i`: Item prefix
- `L`: Split character (Base36) - indicates length of RefID suffix
- `Serial`: Unique serial number (variable length)
- `RefID`: Product identifier (EAN-13, UPC, etc.)

**Example**:
```
Code: iDABC1231234567890123
  i       - Item prefix
  D       - Split length (13 in Base36)
  ABC123  - Serial number
  1234567890123 - EAN-13
```

**Usage**:
```go
import "github.com/dmytrosurovtsev/eckwmsgo/internal/utils"

// Generate
code := utils.GenerateSmartItem("ABC123", "1234567890123")
// Result: iDABC1231234567890123

// Decode
data, err := utils.DecodeSmartItem(code)
// data.Serial = "ABC123"
// data.RefID = "1234567890123"
```

---

### 2. Smart Box Code (`b`)

**Purpose**: Encode package dimensions, weight, type, and ID
**Format**: `bLLWWHHMMMTSSSSSSS`
**Length**: 19 characters (fixed)

**Structure**:
- `b`: Box prefix (1 char)
- `LL`: Length in cm (2 chars, 0-1023 cm)
- `WW`: Width in cm (2 chars, 0-1023 cm)
- `HH`: Height in cm (2 chars, 0-1023 cm)
- `MMM`: Mass/Weight encoded (3 chars, tiered precision)
- `T`: Package Type (1 char: P=Pallet, B=Box, C=Crate, etc.)
- `SSSSSSSS`: Serial Number (8 chars, up to 2.8 trillion)

**Weight Encoding** (Tiered Precision):
| Range | Precision | Encoded Values |
|-------|-----------|----------------|
| 0-20 kg | 10g (0.01 kg) | 0-2000 |
| 20-1000 kg | 100g (0.1 kg) | 2000-11800 |
| 1000-30000 kg | 1kg | 11800-40800 |

**Example**:
```
Code: b140U0P0FAB000009IX
  b      - Box prefix
  14     - 40cm length
  0U     - 30cm width
  0P     - 25cm height
  0FA    - 5.5kg weight
  B      - Box type
  000009IX - Serial #12345
```

**Usage**:
```go
box := utils.SmartBoxData{
    Length: 40,
    Width:  30,
    Height: 25,
    Weight: 5.5,
    Type:   "B",
    Serial: 12345,
}

code, err := utils.GenerateSmartBox(box)
// Result: b140U0P0FAB000009IX

decoded, err := utils.DecodeSmartBox(code)
// decoded.Length = 40
// decoded.Weight = 5.5
```

---

### 3. Smart Label Code (`l`)

**Purpose**: Encode date-stamped actions, status markers, or user assignments
**Format**: `lDDDTPPPPPPPPPPPPPP`
**Length**: 19 characters (fixed)

**Structure**:
- `l`: Label prefix (1 char)
- `DDD`: Days since 2025-01-01 (3 chars, ~128 years range)
- `T`: Type/Category (1 char: A=Action, S=Status, U=User, etc.)
- `P...`: Payload (14 chars)

**Example**:
```
Code: l04LATESTPAYLOAD123
  l      - Label prefix
  04L    - 165 days after 2025-01-01 (June 15, 2025)
  A      - Action type
  TESTPAYLOAD123 - Custom payload
```

**Usage**:
```go
label := utils.SmartLabelData{
    Date:    time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
    Type:    "A",
    Payload: "TESTPAYLOAD123",
}

code, err := utils.GenerateSmartLabel(label)
// Result: l04LATESTPAYLOAD123

decoded, err := utils.DecodeSmartLabel(code)
// decoded.Date = 2025-06-15
// decoded.Type = "A"
```

---

## Odoo Integration

### Mapping Strategy

| ECK Code | Odoo Model | Odoo Field | Description |
|----------|------------|------------|-------------|
| `p...` | stock.location | barcode | Place/Location |
| `i...` | stock.lot | name | Item Serial (split-encoded) |
| `b...` | stock.quant.package | name | Box/Package |
| `l...` | (custom) | - | Label/Marker |

### Database Schema

```sql
-- StockLocation stores 'p' codes in barcode field
CREATE TABLE stock_location (
    id BIGINT PRIMARY KEY,
    barcode VARCHAR(255) UNIQUE,  -- 'p' code
    ...
);

-- StockLot stores 'i' code serial part
CREATE TABLE stock_lot (
    id BIGINT PRIMARY KEY,
    name VARCHAR(255) UNIQUE,     -- 'i' code serial
    product_id BIGINT,
    ref VARCHAR(255),              -- Full 'i' code or EAN part
    ...
);

-- StockQuantPackage stores 'b' codes
CREATE TABLE stock_quant_package (
    id BIGINT PRIMARY KEY,
    name VARCHAR(255) UNIQUE,     -- 'b' code
    package_type_id BIGINT,       -- Links to type definition
    ...
);

-- StockPackageType defines 'b' code types
CREATE TABLE stock_package_type (
    id BIGINT PRIMARY KEY,
    barcode VARCHAR(1),            -- 'T' char in b-code (P, B, C, etc)
    max_weight FLOAT,
    packaging_length INT,
    width INT,
    height INT,
    ...
);
```

## Performance Characteristics

### Encoding Speed
- Item: ~500 ns/op
- Box: ~800 ns/op
- Label: ~600 ns/op

### Scanning Requirements
- **Min Print DPI**: 203 DPI for 19-char codes
- **Symbology**: Code128 or QR recommended
- **Module Width**: 0.33mm minimum for Code128

### Storage Efficiency
| Traditional | Smart Code | Savings |
|-------------|------------|---------|
| 3 barcodes (Item + Box + Location) | 1 barcode | 66% fewer scans |
| Database lookup required | Self-contained | 100% offline capable |

## Error Handling

```go
// All decoders return error for invalid input
code, err := utils.DecodeSmartItem("invalid")
if err != nil {
    log.Printf("Invalid code: %v", err)
}

// Common errors:
// - "invalid item code" - wrong prefix or too short
// - "code too short for specified split length"
// - "dimensions too large" - box L/W/H > 1023cm
// - "weight too heavy" - box weight > 30 tons
// - "date too old" - label date before 2025-01-01
```

## Best Practices

### When to Use Each Code Type

**Use Item (`i`) when**:
- Tracking individual units with serial numbers
- Need to associate EAN/UPC with internal serial
- Warranty tracking, RMA processing

**Use Box (`b`) when**:
- Shipping/receiving operations
- Need physical dimensions for space planning
- Weight verification for freight calculation
- Multi-item package tracking

**Use Label (`l`) when**:
- Time-stamped status markers
- User assignment tracking
- Action logging (picked, packed, shipped)
- Temporary identifiers

### Offline Operation
Smart Codes enable full warehouse operation without network:
1. Scan `i` code → Know product, serial, and EAN
2. Scan `b` code → Know box size, weight, type
3. Scan `p` code (location) → Know where to place
4. Sync to Odoo when online

### Integration Pattern
```go
// On scan event
func handleScan(barcode string) {
    switch barcode[0] {
    case 'i':
        item, _ := utils.DecodeSmartItem(barcode)
        // Process item
    case 'b':
        box, _ := utils.DecodeSmartBox(barcode)
        // Process box
    case 'l':
        label, _ := utils.DecodeSmartLabel(barcode)
        // Process label
    case 'p':
        // Location code (Odoo native)
        // Query database or use cached location data
    }
}
```

## Future Enhancements

- [ ] Support for 2D barcodes (QR, DataMatrix) for extended payload
- [ ] CRC checksum for error detection
- [ ] Batch code generation API
- [ ] Mobile scanner app with offline validation
- [ ] Smart Code printer integration

## References

- Base36 Encoding: https://en.wikipedia.org/wiki/Base36
- Odoo 17 Stock Models: https://www.odoo.com/documentation/17.0/developer/reference/backend/orm.html
- Code128 Symbology: https://en.wikipedia.org/wiki/Code_128
