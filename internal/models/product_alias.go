package models

import (
	"time"

	"gorm.io/gorm"
)

// ProductAlias links external codes (EAN, Tracking) to internal IDs (Items, Boxes)
type ProductAlias struct {
	ID              uint   `gorm:"primaryKey" json:"id"`
	ExternalCode    string `gorm:"column:externalCode;not null;index:idx_aliases_external" json:"externalCode"`
	InternalID      string `gorm:"column:internalId;not null;index:idx_aliases_internal" json:"internalId"`
	Type            string `gorm:"not null" json:"type"`
	IsVerified      bool   `gorm:"column:isVerified;default:false" json:"isVerified"`
	ConfidenceScore int    `gorm:"column:confidenceScore;default:0" json:"confidenceScore"`
	CreatedContext  string `gorm:"column:createdContext" json:"createdContext,omitempty"`

	CreatedAt time.Time      `gorm:"column:createdAt" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"column:updatedAt" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name
func (ProductAlias) TableName() string {
	return "product_aliases"
}
