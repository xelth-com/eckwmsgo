package models

import (
	"time"

	"gorm.io/gorm"
)

// DeviceStatus defines the authorization state of a device
type DeviceStatus string

const (
	DeviceStatusPending DeviceStatus = "pending" // Initial state, waiting for admin approval
	DeviceStatusActive  DeviceStatus = "active"  // Authorized to work
	DeviceStatusBlocked DeviceStatus = "blocked" // Explicitly banned
)

// RegisteredDevice represents a PDA/Scanner that has initiated a handshake
type RegisteredDevice struct {
	DeviceID   string         `gorm:"column:deviceId;primaryKey" json:"deviceId"`
	Name       string         `gorm:"column:deviceName" json:"name"`
	PublicKey  string         `gorm:"column:publicKey;not null" json:"publicKey"` // Base64 encoded Ed25519 public key
	Status     DeviceStatus   `gorm:"default:'pending'" json:"status"`
	LastSeenAt time.Time      `gorm:"column:lastSeenAt" json:"lastSeenAt"`
	CreatedAt  time.Time      `gorm:"column:createdAt" json:"createdAt"`
	UpdatedAt  time.Time      `gorm:"column:updatedAt" json:"updatedAt"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name for RegisteredDevice
func (RegisteredDevice) TableName() string {
	return "registered_devices"
}

// GetEntityID implements SyncableEntity interface for Checksum Engine
func (d RegisteredDevice) GetEntityID() string {
	return d.DeviceID
}

// GetEntityType implements SyncableEntity interface for Checksum Engine
func (d RegisteredDevice) GetEntityType() string {
	return "device"
}
