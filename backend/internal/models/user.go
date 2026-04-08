package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User is a tenant account (email + password).
type User struct {
	ID           string `gorm:"type:text;primaryKey" json:"id"`
	Email        string `gorm:"not null;uniqueIndex" json:"email"`
	PasswordHash string `gorm:"not null" json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.NewString()
	}
	return nil
}
