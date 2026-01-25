package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/xelth-com/eckwmsgo/internal/config"
	"github.com/xelth-com/eckwmsgo/internal/models"
	"golang.org/x/crypto/bcrypt"
)

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	return string(bytes), err
}

// CheckPasswordHash compares a password with a hash
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateTokens generates Access and Refresh tokens
func GenerateTokens(user *models.UserAuth, cfg *config.Config) (string, string, error) {
	// Access Token Claims
	claims := jwt.MapClaims{
		"id":       user.ID,
		"email":    user.Email,
		"role":     user.Role,
		"userType": user.UserType,
		"exp":      time.Now().Add(time.Hour * 1).Unix(), // 1 hour expiration
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := token.SignedString([]byte(cfg.JWTSecret))
	if err != nil {
		return "", "", err
	}

	// Refresh Token Claims
	refreshClaims := jwt.MapClaims{
		"id":  user.ID,
		"exp": time.Now().Add(time.Hour * 24 * 90).Unix(), // 90 days
	}
	refreshTokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshToken, err := refreshTokenObj.SignedString([]byte(cfg.JWTSecret))
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

// GenerateInviteToken creates a short-lived token for auto-approving devices
func GenerateInviteToken(cfg *config.Config) (string, error) {
	claims := jwt.MapClaims{
		"type": "invite",
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(time.Hour * 24).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWTSecret))
}

// ValidateToken parses and validates a token
func ValidateToken(tokenString string, secret string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// GenerateAIAgentToken generates a JWT token for an AI agent
func GenerateAIAgentToken(agentID, name, accessTier string) (string, error) {
	cfg, err := config.Load()
	if err != nil {
		return "", err
	}

	// AI Agent Token Claims (longer expiration than user tokens)
	claims := jwt.MapClaims{
		"agent_id":    agentID,
		"name":        name,
		"access_tier": accessTier,
		"type":        "ai_agent",
		"exp":         time.Now().Add(time.Hour * 24 * 365).Unix(), // 1 year expiration
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(cfg.JWTSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
