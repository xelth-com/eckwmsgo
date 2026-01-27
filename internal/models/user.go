package models

import (
	"time"

	"gorm.io/gorm"
)

// UserAuth represents a user in the system
// Standardized: Go (PascalCase) -> DB (snake_case) -> JSON (camelCase)
type UserAuth struct {
	ID                  string     `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	Username            string     `gorm:"unique;not null" json:"username"`
	Password            string     `gorm:"not null" json:"-"`
	Email               string     `gorm:"unique;not null" json:"email"`
	Name                string     `json:"name,omitempty"`
	Role                string     `gorm:"default:'user'" json:"role"`
	UserType            string     `gorm:"default:'individual'" json:"userType"`
	Company             string     `json:"company,omitempty"`
	GoogleID            *string    `json:"googleId,omitempty"`
	IsActive            bool       `gorm:"default:true" json:"isActive"`
	LastLogin           *time.Time `json:"lastLogin,omitempty"`
	FailedLoginAttempts int        `gorm:"default:0" json:"-"`
	PreferredLanguage   string     `gorm:"default:'en'" json:"preferredLanguage"`

	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name for UserAuth model
func (UserAuth) TableName() string {
	return "user_auths"
}
