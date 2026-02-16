package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Attachment struct {
	ID          string         `gorm:"type:text;primaryKey" json:"id"`
	MessageID   string         `gorm:"type:text;not null;index" json:"message_id"`
	Filename    string         `gorm:"not null" json:"filename"`
	StoragePath string         `gorm:"not null" json:"-"`
	Size        int64          `gorm:"not null" json:"size"`
	MimeType    string         `gorm:"not null" json:"mime_type"`
	CreatedAt   time.Time      `json:"created_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// BeforeCreate hook to generate UUID before creating
func (a *Attachment) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	return nil
}
