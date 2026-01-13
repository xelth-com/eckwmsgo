package sync

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/dmytrosurovtsev/eckwmsgo/internal/models"
)

// SecurityLayer handles encryption/decryption of sync packets
type SecurityLayer struct {
	nodeRole     SyncNodeRole
	sharedSecret []byte // 32 bytes for AES-256
	keyID        string
}

// NewSecurityLayer creates a new SecurityLayer based on the node role
func NewSecurityLayer(role SyncNodeRole) *SecurityLayer {
	// Key should be loaded from secure ENV
	secretHex := os.Getenv("SYNC_NETWORK_KEY") // Must be 32 bytes

	// If Relay, we might not have the key, and that's fine
	if role == RoleBlindRelay && secretHex == "" {
		return &SecurityLayer{nodeRole: role}
	}

	return &SecurityLayer{
		nodeRole:     role,
		sharedSecret: []byte(secretHex), // In production, decode hex
		keyID:        "v1",
	}
}

// EncryptPacket takes a plain entity and wraps it into an EncryptedSyncPacket
func (s *SecurityLayer) EncryptPacket(metadata *EntityMetadata, payload interface{}) (*models.EncryptedSyncPacket, error) {
	if s.nodeRole == RoleBlindRelay {
		return nil, fmt.Errorf("blind relay cannot create new encrypted packets (no key)")
	}

	if len(s.sharedSecret) != 32 {
		return nil, fmt.Errorf("invalid key length: must be 32 bytes for AES-256, got %d", len(s.sharedSecret))
	}

	// 1. Serialize Payload
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// 2. Prepare Cipher
	block, err := aes.NewCipher(s.sharedSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// 3. Encrypt
	ciphertext := gcm.Seal(nil, nonce, jsonPayload, nil)

	// 4. Construct Packet (Metadata is copied plain, Payload is encrypted)
	vcBytes, err := json.Marshal(metadata.VectorClock)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal vector clock: %w", err)
	}

	return &models.EncryptedSyncPacket{
		EntityType:       string(metadata.EntityType),
		EntityID:         metadata.EntityID,
		Version:          metadata.Version,
		SourceInstance:   metadata.InstanceID,
		VectorClock:      vcBytes,
		KeyID:            s.keyID,
		Algorithm:        "AES-256-GCM",
		EncryptedPayload: ciphertext,
		Nonce:            nonce,
	}, nil
}

// DecryptPacket takes an encrypted packet and returns the raw payload
func (s *SecurityLayer) DecryptPacket(packet *models.EncryptedSyncPacket, out interface{}) error {
	if s.nodeRole == RoleBlindRelay {
		return fmt.Errorf("blind relay cannot decrypt packets")
	}

	if len(s.sharedSecret) != 32 {
		return fmt.Errorf("invalid key length: must be 32 bytes for AES-256, got %d", len(s.sharedSecret))
	}

	block, err := aes.NewCipher(s.sharedSecret)
	if err != nil {
		return fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("failed to create GCM: %w", err)
	}

	plaintext, err := gcm.Open(nil, packet.Nonce, packet.EncryptedPayload, nil)
	if err != nil {
		return fmt.Errorf("decryption failed: %w", err)
	}

	if err := json.Unmarshal(plaintext, out); err != nil {
		return fmt.Errorf("failed to unmarshal decrypted data: %w", err)
	}

	return nil
}

// CanEncrypt returns true if this node can encrypt data
func (s *SecurityLayer) CanEncrypt() bool {
	return s.nodeRole != RoleBlindRelay && len(s.sharedSecret) == 32
}

// CanDecrypt returns true if this node can decrypt data
func (s *SecurityLayer) CanDecrypt() bool {
	return s.nodeRole != RoleBlindRelay && len(s.sharedSecret) == 32
}

// GetRole returns the current node role
func (s *SecurityLayer) GetRole() SyncNodeRole {
	return s.nodeRole
}
