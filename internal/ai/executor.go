package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dmytrosurovtsev/eckwmsgo/internal/database"
	"github.com/dmytrosurovtsev/eckwmsgo/internal/models"
	"gorm.io/gorm"
)

// ExecutionContext holds context for AI function execution
type ExecutionContext struct {
	AgentID       string
	FunctionName  string
	Parameters    map[string]interface{}
	IPAddress     string
	UserAgent     string
	RequestTime   time.Time
}

// ExecutionResult holds the result of a function execution
type ExecutionResult struct {
	Success       bool
	Data          interface{}
	Error         string
	ExecutionTime time.Duration
	TokensUsed    int
}

// Executor handles AI function execution with permission checking
type Executor struct {
	db       *database.DB
	registry *FunctionRegistry
}

// NewExecutor creates a new AI function executor
func NewExecutor(db *database.DB) *Executor {
	return &Executor{
		db:       db,
		registry: GetRegistry(),
	}
}

// Execute executes an AI function with full permission checking and auditing
func (e *Executor) Execute(ctx context.Context, execCtx *ExecutionContext) (*ExecutionResult, error) {
	startTime := time.Now()
	result := &ExecutionResult{
		Success: false,
	}

	// 1. Load AI Agent
	agent, err := e.loadAgent(execCtx.AgentID)
	if err != nil {
		result.Error = fmt.Sprintf("agent not found: %v", err)
		e.auditLog(execCtx, result)
		return result, err
	}

	if !agent.IsActive || agent.Status != "active" {
		result.Error = "agent is not active"
		e.auditLog(execCtx, result)
		return result, fmt.Errorf("agent is not active")
	}

	// 2. Check if function exists
	function, err := e.registry.Get(execCtx.FunctionName)
	if err != nil {
		result.Error = fmt.Sprintf("function not found: %v", err)
		e.auditLog(execCtx, result)
		return result, err
	}

	// 3. Check agent access tier
	if !e.checkAccessTier(agent, function) {
		result.Error = "insufficient access tier"
		e.auditLog(execCtx, result)
		return result, fmt.Errorf("insufficient access tier")
	}

	// 4. Check specific permission
	hasPermission, err := e.checkPermission(agent.ID, execCtx.FunctionName)
	if err != nil || !hasPermission {
		result.Error = "permission denied"
		e.auditLog(execCtx, result)
		return result, fmt.Errorf("permission denied")
	}

	// 5. Check rate limits
	if err := e.checkRateLimit(agent, execCtx.FunctionName); err != nil {
		result.Error = fmt.Sprintf("rate limit exceeded: %v", err)
		e.auditLog(execCtx, result)
		return result, err
	}

	// 6. Check if approval is required
	if function.RequireApproval && !e.hasApproval(execCtx) {
		result.Error = "human approval required"
		e.auditLog(execCtx, result)
		return result, fmt.Errorf("human approval required for this operation")
	}

	// 7. Execute the function
	if function.Handler == nil {
		result.Error = "function handler not implemented"
		e.auditLog(execCtx, result)
		return result, fmt.Errorf("function handler not implemented")
	}

	data, execErr := function.Handler(execCtx.Parameters)

	result.ExecutionTime = time.Since(startTime)

	if execErr != nil {
		result.Success = false
		result.Error = execErr.Error()
	} else {
		result.Success = true
		result.Data = data
	}

	// 8. Update rate limits
	e.updateRateLimit(agent.ID, execCtx.FunctionName)

	// 9. Update agent statistics
	e.updateAgentStats(agent.ID, result)

	// 10. Audit log
	e.auditLog(execCtx, result)

	return result, execErr
}

// loadAgent loads an AI agent from the database
func (e *Executor) loadAgent(agentID string) (*models.AIAgent, error) {
	var agent models.AIAgent
	if err := e.db.Where("id = ?", agentID).First(&agent).Error; err != nil {
		return nil, err
	}
	return &agent, nil
}

// checkAccessTier verifies if agent's access tier allows the function category
func (e *Executor) checkAccessTier(agent *models.AIAgent, function *Function) bool {
	tierLevel := map[string]int{
		"workflow": 1,
		"system":   2,
		"admin":    3,
	}

	agentLevel := tierLevel[agent.AccessTier]
	functionLevel := tierLevel[string(function.Category)]

	return agentLevel >= functionLevel
}

