package utils

import (
	"testing"

	"github.com/xelth-com/eckwmsgo/internal/config"
	"github.com/xelth-com/eckwmsgo/internal/models"
)

func TestPasswordHashing(t *testing.T) {
	password := "secret123"

	// Test Hashing
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}
	if hash == password {
		t.Error("Hash should not match plaintext password")
	}
	if len(hash) == 0 {
		t.Error("Hash should not be empty")
	}

	// Test Comparison (Success)
	if !CheckPasswordHash(password, hash) {
		t.Error("Password should match hash")
	}

	// Test Comparison (Failure)
	if CheckPasswordHash("wrongpassword", hash) {
		t.Error("Wrong password should not match hash")
	}
}

func TestJWT(t *testing.T) {
	// Setup Mock Config
	cfg := &config.Config{
		JWTSecret: "test-secret-key-12345",
	}

	user := &models.UserAuth{
		ID:       "uuid-1234",
		Email:    "test@example.com",
		Role:     "admin",
		UserType: "company",
	}

	// Test Generation
	accessToken, refreshToken, err := GenerateTokens(user, cfg)
	if err != nil {
		t.Fatalf("Failed to generate tokens: %v", err)
	}
	if accessToken == "" || refreshToken == "" {
		t.Error("Tokens should not be empty")
	}

	// Test Validation (Success)
	claims, err := ValidateToken(accessToken, cfg.JWTSecret)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	if claims["id"] != user.ID {
		t.Errorf("Expected user ID %s, got %v", user.ID, claims["id"])
	}
	if claims["email"] != user.Email {
		t.Errorf("Expected email %s, got %v", user.Email, claims["email"])
	}

	// Test Validation (Failure - Wrong Key)
	_, err = ValidateToken(accessToken, "wrong-key")
	if err == nil {
		t.Error("Validation should fail with wrong key")
	}
}
