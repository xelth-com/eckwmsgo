# AI System Access Control - Complete Implementation Guide

## 🎯 Overview

This system provides **complete AI access control** for warehouse management operations and system functions. AI agents (Gemini 3 Flash, Claude, Chinese models, etc.) can interact with the system through a secure, permission-based API with full audit logging and rate limiting.

## 📋 Table of Contents

1. [Architecture](#architecture)
2. [Access Tiers](#access-tiers)
3. [Function Registry](#function-registry)
4. [Authentication](#authentication)
5. [API Endpoints](#api-endpoints)
6. [Usage Examples](#usage-examples)
7. [Security Features](#security-features)
8. [Extending the System](#extending-the-system)

---

## 🏗️ Architecture

```
┌─────────────────────────────────────────────┐
│         AI Agent (Gemini/Claude/etc)        │
└───────────────────┬─────────────────────────┘
                    │ JWT Token + Agent ID
┌───────────────────▼─────────────────────────┐
│      AI Agent Authentication Layer           │
│  - Verify Agent ID & Token                   │
│  - Load Agent Permissions                    │
└───────────────────┬─────────────────────────┘
                    │
┌───────────────────▼─────────────────────────┐
│      Permission Validator                    │
│  - Check function whitelist                  │
│  - Validate parameters                       │
│  - Check rate limits                         │
└───────────────────┬─────────────────────────┘
                    │
┌───────────────────▼─────────────────────────┐
│      Function Registry & Executor            │
│  Tier 1 (Workflow - 100% access)            │
│  Tier 2 (System - Controlled access)        │
│  Tier 3 (Admin - Highly restricted)         │
└───────────────────┬─────────────────────────┘
                    │
┌───────────────────▼─────────────────────────┐
│         Audit Logger                         │
│  - Log all AI operations                     │
│  - Track token usage                         │
│  - Monitor anomalies                         │
└─────────────────────────────────────────────┘
```

### Database Schema

#### AIAgent Table
```sql
CREATE TABLE ai_agents (
    id UUID PRIMARY KEY,
    name VARCHAR NOT NULL,
    model_type VARCHAR NOT NULL,        -- gemini-3-flash, claude-3, etc
    model_version VARCHAR,
    api_key VARCHAR NOT NULL,           -- Encrypted API key
    description TEXT,
    is_active BOOLEAN DEFAULT true,
    status VARCHAR DEFAULT 'active',
    access_tier VARCHAR DEFAULT 'workflow',  -- workflow, system, admin
    max_rate_per_min INT DEFAULT 60,
    max_tokens_per_day INT DEFAULT 1000000,
    total_requests INT DEFAULT 0,
    total_tokens_used BIGINT DEFAULT 0,
    last_request_at TIMESTAMP,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);
```

#### AIPermission Table
```sql
CREATE TABLE ai_permissions (
    id SERIAL PRIMARY KEY,
    agent_id UUID NOT NULL,
    function_name VARCHAR NOT NULL,      -- e.g., "orders.create", "system.network.config"
    scope VARCHAR DEFAULT '*',           -- Data scope filter
    max_rate INT DEFAULT 10,            -- Per-minute rate limit for this function
    is_enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);
```

#### AIAuditLog Table
```sql
CREATE TABLE ai_audit_logs (
    id SERIAL PRIMARY KEY,
    agent_id UUID NOT NULL,
    function_name VARCHAR NOT NULL,
    request_data JSONB,                 -- Input parameters
    response_data JSONB,                -- Output data
    status VARCHAR DEFAULT 'success',   -- success, failed, denied
    error_message TEXT,
    execution_time_ms INT,
    tokens_used INT,
    ip_address VARCHAR,
    created_at TIMESTAMP
);
```

#### AIRateLimit Table
```sql
CREATE TABLE ai_rate_limits (
    agent_id UUID NOT NULL,
    function_name VARCHAR NOT NULL,
    window_start TIMESTAMP NOT NULL,    -- 1-minute window
    request_count INT DEFAULT 0,
    tokens_used INT DEFAULT 0,
    updated_at TIMESTAMP,
    PRIMARY KEY (agent_id, function_name, window_start)
);
```

---

## 🎚️ Access Tiers

### Tier 1: Workflow (100% Access)
AI has **full autonomous access** to all business logic operations:

- **Orders Management**: Create, read, update, delete orders
- **Items Management**: Create, read, update items and inventory
- **Warehouse Operations**: Manage warehouses, racks, and locations
- **Printing**: Generate labels and QR codes
- **Device Control**: Send commands to registered devices via WebSocket

**No human approval required** for workflow operations.

### Tier 2: System (Controlled Access)
AI has **controlled access** to system functions with optional approval:

- **Network Configuration**: Read config (no approval), modify config (requires approval)
- **Printer Management**: List printers (no approval), install drivers (requires approval)
- **System Information**: Read CPU, memory, disk usage
- **File Operations**: Read/write files in restricted paths only
- **Device Management**: Register or revoke devices (requires approval)

**Human approval required** for high-risk operations.

### Tier 3: Admin (Highly Restricted)
AI has **highly restricted access** to administrative functions:

- **User Management**: Create/delete user accounts (requires approval)
- **Database Operations**: Backup database (requires approval)
- **System Control**: Restart system (requires approval)

**Always requires human approval** for admin operations.

---

## 📚 Function Registry

All AI-callable functions are registered in `/internal/ai/registry.go`:

```go
// Example function registration
registry.Register(&Function{
    Name:            "orders.create",
    Category:        CategoryWorkflow,
    Description:     "Create a new order",
    Handler:         orderCreateHandler,
    RiskLevel:       "low",
    RequireApproval: false,
})

registry.Register(&Function{
    Name:            "system.network.set_config",
    Category:        CategorySystem,
    Description:     "Update network configuration",
    Handler:         networkConfigHandler,
    RiskLevel:       "high",
    RequireApproval: true,
})
```

### Available Functions (Initial Set)

#### Workflow Functions
- `orders.list`, `orders.get`, `orders.create`, `orders.update`, `orders.delete`
- `items.list`, `items.get`, `items.create`, `items.update`
- `warehouse.list`, `warehouse.get`, `warehouse.create`
- `print.labels`
- `device.send_command`, `device.list`

#### System Functions
- `system.network.get_config`, `system.network.set_config`, `system.network.test_connection`
- `system.printer.list`, `system.printer.get_status`, `system.printer.set_default`, `system.printer.install_driver`
- `system.info.get`, `system.info.get_processes`
- `system.file.read`, `system.file.write`, `system.file.list_dir`
- `system.device.register`, `system.device.revoke`

#### Admin Functions
- `admin.user.create`, `admin.user.delete`
- `admin.database.backup`
- `admin.system.restart`

---

## 🔐 Authentication

### Creating an AI Agent

**Admin Endpoint**: `POST /api/admin/ai/agents`

```json
{
  "name": "Gemini-Main",
  "model_type": "gemini-3-flash-preview",
  "model_version": "2.0",
  "api_key": "YOUR_GEMINI_API_KEY",
  "description": "Main AI assistant for warehouse operations",
  "access_tier": "system",
  "max_rate_per_min": 100,
  "max_tokens_per_day": 5000000
}
```

**Response**:
```json
{
  "agent": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "Gemini-Main",
    "model_type": "gemini-3-flash-preview",
    "access_tier": "system",
    "status": "active"
  },
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

### Agent Authentication

AI agents use JWT tokens in the `Authorization` header:

```http
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

Token contains:
- `agent_id`: Unique agent identifier
- `name`: Agent name
- `access_tier`: workflow, system, or admin
- `type`: "ai_agent"
- `exp`: Expiration (1 year)

---

## 🌐 API Endpoints

### AI Agent Endpoints (Protected with AI Auth)

#### Execute Function
```http
POST /api/ai/execute
Content-Type: application/json
Authorization: Bearer <ai_agent_token>

{
  "function_name": "orders.create",
  "parameters": {
    "order_type": "rma",
    "customer_email": "customer@example.com",
    "items": [
      {"sku": "ITEM001", "quantity": 2}
    ]
  }
}
```

**Response**:
```json
{
  "success": true,
  "data": {
    "order_id": "ORD-12345",
    "status": "pending"
  },
  "execution_time_ms": 123
}
```

#### List Available Functions
```http
GET /api/ai/functions
Authorization: Bearer <ai_agent_token>
```

**Response**:
```json
{
  "functions": [
    {
      "name": "orders.create",
      "category": "workflow",
      "description": "Create a new order",
      "risk_level": "low",
      "require_approval": false
    }
  ],
  "count": 25
}
```

#### Get Agent Status
```http
GET /api/ai/status
Authorization: Bearer <ai_agent_token>
```

**Response**:
```json
{
  "agent_id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Gemini-Main",
  "model_type": "gemini-3-flash-preview",
  "access_tier": "system",
  "status": "active",
  "total_requests": 12345,
  "total_tokens_used": 500000,
  "last_request_at": "2026-01-11T10:30:00Z"
}
```

#### Get Permissions
```http
GET /api/ai/permissions
Authorization: Bearer <ai_agent_token>
```

### Admin Endpoints (Protected with User Auth)

#### List All AI Agents
```http
GET /api/admin/ai/agents
Authorization: Bearer <user_token>
```

#### Create AI Agent
```http
POST /api/admin/ai/agents
Authorization: Bearer <user_token>
```

#### Update Agent Status
```http
PUT /api/admin/ai/agents/{agent_id}/status
Authorization: Bearer <user_token>

{
  "is_active": true,
  "status": "active"
}
```

#### Grant Permission
```http
POST /api/admin/ai/agents/{agent_id}/permissions
Authorization: Bearer <user_token>

{
  "function_name": "system.network.set_config",
  "scope": "*",
  "max_rate": 5
}
```

#### Revoke Permission
```http
DELETE /api/admin/ai/agents/{agent_id}/permissions
Authorization: Bearer <user_token>

{
  "function_name": "system.network.set_config"
}
```

#### Get Audit Logs
```http
GET /api/admin/ai/agents/{agent_id}/audit
Authorization: Bearer <user_token>
```

---

## 💡 Usage Examples

### Example 1: Gemini Agent Creating an Order

```go
package main

import (
    "bytes"
    "encoding/json"
    "net/http"
)

func main() {
    // AI agent token (obtained from admin)
    agentToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

    // Prepare function execution request
    req := map[string]interface{}{
        "function_name": "orders.create",
        "parameters": map[string]interface{}{
            "order_type":     "rma",
            "customer_email": "customer@example.com",
            "items": []map[string]interface{}{
                {"sku": "ITEM001", "quantity": 2},
            },
        },
    }

    jsonData, _ := json.Marshal(req)

    // Send request to AI API
    httpReq, _ := http.NewRequest("POST", "http://localhost:3001/api/ai/execute", bytes.NewBuffer(jsonData))
    httpReq.Header.Set("Content-Type", "application/json")
    httpReq.Header.Set("Authorization", "Bearer "+agentToken)

    client := &http.Client{}
    resp, _ := client.Do(httpReq)
    defer resp.Body.Close()

    // Process response
    var result map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&result)

    println("Order created:", result["data"])
}
```

### Example 2: Using Gemini Client

```go
package main

import (
    "fmt"
    "github.com/dmytrosurovtsev/eckwmsgo/internal/ai"
)

func main() {
    // Create Gemini client
    geminiAPIKey := "YOUR_GEMINI_API_KEY"
    client := ai.NewGeminiClient(geminiAPIKey, "gemini-3-flash-preview")

    // Send prompt to Gemini
    prompt := "Create an RMA order for customer john@example.com with 2 units of SKU-12345"
    response, err := client.GenerateContent(prompt)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    fmt.Printf("Gemini response: %s\n", response)

    // Parse response and extract function call
    // Then execute via AI API:
    // POST /api/ai/execute with function_name="orders.create"
}
```

### Example 3: AI System Info Query

```javascript
// From frontend or external service
const agentToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...";

// AI wants to check system info
const response = await fetch("http://localhost:3001/api/ai/execute", {
  method: "POST",
  headers: {
    "Content-Type": "application/json",
    "Authorization": `Bearer ${agentToken}`
  },
  body: JSON.stringify({
    function_name: "system.info.get",
    parameters: {}
  })
});

const result = await response.json();
console.log("System info:", result.data);
// Output: { cpu: "Intel i7", memory: "16GB", disk: "500GB", os: "Linux" }
```

---

## 🛡️ Security Features

### 1. Multi-Layer Permission Checking

```go
// Execution flow in executor.go
1. Load AI Agent → Verify active status
2. Check function exists in registry
3. Verify agent access tier matches function category
4. Check specific permission for function
5. Validate rate limits (per-minute, per-day)
6. Check if human approval required
7. Execute function with audit logging
```

### 2. Rate Limiting

Two levels of rate limiting:
- **Agent-level**: Max requests per minute (e.g., 60 req/min)
- **Function-level**: Per-function limits (e.g., 5 req/min for `system.network.set_config`)

Rate limits are tracked in 1-minute windows and automatically reset.

### 3. Audit Logging

All AI operations are logged with:
- Agent ID and function name
- Input parameters and output data
- Execution time and token usage
- Success/failure status
- IP address and timestamp

Audit logs are queryable via admin API for compliance and monitoring.

### 4. API Key Encryption

AI model API keys are encrypted using AES-192-GCM before storage:
```go
encryptedAPIKey, err := utils.EncryptString(apiKey)
```

### 5. Approval Workflow

High-risk operations require human approval:
```go
function.RequireApproval = true
```

When approval is required, the operation is logged but not executed until manually approved (future enhancement).

### 6. Scope Filtering

Permissions can be scoped to specific resources:
```json
{
  "function_name": "orders.update",
  "scope": "warehouse_id:123"
}
```

This restricts the AI to only operate on orders from warehouse 123.

---

## 🚀 Extending the System

### Adding New Functions

1. **Register function in registry.go**:
```go
registry.Register(&Function{
    Name:        "custom.my_function",
    Category:    CategorySystem,
    Description: "My custom function",
    Handler:     myCustomHandler,
    RiskLevel:   "medium",
})
```

2. **Implement handler**:
```go
func myCustomHandler(params map[string]interface{}) (interface{}, error) {
    // Your function logic
    return result, nil
}
```

3. **Grant permission to AI agent**:
```http
POST /api/admin/ai/agents/{agent_id}/permissions
{
  "function_name": "custom.my_function",
  "scope": "*",
  "max_rate": 10
}
```

### Adding New AI Model Support

To add support for Claude, Chinese models, etc.:

1. **Create client** (similar to `gemini_client.go`):
```go
type ClaudeClient struct {
    APIKey string
    Model  string
}

func (c *ClaudeClient) GenerateContent(prompt string) (string, error) {
    // Claude API integration
}
```

2. **Register agent with new model type**:
```http
POST /api/admin/ai/agents
{
  "name": "Claude-Assistant",
  "model_type": "claude-3-opus",
  "api_key": "CLAUDE_API_KEY",
  "access_tier": "workflow"
}
```

### Adding System Functions

For network, printer, or other system operations:

1. **Implement system adapter**:
```go
package system

func GetNetworkConfig() (map[string]interface{}, error) {
    // Read network configuration from OS
}

func SetNetworkConfig(config map[string]interface{}) error {
    // Update network configuration
}
```

2. **Register in function registry**:
```go
registry.Register(&Function{
    Name:            "system.network.set_config",
    Handler:         networkSetConfigWrapper,
    RequireApproval: true,
})
```

3. **Link handler to system adapter**:
```go
func networkSetConfigWrapper(params map[string]interface{}) (interface{}, error) {
    return nil, system.SetNetworkConfig(params)
}
```

---

## 📊 Monitoring & Analytics

### Audit Log Query Examples

**Get all AI operations in last 24 hours**:
```sql
SELECT * FROM ai_audit_logs
WHERE created_at > NOW() - INTERVAL '24 hours'
ORDER BY created_at DESC;
```

**Get failed operations**:
```sql
SELECT * FROM ai_audit_logs
WHERE status = 'failed'
ORDER BY created_at DESC
LIMIT 100;
```

**Get high-risk operations requiring approval**:
```sql
SELECT a.*, f.risk_level
FROM ai_audit_logs a
JOIN function_registry f ON a.function_name = f.name
WHERE f.require_approval = true
ORDER BY created_at DESC;
```

### Rate Limit Monitoring

```sql
SELECT agent_id, function_name, SUM(request_count) as total_requests
FROM ai_rate_limits
WHERE window_start > NOW() - INTERVAL '1 hour'
GROUP BY agent_id, function_name
ORDER BY total_requests DESC;
```

---

## 🎓 Best Practices

1. **Start with Workflow Tier**: New AI agents should start with `access_tier: "workflow"` and be gradually promoted.

2. **Use Function-Specific Rate Limits**: Set lower rate limits for high-risk functions.

3. **Monitor Audit Logs**: Regularly review audit logs for anomalous behavior.

4. **Rotate API Keys**: Periodically update AI model API keys.

5. **Test in Staging**: Always test new AI agents in staging environment first.

6. **Set Token Budgets**: Use `max_tokens_per_day` to prevent excessive API costs.

7. **Scope Permissions**: Use scope filtering to limit AI access to specific resources.

8. **Enable Approval for Critical Ops**: Always require approval for destructive operations.

---

## 📝 Configuration

### Environment Variables

```bash
# Database
DATABASE_URL=postgresql://user:pass@host:port/dbname
ENC_KEY=48_hex_characters_for_AES192_encryption

# JWT Secret
JWT_SECRET=your_jwt_secret_key

# AI Model API Keys (optional, stored in database)
OPENAI_API_KEY=sk-...
GEMINI_API_KEY=...
```

### Access Tier Configuration

Edit `/internal/models/ai_agent.go` to change defaults:
```go
AccessTier:      "workflow",  // Default tier for new agents
MaxRatePerMin:   60,          // Default rate limit
MaxTokensPerDay: 1000000,     // Default daily token budget
```

---

## 🐛 Troubleshooting

### "Permission denied" error
- Check agent's `access_tier` matches function's `category`
- Verify specific permission exists in `ai_permissions` table
- Ensure `is_enabled = true` for the permission

### "Rate limit exceeded" error
- Check current rate limit window in `ai_rate_limits` table
- Increase `max_rate_per_min` or wait for window to reset
- Adjust function-specific rate limit in permission

### "Agent not found" error
- Verify JWT token is valid and not expired
- Check agent exists in `ai_agents` table
- Ensure `is_active = true` and `status = 'active'`

### "Function not implemented" error
- Function exists in registry but handler is `nil`
- Implement handler function and link it to registry

---

## 📚 Additional Resources

- **API Documentation**: See `/docs/API.md`
- **Database Schema**: See `/docs/DATABASE.md`
- **Gemini API Docs**: https://ai.google.dev/docs
- **Security Guide**: See `/docs/SECURITY.md`

---

**Version**: 1.0.0
**Last Updated**: 2026-01-11
**Maintainer**: System Architecture Team
