package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RefreshSession stores hashed refresh tokens for v2 mobile-style auth.
type RefreshSession struct {
	ID                  string     `gorm:"type:text;primaryKey"`
	UserID              string     `gorm:"type:text;index;not null"`
	User                User       `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE;" json:"-"`
	TokenHash           string     `gorm:"type:text;uniqueIndex;not null"`
	ExpiresAt           time.Time  `gorm:"index;not null"`
	RevokedAt           *time.Time `gorm:"index"`
	ReplacedByTokenHash string     `gorm:"type:text"`
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func (r *RefreshSession) BeforeCreate(tx *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}
