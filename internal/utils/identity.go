package utils

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ServerIdentity holds the persistent identity of this instance
type ServerIdentity struct {
	InstanceID string `json:"instance_id"`
	PrivateKey string `json:"private_key"` // Base64
	PublicKey  string `json:"public_key"`  // Base64
}

var currentIdentity *ServerIdentity

// GetServerIdentity returns the loaded identity or panics if not initialized
func GetServerIdentity() *ServerIdentity {
	if currentIdentity == nil {
		// Fallback for safety, though LoadOrGenerate should be called in main
		_ = LoadOrGenerateServerIdentity()
	}
	return currentIdentity
}

// LoadOrGenerateServerIdentity ensures the server has a stable identity across restarts.
// It checks ENV vars first, then a local file, and generates new keys if neither exist.
func LoadOrGenerateServerIdentity() error {
	// 1. Check Env Vars (Priority)
	envID := os.Getenv("INSTANCE_ID")
	envPub := os.Getenv("SERVER_PUBLIC_KEY")
	envPriv := os.Getenv("SERVER_PRIVATE_KEY")

	if envID != "" && envPub != "" && envPriv != "" {
		currentIdentity = &ServerIdentity{
			InstanceID: envID,
			PublicKey:  envPub,
			PrivateKey: envPriv,
		}
		return nil
	}

	// 2. Check local persistence file
	configDir := ".eck"
	identityFile := filepath.Join(configDir, "server_identity.json")

	if _, err := os.Stat(identityFile); err == nil {
		data, err := os.ReadFile(identityFile)
		if err == nil {
			var identity ServerIdentity
			if err := json.Unmarshal(data, &identity); err == nil {
				currentIdentity = &identity
				return nil
			}
		}
	}

	// 3. Generate New Identity
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate keys: %w", err)
	}

	// Generate UUID (Simple random string for now)
	uuid := generatePseudoUUID()

	currentIdentity = &ServerIdentity{
		InstanceID: uuid,
		PublicKey:  base64.StdEncoding.EncodeToString(pub),
		PrivateKey: base64.StdEncoding.EncodeToString(priv),
	}

	// Save to file for persistence
	_ = os.MkdirAll(configDir, 0755)
	data, _ := json.MarshalIndent(currentIdentity, "", "  ")
	_ = os.WriteFile(identityFile, data, 0600)

	return nil
}

func generatePseudoUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // Version 4
	b[8] = (b[8] & 0x3f) | 0x80 // Variant is 10
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// GetPublicKeyHex returns the public key as an Uppercase Hex string (for QR codes)
func (s *ServerIdentity) GetPublicKeyHex() (string, error) {
	bytes, err := base64.StdEncoding.DecodeString(s.PublicKey)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// VerifySignature checks an Ed25519 signature
func VerifySignature(publicKeyBase64, message, signatureBase64 string) (bool, error) {
	pubBytes, err := base64.StdEncoding.DecodeString(publicKeyBase64)
	if err != nil {
		return false, fmt.Errorf("invalid public key: %v", err)
	}
	if len(pubBytes) != ed25519.PublicKeySize {
		return false, fmt.Errorf("invalid public key size")
	}

	sigBytes, err := base64.StdEncoding.DecodeString(signatureBase64)
	if err != nil {
		return false, fmt.Errorf("invalid signature: %v", err)
	}

	return ed25519.Verify(pubBytes, []byte(message), sigBytes), nil
}
