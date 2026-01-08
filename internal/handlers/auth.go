package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/dmytrosurovtsev/eckwmsgo/internal/config"
	"github.com/dmytrosurovtsev/eckwmsgo/internal/models"
	"github.com/dmytrosurovtsev/eckwmsgo/internal/utils"
)

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Company  string `json:"company"`
}

// login handles user login
func (r *Router) login(w http.ResponseWriter, req *http.Request) {
	var loginReq LoginRequest
	if err := json.NewDecoder(req.Body).Decode(&loginReq); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// 1. Find User
	var user models.UserAuth
	if err := r.db.Where("email = ?", loginReq.Email).First(&user).Error; err != nil {
		respondError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// 2. Check Password
	if !utils.CheckPasswordHash(loginReq.Password, user.Password) {
		respondError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// 3. Update Last Login
	now := time.Now()
	user.LastLogin = &now
	r.db.Save(&user)

	// 4. Generate Tokens
	cfg, _ := config.Load()
	accessToken, refreshToken, err := utils.GenerateTokens(&user, cfg)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate tokens")
		return
	}

	// 5. Respond matching Node.js structure
	response := map[string]interface{}{
		"tokens": map[string]string{
			"accessToken":  accessToken,
			"refreshToken": refreshToken,
		},
		"user": user,
	}

	respondJSON(w, http.StatusOK, response)
}

// register handles user registration
func (r *Router) register(w http.ResponseWriter, req *http.Request) {
	var regReq RegisterRequest
	if err := json.NewDecoder(req.Body).Decode(&regReq); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// 1. Hash Password
	hashedPassword, err := utils.HashPassword(regReq.Password)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	// 2. Create User
	user := models.UserAuth{
		Username: regReq.Username,
		Email:    regReq.Email,
		Password: hashedPassword,
		Name:     regReq.Name,
		Company:  regReq.Company,
		Role:     "user",
		UserType: "individual",
	}

	if regReq.Company != "" {
		user.UserType = "company"
	}

	if err := r.db.Create(&user).Error; err != nil {
		respondError(w, http.StatusBadRequest, "Failed to create user (email or username might exist)")
		return
	}

	// 3. Generate Tokens for immediate login
	cfg, _ := config.Load()
	accessToken, refreshToken, err := utils.GenerateTokens(&user, cfg)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "User created but failed to generate tokens")
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "User registered successfully",
		"tokens": map[string]string{
			"accessToken":  accessToken,
			"refreshToken": refreshToken,
		},
		"user": user,
	})
}

// logout handles user logout
func (r *Router) logout(w http.ResponseWriter, req *http.Request) {
	// Client-side mostly, but we can handle cookie clearing here if needed
	respondJSON(w, http.StatusOK, map[string]string{"message": "Logged out successfully"})
}
