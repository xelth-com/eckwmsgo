package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/xelth-com/eckwmsgo/internal/ai"
	"github.com/xelth-com/eckwmsgo/internal/middleware"
	"github.com/xelth-com/eckwmsgo/internal/models"
	"github.com/xelth-com/eckwmsgo/internal/utils"
	"github.com/gorilla/mux"
)

// ExecuteFunctionRequest represents a request to execute an AI function
type ExecuteFunctionRequest struct {
	FunctionName string                 `json:"function_name"`
	Parameters   map[string]interface{} `json:"parameters"`
}

// ExecuteFunctionResponse represents the response from function execution
type ExecuteFunctionResponse struct {
	Success       bool        `json:"success"`
	Data          interface{} `json:"data,omitempty"`
	Error         string      `json:"error,omitempty"`
	ExecutionTime int64       `json:"execution_time_ms"`
}

// executeAIFunction handles AI function execution requests
func (rt *Router) executeAIFunction(w http.ResponseWriter, r *http.Request) {
	// Get AI agent from context
	agent, ok := middleware.GetAIAgentFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "AI agent not found in context")
		return
	}

	// Parse request
	var req ExecuteFunctionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Create executor
	executor := ai.NewExecutor(rt.db)

	// Build execution context
	execCtx := &ai.ExecutionContext{
		AgentID:      agent.ID,
		FunctionName: req.FunctionName,
		Parameters:   req.Parameters,
		IPAddress:    r.RemoteAddr,
		UserAgent:    r.Header.Get("User-Agent"),
		RequestTime:  time.Now(),
	}

	// Execute function
	result, _ := executor.Execute(r.Context(), execCtx)

	// Build response
	response := ExecuteFunctionResponse{
		Success:       result.Success,
		Data:          result.Data,
		Error:         result.Error,
		ExecutionTime: result.ExecutionTime.Milliseconds(),
	}

	if result.Success {
		respondJSON(w, http.StatusOK, response)
	} else {
		respondJSON(w, http.StatusBadRequest, response)
	}
}

// listAIFunctions lists all available functions for the AI agent
func (rt *Router) listAIFunctions(w http.ResponseWriter, r *http.Request) {
	agent, ok := middleware.GetAIAgentFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "AI agent not found in context")
		return
	}

	registry := ai.GetRegistry()
	allFunctions := registry.List()

	// Filter functions based on agent's access tier
	tierLevel := map[string]int{
		"workflow": 1,
		"system":   2,
		"admin":    3,
	}
	agentLevel := tierLevel[agent.AccessTier]

	availableFunctions := make([]map[string]interface{}, 0)
	for _, fn := range allFunctions {
		functionLevel := tierLevel[string(fn.Category)]
		if agentLevel >= functionLevel {
			availableFunctions = append(availableFunctions, map[string]interface{}{
				"name":             fn.Name,
				"category":         fn.Category,
				"description":      fn.Description,
				"risk_level":       fn.RiskLevel,
				"require_approval": fn.RequireApproval,
			})
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"functions": availableFunctions,
		"count":     len(availableFunctions),
	})
}

// getAIAgentStatus returns the current status of the AI agent
func (rt *Router) getAIAgentStatus(w http.ResponseWriter, r *http.Request) {
	agent, ok := middleware.GetAIAgentFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "AI agent not found in context")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"agent_id":           agent.ID,
		"name":               agent.Name,
		"model_type":         agent.ModelType,
		"access_tier":        agent.AccessTier,
		"status":             agent.Status,
		"total_requests":     agent.TotalRequests,
		"total_tokens_used":  agent.TotalTokensUsed,
		"last_request_at":    agent.LastRequestAt,
		"max_rate_per_min":   agent.MaxRatePerMin,
		"max_tokens_per_day": agent.MaxTokensPerDay,
	})
}

// getAIAgentPermissions returns all permissions for the AI agent
func (rt *Router) getAIAgentPermissions(w http.ResponseWriter, r *http.Request) {
	agent, ok := middleware.GetAIAgentFromContext(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "AI agent not found in context")
		return
	}

	executor := ai.NewExecutor(rt.db)
	permissions, err := executor.GetAgentPermissions(agent.ID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to load permissions")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"permissions": permissions,
		"count":       len(permissions),
	})
}

