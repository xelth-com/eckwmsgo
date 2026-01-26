package models

import (
	"time"

	"gorm.io/gorm"
)

// UserAuth represents a user in the system
// Refactored to map Go fields (PascalCase) to Legacy DB columns (camelCase)
type UserAuth struct {
	ID                  string     `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	Username            string     `gorm:"unique;not null" json:"username"`
	Password            string     `gorm:"not null" json:"-"`
	Email               string     `gorm:"unique;not null" json:"email"`
	Name                string     `json:"name,omitempty"`
	Role                string     `gorm:"default:'user'" json:"role"`
	UserType            string     `gorm:"column:userType;default:'individual'" json:"userType"`
	Company             string     `json:"company,omitempty"`
	GoogleID            *string    `gorm:"column:googleId" json:"googleId,omitempty"`
	IsActive            bool       `gorm:"column:isActive;default:true" json:"isActive"`
	LastLogin           *time.Time `gorm:"column:lastLogin" json:"lastLogin,omitempty"`
	FailedLoginAttempts int        `gorm:"column:failedLoginAttempts;default:0" json:"-"`
	PreferredLanguage   string     `gorm:"column:preferredLanguage;default:'en'" json:"preferredLanguage"`

	CreatedAt time.Time      `gorm:"column:createdAt" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"column:updatedAt" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name for UserAuth model
func (UserAuth) TableName() string {
	return "user_auths"
}
