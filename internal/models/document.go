package models

import (
	"time"

	"gorm.io/gorm"
)

// Document represents a structured report from a mobile device (e.g., Workflow Result)
type Document struct {
	ID        string         `gorm:"column:document_id;primaryKey;type:uuid;default:gen_random_uuid()" json:"documentId"`
	Type      string         `gorm:"column:type;not null;index" json:"type"`       // e.g., "ManualRestock", "RMA_Result"
	Status    string         `gorm:"column:status;default:'pending';index" json:"status"` // pending, processed, error
	Payload   JSONB          `gorm:"column:payload;type:jsonb" json:"payload"`    // Flexible JSON content
	DeviceID  string         `gorm:"column:device_id;index" json:"deviceId"`      // Which device sent it
	UserID    string         `gorm:"column:user_id;index" json:"userId"`          // Which user (optional)

	CreatedAt time.Time      `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"column:updated_at" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name
func (Document) TableName() string {
	return "documents"
}
