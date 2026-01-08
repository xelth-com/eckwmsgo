package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/dmytrosurovtsev/eckwmsgo/internal/models"
)

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

// login handles user login
func (r *Router) login(w http.ResponseWriter, req *http.Request) {
	var loginReq LoginRequest
	if err := json.NewDecoder(req.Body).Decode(&loginReq); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// TODO: Implement authentication logic
	// For now, return a placeholder response
	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Login successful",
		"token":   "placeholder_token",
	})
}

// register handles user registration
func (r *Router) register(w http.ResponseWriter, req *http.Request) {
	var regReq RegisterRequest
	if err := json.NewDecoder(req.Body).Decode(&regReq); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Create new user
	user := models.UserAuth{
		Username: regReq.Username,
		Email:    regReq.Email,
		Password: regReq.Password, // TODO: Hash password
		Role:     "user",
	}

	if err := r.db.Create(&user).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "User registered successfully",
		"user_id": user.ID,
	})
}

// logout handles user logout
func (r *Router) logout(w http.ResponseWriter, req *http.Request) {
	// TODO: Implement logout logic (invalidate token, etc.)
	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Logout successful",
	})
}
