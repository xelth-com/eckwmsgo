package models

import (
	"time"

	"gorm.io/gorm"
)

// RegisteredDevice represents a paired mobile device
type RegisteredDevice struct {
	DeviceID   string         `gorm:"primaryKey" json:"deviceId"`
	InstanceID *string        `json:"instance_id,omitempty"`
	IsActive   bool           `gorm:"default:true" json:"is_active"`
	Status     string         `gorm:"default:'pending'" json:"status"` // active, pending, blocked
	PublicKey  string         `gorm:"not null" json:"publicKey"`       // Base64
	DeviceName string         `json:"deviceName"`
	RoleID     *uint          `json:"role_id,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name for RegisteredDevice
func (RegisteredDevice) TableName() string {
	return "registered_devices"
}
