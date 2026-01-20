package mesh

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/xelth-com/eckwmsgo/internal/config"
)

// TokenConfig is a minimal config for generating mesh tokens
type TokenConfig struct {
	InstanceID string
	MeshSecret string
	Role       string
	BaseURL    string
	Weight     int
}

// GenerateNodeToken creates a JWT token for mesh handshake
func GenerateNodeToken(cfg interface{}) (string, error) {
	var instanceID, role, baseURL, meshSecret string
	var weight int

	switch c := cfg.(type) {
	case *config.Config:
		instanceID = c.InstanceID
		role = string(c.NodeRole)
		baseURL = c.BaseURL
		weight = c.NodeWeight
		meshSecret = c.MeshSecret
	case *TokenConfig:
		instanceID = c.InstanceID
		role = c.Role
		baseURL = c.BaseURL
		weight = c.Weight
		meshSecret = c.MeshSecret
	default:
		return "", fmt.Errorf("unsupported config type")
	}

	claims := jwt.MapClaims{
		"id":     instanceID,
		"role":   role,
		"url":    baseURL,
		"weight": weight,
		"type":   "mesh_handshake",
		"exp":    time.Now().Add(time.Hour * 1).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(meshSecret))
}

// ValidateNodeToken validates a mesh handshake token and returns node info
func ValidateNodeToken(tokenString string, secret string) (*NodeInfo, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if claims["type"] != "mesh_handshake" {
			return nil, fmt.Errorf("invalid token type")
		}

		return &NodeInfo{
			InstanceID: claims["id"].(string),
			Role:       claims["role"].(string),
			BaseURL:    claims["url"].(string),
			Weight:     int(claims["weight"].(float64)),
			LastSeen:   time.Now(),
			IsOnline:   true,
		}, nil
	}

	return nil, fmt.Errorf("invalid token")
}