// getAIAuditLogs returns audit logs for the AI agent (admin only)
func (rt *Router) getAIAuditLogs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["agent_id"]

	var logs []models.AIAuditLog
	query := rt.db.Where("agent_id = ?", agentID).Order("created_at DESC").Limit(100)

	if err := query.Find(&logs).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to load audit logs")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"logs":  logs,
		"count": len(logs),
	})
}

// Admin endpoints for managing AI agents

// CreateAIAgentRequest represents a request to create a new AI agent
type CreateAIAgentRequest struct {
	Name            string `json:"name"`
	ModelType       string `json:"model_type"`
	ModelVersion    string `json:"model_version,omitempty"`
	APIKey          string `json:"api_key"`
	Description     string `json:"description,omitempty"`
	AccessTier      string `json:"access_tier"`
	MaxRatePerMin   int    `json:"max_rate_per_min,omitempty"`
	MaxTokensPerDay int    `json:"max_tokens_per_day,omitempty"`
}

// createAIAgent creates a new AI agent (admin only)
func (rt *Router) createAIAgent(w http.ResponseWriter, r *http.Request) {
	var req CreateAIAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate access tier
	if req.AccessTier != "workflow" && req.AccessTier != "system" && req.AccessTier != "admin" {
		respondError(w, http.StatusBadRequest, "Invalid access tier")
		return
	}

	// Set defaults
	if req.MaxRatePerMin == 0 {
		req.MaxRatePerMin = 60
	}
	if req.MaxTokensPerDay == 0 {
		req.MaxTokensPerDay = 1000000
	}

	// Encrypt API key
	encryptedAPIKey, err := utils.EncryptString(req.APIKey)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to encrypt API key")
		return
	}

	agent := models.AIAgent{
		Name:            req.Name,
		ModelType:       req.ModelType,
		ModelVersion:    req.ModelVersion,
		APIKey:          encryptedAPIKey,
		Description:     req.Description,
		IsActive:        true,
		Status:          "active",
		AccessTier:      req.AccessTier,
		MaxRatePerMin:   req.MaxRatePerMin,
		MaxTokensPerDay: req.MaxTokensPerDay,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := rt.db.Create(&agent).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create AI agent")
		return
	}

	// Generate JWT token for the agent
	token, err := utils.GenerateAIAgentToken(agent.ID, agent.Name, agent.AccessTier)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"agent": agent,
		"token": token,
	})
}

// listAIAgents lists all AI agents (admin only)
func (rt *Router) listAIAgents(w http.ResponseWriter, r *http.Request) {
	var agents []models.AIAgent
	if err := rt.db.Find(&agents).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to load AI agents")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"agents": agents,
		"count":  len(agents),
	})
}

// updateAIAgentStatus updates the status of an AI agent (admin only)
func (rt *Router) updateAIAgentStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["agent_id"]

	var req struct {
		IsActive bool   `json:"is_active"`
		Status   string `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := rt.db.Model(&models.AIAgent{}).Where("id = ?", agentID).Updates(map[string]interface{}{
		"is_active": req.IsActive,
		"status":    req.Status,
		"updated_at": time.Now(),
	}).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update AI agent")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "AI agent updated successfully",
	})
}

// grantAIPermission grants a permission to an AI agent (admin only)
func (rt *Router) grantAIPermission(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["agent_id"]

	var req struct {
		FunctionName string `json:"function_name"`
		Scope        string `json:"scope"`
		MaxRate      int    `json:"max_rate"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	executor := ai.NewExecutor(rt.db)
	if err := executor.GrantPermission(agentID, req.FunctionName, req.Scope, req.MaxRate); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to grant permission")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Permission granted successfully",
	})
}

// revokeAIPermission revokes a permission from an AI agent (admin only)
func (rt *Router) revokeAIPermission(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["agent_id"]

	var req struct {
		FunctionName string `json:"function_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	executor := ai.NewExecutor(rt.db)
	if err := executor.RevokePermission(agentID, req.FunctionName); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to revoke permission")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Permission revoked successfully",
	})
}
