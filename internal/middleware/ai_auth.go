package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/dmytrosurovtsev/eckwmsgo/internal/config"
	"github.com/dmytrosurovtsev/eckwmsgo/internal/database"
	"github.com/dmytrosurovtsev/eckwmsgo/internal/models"
	"github.com/dmytrosurovtsev/eckwmsgo/internal/utils"
)

type aiContextKey string

const AIAgentContextKey aiContextKey = "ai_agent"

// AIAuthMiddleware verifies JWT tokens for AI agents
// Similar to AuthMiddleware but validates against AIAgent table instead of UserAuth
func AIAuthMiddleware(db *database.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header required", http.StatusUnauthorized)
				return
			}

			// Bearer token
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
				return
			}

			tokenString := parts[1]
			cfg, _ := config.Load()

			claims, err := utils.ValidateToken(tokenString, cfg.JWTSecret)
			if err != nil {
				http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
				return
			}

			// Extract agent ID from claims
			agentID, ok := claims["agent_id"].(string)
			if !ok {
				http.Error(w, "Invalid token: missing agent_id", http.StatusUnauthorized)
				return
			}

			// Load AI agent from database
			var agent models.AIAgent
			if err := db.Where("id = ?", agentID).First(&agent).Error; err != nil {
				http.Error(w, "AI agent not found", http.StatusUnauthorized)
				return
			}

			// Check if agent is active
			if !agent.IsActive || agent.Status != "active" {
				http.Error(w, "AI agent is not active", http.StatusForbidden)
				return
			}

			// Add agent to context
			ctx := context.WithValue(r.Context(), AIAgentContextKey, &agent)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetAIAgentFromContext retrieves the AI agent from request context
func GetAIAgentFromContext(ctx context.Context) (*models.AIAgent, bool) {
	agent, ok := ctx.Value(AIAgentContextKey).(*models.AIAgent)
	return agent, ok
}
