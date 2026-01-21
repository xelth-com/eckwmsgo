# Development Guide & Standards

## 🛠 Tech Stack Rules

1. **Single Binary:** Always ensure the frontend is built (`npm run build`) before compiling Go, as it's embedded. Use `make build`.
2. **Database:** Use GORM models. The system auto-migrates on startup.
3. **Paths:** All API paths should be lowercase. Static assets reside in `/i/` (internal).

## 📱 Smart Code Logic

Our system uses 1-character prefixes for instant identification:

| Prefix | Type | Description | Example |
|--------|------|-------------|---------|
| `i` | **Item** | Encoded Serial + EAN | `iDABC1231234567890123` |
| `b` | **Box** | 19-char fixed: dimensions, weight, serial | `b140U0P0FAB000009IX` |
| `p` | **Place/Location** | Odoo-native barcodes | Odoo location barcode |
| `l` | **Label** | System actions/status markers | `l04LATESTPAYLOAD123` |

### Item Code (`i`)
Variable length format: `i` + Serial + split-char + EAN
- Split char indicates length of suffix (Base36 encoded)
- Example: `iDABC1231234567890123` → Serial="ABC123", EAN="1234567890123"

### Box Code (`b`)
19-char fixed format encoding L/W/H, Weight, Type, Serial
- Tiered weight encoding (10g/100g/1kg precision)
- Example: `b140U0P0FAB000009IX` → 40×30×25cm, 5.5kg, Type=Box

### Label Code (`l`)
19-char fixed format for date-stamped actions/markers
- Days since 2025-01-01 epoch
- 14-char payload for custom data

## 🔐 Security & Sync

- **E2EE:** Trusted nodes share `SYNC_NETWORK_KEY`.
- **Relays:** Role `blind_relay` cannot see data, only routes encrypted packets.
- **Mesh:** All inter-node communication is signed via JWT using `MESH_SECRET`.

## 💻 Cross-Platform Notes

### Windows Build
Use `MSYS_NO_PATHCONV=1` when building the frontend to prevent Git Bash from mangling the `BASE_PATH=/E` variable:

```bash
cd web
MSYS_NO_PATHCONV=1 BASE_PATH=/E npm run build
```

### Embedded DB
If `PG_PASSWORD` is empty in `.env`, the system downloads and runs its own PostgreSQL in `./db_data`. No external database required for development.

## 📁 Project Structure

```
eckwmsgo/
├── cmd/
│   └── api/
│       └── main.go              # Application entry point
├── internal/
│   ├── config/
│   │   └── config.go            # Environment configuration
│   ├── database/
│   │   └── database.go          # PostgreSQL connection (GORM)
│   ├── models/
│   │   ├── user.go              # User authentication
│   │   ├── warehouse.go         # Odoo-aligned stock models
│   │   └── ...
│   ├── handlers/
│   │   ├── router.go            # HTTP router
│   │   ├── auth.go              # Authentication
│   │   └── ...
│   ├── services/
│   │   └── delivery/            # OPAL delivery integration
│   └── utils/
│       ├── smart_code.go        # Barcode encoding/decoding
│       └── encryption.go        # AES-192 encryption
├── web/                         # SvelteKit Frontend
│   ├── src/
│   └── build/                   # Embedded static files
├── scripts/
│   ├── setup.sh                 # Dependency installation
│   └── build_release.sh         # Multi-arch build script
└── docs/
    ├── DEVELOPMENT.md           # This file
    ├── ODOO_SYNC_API.md         # Odoo integration details
    └── ZERO_KNOWLEDGE_RELAY.md  # Relay architecture
```

## 🔧 Common Tasks

### Add a new API endpoint

1. Create handler in `internal/handlers/`
2. Register route in `router.go`
3. Add JWT middleware for protected routes
4. Update OpenAPI spec if applicable

### Sync from Odoo

The Odoo sync runs automatically based on `ODOO_SYNC_INTERVAL` (minutes). Trigger manually via:
```bash
# Via API
curl -X POST http://localhost:3210/api/odoo/sync
```

### Import OPAL orders

```bash
# Via API (runs in background)
curl -X POST http://localhost:3210/api/delivery/import/opal
```

## 🚀 Deployment Commands

```bash
# Development
make dev

# Production build (current OS)
make build

# Cross-compile for TSD (Linux ARM64)
make build-arm64

# Clean build artifacts
make clean
```
