package models

import (
	"time"

	"gorm.io/gorm"
)

// ProductAlias links external codes (EAN, Tracking) to internal IDs (Items, Boxes)
// Standardized: Go (PascalCase) -> DB (snake_case) -> JSON (camelCase)
type ProductAlias struct {
	ID              uint   `gorm:"primaryKey" json:"id"`
	ExternalCode    string `gorm:"not null;index:idx_aliases_external" json:"externalCode"`
	InternalID      string `gorm:"not null;index:idx_aliases_internal" json:"internalId"`
	Type            string `gorm:"not null" json:"type"`
	IsVerified      bool   `gorm:"default:false" json:"isVerified"`
	ConfidenceScore int    `gorm:"default:0" json:"confidenceScore"`
	CreatedContext  string `json:"createdContext,omitempty"`

	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name
func (ProductAlias) TableName() string {
	return "product_aliases"
}