// checkPermission checks if agent has specific permission for the function
func (e *Executor) checkPermission(agentID, functionName string) (bool, error) {
	var permission models.AIPermission
	err := e.db.Where("agent_id = ? AND function_name = ? AND is_enabled = ?",
		agentID, functionName, true).First(&permission).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// No specific permission, check if function is in workflow category (auto-allowed)
			function, _ := e.registry.Get(functionName)
			if function != nil && function.Category == CategoryWorkflow {
				return true, nil
			}
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// checkRateLimit verifies if agent is within rate limits
func (e *Executor) checkRateLimit(agent *models.AIAgent, functionName string) error {
	windowStart := time.Now().Truncate(time.Minute)

	var rateLimit models.AIRateLimit
	err := e.db.Where("agent_id = ? AND function_name = ? AND window_start = ?",
		agent.ID, functionName, windowStart).First(&rateLimit).Error

	if err == gorm.ErrRecordNotFound {
		// No limit record yet, OK to proceed
		return nil
	}

	if err != nil {
		return err
	}

	// Check function-specific rate limit
	var permission models.AIPermission
	err = e.db.Where("agent_id = ? AND function_name = ?", agent.ID, functionName).First(&permission).Error

	maxRate := agent.MaxRatePerMin
	if err == nil && permission.MaxRate > 0 {
		maxRate = permission.MaxRate
	}

	if rateLimit.RequestCount >= maxRate {
		return fmt.Errorf("rate limit exceeded: %d requests per minute", maxRate)
	}

	return nil
}

// updateRateLimit updates the rate limit counter
func (e *Executor) updateRateLimit(agentID, functionName string) {
	windowStart := time.Now().Truncate(time.Minute)

	var rateLimit models.AIRateLimit
	err := e.db.Where("agent_id = ? AND function_name = ? AND window_start = ?",
		agentID, functionName, windowStart).First(&rateLimit).Error

	if err == gorm.ErrRecordNotFound {
		// Create new rate limit record
		rateLimit = models.AIRateLimit{
			AgentID:      agentID,
			FunctionName: functionName,
			WindowStart:  windowStart,
			RequestCount: 1,
			UpdatedAt:    time.Now(),
		}
		e.db.Create(&rateLimit)
	} else {
		// Update existing record
		e.db.Model(&rateLimit).Updates(map[string]interface{}{
			"request_count": gorm.Expr("request_count + ?", 1),
			"updated_at":    time.Now(),
		})
	}
}

// updateAgentStats updates agent usage statistics
func (e *Executor) updateAgentStats(agentID string, result *ExecutionResult) {
	updates := map[string]interface{}{
		"total_requests":   gorm.Expr("total_requests + ?", 1),
		"last_request_at":  time.Now(),
	}

	if result.TokensUsed > 0 {
		updates["total_tokens_used"] = gorm.Expr("total_tokens_used + ?", result.TokensUsed)
	}

	e.db.Model(&models.AIAgent{}).Where("id = ?", agentID).Updates(updates)
}

// auditLog creates an audit log entry
func (e *Executor) auditLog(execCtx *ExecutionContext, result *ExecutionResult) {
	requestData, _ := json.Marshal(execCtx.Parameters)
	responseData, _ := json.Marshal(result.Data)

	status := "success"
	if !result.Success {
		status = "failed"
	}

	auditLog := models.AIAuditLog{
		AgentID:       execCtx.AgentID,
		FunctionName:  execCtx.FunctionName,
		RequestData:   string(requestData),
		ResponseData:  string(responseData),
		Status:        status,
		ErrorMessage:  result.Error,
		ExecutionTime: int(result.ExecutionTime.Milliseconds()),
		TokensUsed:    result.TokensUsed,
		IPAddress:     execCtx.IPAddress,
		UserAgent:     execCtx.UserAgent,
		CreatedAt:     time.Now(),
	}

	e.db.Create(&auditLog)
}

// hasApproval checks if the operation has human approval (stub for now)
func (e *Executor) hasApproval(execCtx *ExecutionContext) bool {
	// TODO: Implement approval workflow
	// For now, return false (all approval-required functions will be denied)
	return false
}

// GetAgentPermissions returns all permissions for an agent
func (e *Executor) GetAgentPermissions(agentID string) ([]models.AIPermission, error) {
	var permissions []models.AIPermission
	err := e.db.Where("agent_id = ?", agentID).Find(&permissions).Error
	return permissions, err
}

// GrantPermission grants a permission to an agent
func (e *Executor) GrantPermission(agentID, functionName, scope string, maxRate int) error {
	permission := models.AIPermission{
		AgentID:      agentID,
		FunctionName: functionName,
		Scope:        scope,
		MaxRate:      maxRate,
		IsEnabled:    true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	return e.db.Create(&permission).Error
}

// RevokePermission revokes a permission from an agent
func (e *Executor) RevokePermission(agentID, functionName string) error {
	return e.db.Where("agent_id = ? AND function_name = ?", agentID, functionName).
		Delete(&models.AIPermission{}).Error
}
