package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MessageStatus string

const (
	StatusActive    MessageStatus = "active"
	StatusTriggered MessageStatus = "triggered"
)

type Message struct {
	ID              uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Content         string         `gorm:"column:encrypted_content;not null" json:"content"`
	KeyFragment     string         `gorm:"column:key_fragment;not null" json:"-"`
	ManagementToken string         `gorm:"column:management_token;not null" json:"-"`
	RecipientEmail  string         `gorm:"not null" json:"recipient_email"`
	TriggerDuration int            `gorm:"not null" json:"trigger_duration"`
	LastSeen        time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP" json:"last_seen"`
	Status          MessageStatus  `gorm:"default:'active'" json:"status"`
	ReminderSent    bool           `gorm:"default:false" json:"-"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

