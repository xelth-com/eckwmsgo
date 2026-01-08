package models

import (
	"time"

	"gorm.io/gorm"
)

// UserAuth represents a user in the system
type UserAuth struct {
	ID                  string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	Username            string         `gorm:"unique;not null" json:"username"`
	Password            string         `gorm:"not null" json:"-"` // Never send password in JSON
	Email               string         `gorm:"unique;not null" json:"email"`
	Name                string         `json:"name,omitempty"`
	Role                string         `gorm:"default:'user'" json:"role"`
	UserType            string         `gorm:"default:'individual'" json:"user_type"`
	Company             string         `json:"company,omitempty"`
	GoogleID            *string        `json:"google_id,omitempty"`
	IsActive            bool           `gorm:"default:true" json:"is_active"`
	LastLogin           *time.Time     `json:"last_login,omitempty"`
	FailedLoginAttempts int            `gorm:"default:0" json:"-"`
	PreferredLanguage   string         `gorm:"default:'en'" json:"preferred_language"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
	DeletedAt           gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name for UserAuth model
func (UserAuth) TableName() string {
	return "user_auths"
}
