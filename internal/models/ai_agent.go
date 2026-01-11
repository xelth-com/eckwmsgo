package models

import (
	"time"

	"gorm.io/gorm"
)

// AIAgent represents an AI agent in the system (Gemini, Claude, etc)
type AIAgent struct {
	ID           string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	Name         string         `gorm:"not null" json:"name"`                          // e.g., "Gemini-Main", "Claude-Assistant"
	ModelType    string         `gorm:"not null" json:"model_type"`                    // gemini-3-flash, claude-3, etc
	ModelVersion string         `json:"model_version,omitempty"`                       // 2.0-preview, etc
	APIKey       string         `json:"-"`                                             // Encrypted API key for the model
	Description  string         `json:"description,omitempty"`
	IsActive     bool           `gorm:"default:true" json:"is_active"`
	Status       string         `gorm:"default:'active'" json:"status"`                // active, suspended, terminated

	// Access Control
	AccessTier   string         `gorm:"default:'workflow'" json:"access_tier"`         // workflow, system, admin
	MaxRatePerMin int           `gorm:"default:60" json:"max_rate_per_min"`           // Rate limit: requests per minute
	MaxTokensPerDay int         `gorm:"default:1000000" json:"max_tokens_per_day"`    // Token budget per day

	// Usage Tracking
	TotalRequests     int       `gorm:"default:0" json:"total_requests"`
	TotalTokensUsed   int64     `gorm:"default:0" json:"total_tokens_used"`
	LastRequestAt     *time.Time `json:"last_request_at,omitempty"`

	// Security
	AllowedIPRanges   string    `json:"allowed_ip_ranges,omitempty"`                  // CIDR notation, comma-separated
	RequireApproval   bool      `gorm:"default:false" json:"require_approval"`        // Require human approval for sensitive ops

	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name for AIAgent model
func (AIAgent) TableName() string {
	return "ai_agents"
}

// AIPermission represents a specific permission granted to an AI agent
type AIPermission struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	AgentID     string         `gorm:"not null;index" json:"agent_id"`
	FunctionName string        `gorm:"not null" json:"function_name"`                 // e.g., "orders.create", "system.network.config"
	Scope       string         `gorm:"default:'*'" json:"scope"`                      // Data scope: *, warehouse_id:123, user_id:456
	MaxRate     int            `gorm:"default:10" json:"max_rate"`                    // Per-minute rate limit for this function
	IsEnabled   bool           `gorm:"default:true" json:"is_enabled"`

	// Parameter restrictions (JSON)
	AllowedParams   string      `gorm:"type:jsonb" json:"allowed_params,omitempty"`   // Whitelist of allowed parameters
	DeniedParams    string      `gorm:"type:jsonb" json:"denied_params,omitempty"`    // Blacklist of denied parameters

	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name for AIPermission model
func (AIPermission) TableName() string {
	return "ai_permissions"
}

// AIAuditLog represents a log entry for AI operations
type AIAuditLog struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	AgentID       string     `gorm:"not null;index" json:"agent_id"`
	FunctionName  string     `gorm:"not null;index" json:"function_name"`
	RequestData   string     `gorm:"type:jsonb" json:"request_data,omitempty"`        // Input parameters (JSON)
	ResponseData  string     `gorm:"type:jsonb" json:"response_data,omitempty"`       // Output data (JSON)
	Status        string     `gorm:"default:'success'" json:"status"`                 // success, failed, denied
	ErrorMessage  string     `json:"error_message,omitempty"`
	ExecutionTime int        `json:"execution_time_ms"`                               // Milliseconds
	TokensUsed    int        `json:"tokens_used"`
	IPAddress     string     `json:"ip_address,omitempty"`
	UserAgent     string     `json:"user_agent,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// TableName specifies the table name for AIAuditLog model
func (AIAuditLog) TableName() string {
	return "ai_audit_logs"
}

// AIRateLimit tracks rate limiting for AI agents
type AIRateLimit struct {
	AgentID       string    `gorm:"primaryKey" json:"agent_id"`
	FunctionName  string    `gorm:"primaryKey" json:"function_name"`
	WindowStart   time.Time `gorm:"primaryKey" json:"window_start"`                   // Start of rate limit window (1 minute)
	RequestCount  int       `gorm:"default:0" json:"request_count"`
	TokensUsed    int       `gorm:"default:0" json:"tokens_used"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// TableName specifies the table name for AIRateLimit model
func (AIRateLimit) TableName() string {
	return "ai_rate_limits"
}
