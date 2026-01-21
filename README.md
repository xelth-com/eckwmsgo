# eckWMS Go 🚀

Modern, high-performance Warehouse Management System migrated from Node.js to **Go + SvelteKit**.

## 📍 Current Status: "Portability & Mesh Ready"

The system has completed its core migration. It is now fully portable across Windows/Linux/ARM64 and supports distributed Mesh Network operations.

**Current Focus:** Odoo 17 deep integration (Write-back API).

## 🏗 Architecture

- **Backend:** Go 1.21+ (Gorilla Mux, GORM)
- **Frontend:** SvelteKit (Embedded into binary via `embed`)
- **Database:** PostgreSQL (Hybrid: Auto-switches between External and Embedded)
- **Sync:** End-to-End Encrypted Mesh Sync + Odoo XML-RPC
- **Smart Codes:** Self-contained `i/b/p/l` barcodes for offline ops.

## 🚀 Quick Start (Clone & Run)

If you have Go and Node.js installed:

```bash
# 1. Setup environment and dependencies
make setup

# 2. Configure (Edit .env with your keys)
cp .env.example .env

# 3. Build and Run
make build
./eckwms  # or eckwms.exe on Windows
```

## 🌐 Mesh Network Topology

The system is designed for multi-warehouse deployments:

```
┌─────────────────────────────────────────────────────────────┐
│                    MASTER NODE                              │
│         (Headquarters - pda.repair/E)                       │
│                                                             │
│    ┌───────────────────────────────────────────────────┐    │
│    │  JWT Handshake | Sync Orchestration | Registry   │    │
│    └───────────────────────────────────────────────────┘    │
│                          │                                   │
│           ┌──────────────┼──────────────┐                  │
│           │              │              │                  │
│           ▼              ▼              ▼                  │
│    ┌──────────┐   ┌──────────┐   ┌──────────┐             │
│    │  PEER 1  │   │  PEER 2  │   │  PEER N  │             │
│    │ Warehouse│   │  Store   │   │  Remote  │             │
│    │  (Full)  │   │  (Full)  │   │  (Full)  │             │
│    └──────────┘   └──────────┘   └──────────┘             │
│                                                             │
│              Optional: BLIND RELAY (Encrypted Proxy)        │
└─────────────────────────────────────────────────────────────┘
```

- **Master Node:** Central coordinator (Headquarters).
- **Peer Node:** Trusted warehouse location (Full DB).
- **Relay Node:** Zero-Knowledge proxy (Encrypted routing).

## 📖 Documentation

- [Development Guide](docs/DEVELOPMENT.md) - Technical rules and Smart Code logic.
- [Odoo Integration](docs/ODOO_SYNC_API.md) - How we talk to Odoo.
- [Zero-Knowledge Relay](docs/ZERO_KNOWLEDGE_RELAY.md) - Secure sync details.
- [Smart Codes](docs/SMART_CODES.md) - i/b/p/l barcode specification.

## 🛠 Tech Stack

| Component | Technology |
|-----------|------------|
| Backend | Go 1.21+, Gorilla Mux, GORM |
| Frontend | SvelteKit, TailwindCSS |
| Database | PostgreSQL (External or Embedded) |
| Auth | JWT + Bcrypt |
| Sync | Mesh Network + E2E Encryption |
| Scraper | Node.js + Playwright (OPAL delivery) |
