package ai

import (
	"fmt"
	"sync"
)

// FunctionCategory represents different tiers of AI access
type FunctionCategory string

const (
	// Tier 1: Workflow functions (100% access for AI)
	CategoryWorkflow FunctionCategory = "workflow"

	// Tier 2: System functions (controlled access)
	CategorySystem FunctionCategory = "system"

	// Tier 3: Admin functions (restricted, requires approval)
	CategoryAdmin FunctionCategory = "admin"
)

// Function represents a callable system function for AI
type Function struct {
	Name        string                                  // Fully qualified name (e.g., "orders.create")
	Category    FunctionCategory                        // Access tier
	Description string                                  // Human-readable description
	Handler     func(params map[string]interface{}) (interface{}, error)  // Actual function to execute
	RequireApproval bool                                // Requires human approval?
	RiskLevel   string                                  // low, medium, high, critical
}

// FunctionRegistry manages all AI-callable functions
type FunctionRegistry struct {
	functions map[string]*Function
	mu        sync.RWMutex
}

var (
	registry     *FunctionRegistry
	registryOnce sync.Once
)

// GetRegistry returns the singleton function registry
func GetRegistry() *FunctionRegistry {
	registryOnce.Do(func() {
		registry = &FunctionRegistry{
			functions: make(map[string]*Function),
		}
		registry.registerDefaultFunctions()
	})
	return registry
}

// Register adds a function to the registry
func (r *FunctionRegistry) Register(fn *Function) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.functions[fn.Name]; exists {
		return fmt.Errorf("function %s already registered", fn.Name)
	}

	r.functions[fn.Name] = fn
	return nil
}

// Get retrieves a function from the registry
func (r *FunctionRegistry) Get(name string) (*Function, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	fn, exists := r.functions[name]
	if !exists {
		return nil, fmt.Errorf("function %s not found", name)
	}

	return fn, nil
}

// List returns all registered functions
func (r *FunctionRegistry) List() []*Function {
	r.mu.RLock()
	defer r.mu.RUnlock()

	functions := make([]*Function, 0, len(r.functions))
	for _, fn := range r.functions {
		functions = append(functions, fn)
	}
	return functions
}

// ListByCategory returns functions in a specific category
func (r *FunctionRegistry) ListByCategory(category FunctionCategory) []*Function {
	r.mu.RLock()
	defer r.mu.RUnlock()

	functions := make([]*Function, 0)
	for _, fn := range r.functions {
		if fn.Category == category {
			functions = append(functions, fn)
		}
	}
	return functions
}

