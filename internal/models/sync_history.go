package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// SyncHistory records each synchronization attempt with external providers
type SyncHistory struct {
	ID          int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Provider    string         `gorm:"column:provider;not null;index" json:"provider"` // "opal", "dhl", "odoo"
	Status      string         `gorm:"column:status;not null;index" json:"status"`     // "success", "error", "partial"
	StartedAt   time.Time      `gorm:"column:started_at;not null" json:"startedAt"`
	CompletedAt *time.Time     `gorm:"column:completed_at" json:"completedAt"`
	Duration    int            `gorm:"column:duration;default:0" json:"duration"` // milliseconds
	Created     int            `gorm:"column:created;default:0" json:"created"`   // records created
	Updated     int            `gorm:"column:updated;default:0" json:"updated"`   // records updated
	Skipped     int            `gorm:"column:skipped;default:0" json:"skipped"`   // records skipped
	Errors      int            `gorm:"column:errors;default:0" json:"errors"`     // error count
	ErrorDetail string         `gorm:"column:error_detail;type:text" json:"errorDetail"`
	DebugInfo   JSONB          `gorm:"column:debug_info;type:jsonb" json:"debugInfo"` // Full error context for AI analysis
	CreatedAt   time.Time      `gorm:"column:created_at" json:"-"`
	UpdatedAt   time.Time      `gorm:"column:updated_at" json:"-"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name
func (SyncHistory) TableName() string {
	return "sync_history"
}

// GetEntityID implements SyncableEntity interface
func (s SyncHistory) GetEntityID() string {
	return fmt.Sprintf("%d", s.ID)
}

// GetEntityType implements SyncableEntity interface
func (s SyncHistory) GetEntityType() string {
	return "sync_history"
}
