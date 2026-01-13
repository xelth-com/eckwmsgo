package models

import (
	"time"
)

// EncryptedSyncPacket represents a data packet for Blind Relay servers.
// The relay server can read Metadata to handle routing and versioning,
// but cannot read EncryptedPayload.
type EncryptedSyncPacket struct {
	ID uint `gorm:"primaryKey" json:"id"`

	// Routing Metadata (Visible to Relay)
	EntityType     string `gorm:"index:idx_relay_lookup" json:"entity_type"`
	EntityID       string `gorm:"index:idx_relay_lookup" json:"entity_id"`
	Version        int64  `json:"version"`
	SourceInstance string `json:"source_instance"`
	VectorClock    []byte `gorm:"type:jsonb" json:"vector_clock"` // Raw JSON bytes

	// Security Metadata
	KeyID     string `json:"key_id"`     // To handle key rotation
	Algorithm string `json:"algorithm"`  // e.g., "AES-256-GCM"

	// The Payload (Opaque to Relay)
	EncryptedPayload []byte `gorm:"type:bytea" json:"encrypted_payload"`
	Nonce            []byte `gorm:"type:bytea" json:"nonce"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName specifies the table name
func (EncryptedSyncPacket) TableName() string {
	return "encrypted_sync_packets"
}