// registerDefaultFunctions registers all built-in AI functions
func (r *FunctionRegistry) registerDefaultFunctions() {
	// ==========================================
	// TIER 1: WORKFLOW FUNCTIONS (100% access)
	// ==========================================

	// Orders Management
	r.Register(&Function{
		Name:        "orders.list",
		Category:    CategoryWorkflow,
		Description: "List all orders with optional filters",
		Handler:     nil, // Will be set by handler registration
		RiskLevel:   "low",
	})

	r.Register(&Function{
		Name:        "orders.get",
		Category:    CategoryWorkflow,
		Description: "Get a specific order by ID",
		Handler:     nil,
		RiskLevel:   "low",
	})

	r.Register(&Function{
		Name:        "orders.create",
		Category:    CategoryWorkflow,
		Description: "Create a new order",
		Handler:     nil,
		RiskLevel:   "low",
	})

	r.Register(&Function{
		Name:        "orders.update",
		Category:    CategoryWorkflow,
		Description: "Update an existing order",
		Handler:     nil,
		RiskLevel:   "medium",
	})

	r.Register(&Function{
		Name:        "orders.delete",
		Category:    CategoryWorkflow,
		Description: "Delete an order",
		Handler:     nil,
		RiskLevel:   "medium",
	})

	// Items Management
	r.Register(&Function{
		Name:        "items.list",
		Category:    CategoryWorkflow,
		Description: "List all items with optional filters",
		Handler:     nil,
		RiskLevel:   "low",
	})

	r.Register(&Function{
		Name:        "items.get",
		Category:    CategoryWorkflow,
		Description: "Get a specific item by ID",
		Handler:     nil,
		RiskLevel:   "low",
	})

	r.Register(&Function{
		Name:        "items.create",
		Category:    CategoryWorkflow,
		Description: "Create a new item",
		Handler:     nil,
		RiskLevel:   "low",
	})

	r.Register(&Function{
		Name:        "items.update",
		Category:    CategoryWorkflow,
		Description: "Update an existing item",
		Handler:     nil,
		RiskLevel:   "medium",
	})

	// Warehouse Management
	r.Register(&Function{
		Name:        "warehouse.list",
		Category:    CategoryWorkflow,
		Description: "List all warehouses",
		Handler:     nil,
		RiskLevel:   "low",
	})

	r.Register(&Function{
		Name:        "warehouse.get",
		Category:    CategoryWorkflow,
		Description: "Get warehouse details",
		Handler:     nil,
		RiskLevel:   "low",
	})

	r.Register(&Function{
		Name:        "warehouse.create",
		Category:    CategoryWorkflow,
		Description: "Create a new warehouse",
		Handler:     nil,
		RiskLevel:   "medium",
	})

	// Printing
	r.Register(&Function{
		Name:        "print.labels",
		Category:    CategoryWorkflow,
		Description: "Generate and print labels",
		Handler:     nil,
		RiskLevel:   "low",
	})

	// WebSocket Commands (device control)
	r.Register(&Function{
		Name:        "device.send_command",
		Category:    CategoryWorkflow,
		Description: "Send command to a registered device via WebSocket",
		Handler:     nil,
		RiskLevel:   "medium",
	})

	r.Register(&Function{
		Name:        "device.list",
		Category:    CategoryWorkflow,
		Description: "List all registered devices",
		Handler:     nil,
		RiskLevel:   "low",
	})

	// ==========================================
	// TIER 2: SYSTEM FUNCTIONS (Controlled)
	// ==========================================

	// Network Configuration
	r.Register(&Function{
		Name:            "system.network.get_config",
		Category:        CategorySystem,
		Description:     "Get current network configuration",
		Handler:         nil,
		RiskLevel:       "medium",
		RequireApproval: false,
	})

	r.Register(&Function{
		Name:            "system.network.set_config",
		Category:        CategorySystem,
		Description:     "Update network configuration (IP, DNS, etc)",
		Handler:         nil,
		RiskLevel:       "high",
		RequireApproval: true,
	})

	r.Register(&Function{
		Name:            "system.network.test_connection",
		Category:        CategorySystem,
		Description:     "Test network connectivity",
		Handler:         nil,
		RiskLevel:       "low",
		RequireApproval: false,
	})

	// Printer Management
	r.Register(&Function{
		Name:            "system.printer.list",
		Category:        CategorySystem,
		Description:     "List all available printers",
		Handler:         nil,
		RiskLevel:       "low",
		RequireApproval: false,
	})

	r.Register(&Function{
		Name:            "system.printer.get_status",
		Category:        CategorySystem,
		Description:     "Get printer status (ready, offline, error)",
		Handler:         nil,
		RiskLevel:       "low",
		RequireApproval: false,
	})

	r.Register(&Function{
		Name:            "system.printer.set_default",
		Category:        CategorySystem,
		Description:     "Set default printer",
		Handler:         nil,
		RiskLevel:       "medium",
		RequireApproval: true,
	})

	r.Register(&Function{
		Name:            "system.printer.install_driver",
		Category:        CategorySystem,
		Description:     "Install or update printer driver",
		Handler:         nil,
		RiskLevel:       "high",
		RequireApproval: true,
	})

	// System Information
	r.Register(&Function{
		Name:            "system.info.get",
		Category:        CategorySystem,
		Description:     "Get system information (OS, CPU, memory, disk)",
		Handler:         nil,
		RiskLevel:       "low",
		RequireApproval: false,
	})

	r.Register(&Function{
		Name:            "system.info.get_processes",
		Category:        CategorySystem,
		Description:     "List running processes",
		Handler:         nil,
		RiskLevel:       "medium",
		RequireApproval: false,
	})

	// File Operations (restricted to specific paths)
	r.Register(&Function{
		Name:            "system.file.read",
		Category:        CategorySystem,
		Description:     "Read file contents (restricted paths only)",
		Handler:         nil,
		RiskLevel:       "medium",
		RequireApproval: false,
	})

	r.Register(&Function{
		Name:            "system.file.write",
		Category:        CategorySystem,
		Description:     "Write file contents (restricted paths only)",
		Handler:         nil,
		RiskLevel:       "high",
		RequireApproval: true,
	})

	r.Register(&Function{
		Name:            "system.file.list_dir",
		Category:        CategorySystem,
		Description:     "List directory contents (restricted paths only)",
		Handler:         nil,
		RiskLevel:       "medium",
		RequireApproval: false,
	})

	// Device Management
	r.Register(&Function{
		Name:            "system.device.register",
		Category:        CategorySystem,
		Description:     "Register a new device",
		Handler:         nil,
		RiskLevel:       "high",
		RequireApproval: true,
	})

	r.Register(&Function{
		Name:            "system.device.revoke",
		Category:        CategorySystem,
		Description:     "Revoke device access",
		Handler:         nil,
		RiskLevel:       "high",
		RequireApproval: true,
	})

	// ==========================================
	// TIER 3: ADMIN FUNCTIONS (Highly restricted)
	// ==========================================

	r.Register(&Function{
		Name:            "admin.user.create",
		Category:        CategoryAdmin,
		Description:     "Create a new user account",
		Handler:         nil,
		RiskLevel:       "critical",
		RequireApproval: true,
	})

	r.Register(&Function{
		Name:            "admin.user.delete",
		Category:        CategoryAdmin,
		Description:     "Delete a user account",
		Handler:         nil,
		RiskLevel:       "critical",
		RequireApproval: true,
	})

	r.Register(&Function{
		Name:            "admin.database.backup",
		Category:        CategoryAdmin,
		Description:     "Create database backup",
		Handler:         nil,
		RiskLevel:       "high",
		RequireApproval: true,
	})

	r.Register(&Function{
		Name:            "admin.system.restart",
		Category:        CategoryAdmin,
		Description:     "Restart the system",
		Handler:         nil,
		RiskLevel:       "critical",
		RequireApproval: true,
	})
}
