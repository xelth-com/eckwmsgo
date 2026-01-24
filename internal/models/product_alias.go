package models

import (
	"time"

	"gorm.io/gorm"
)

// ProductAlias links external codes (EAN, Tracking) to internal IDs (Items, Boxes)
// Ported from Node.js src/shared/models/postgresql/ProductAlias.js
type ProductAlias struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	ExternalCode    string         `gorm:"not null;index:idx_aliases_external" json:"external_code"` // The unknown code scanned
	InternalID      string         `gorm:"not null;index:idx_aliases_internal" json:"internal_id"`   // The known item/box it links to
	Type            string         `gorm:"not null" json:"type"`                                     // 'ean', 'tracking', 'serial', 'manual_link'
	IsVerified      bool           `gorm:"default:false" json:"is_verified"`                         // True if human confirmed
	ConfidenceScore int            `gorm:"default:0" json:"confidence_score"`
	CreatedContext  string         `json:"created_context,omitempty"` // 'receiving', 'moving', etc.
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name
func (ProductAlias) TableName() string {
	return "product_aliases"
}
