package mesh

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/xelth-com/eckwmsgo/internal/config"
)

// GenerateNodeToken creates a JWT token for mesh handshake
func GenerateNodeToken(cfg *config.Config) (string, error) {
	claims := jwt.MapClaims{
		"id":     cfg.InstanceID,
		"role":   string(cfg.NodeRole),
		"url":    cfg.BaseURL,
		"weight": cfg.NodeWeight,
		"type":   "mesh_handshake",
		"exp":    time.Now().Add(time.Hour * 1).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.MeshSecret))
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
