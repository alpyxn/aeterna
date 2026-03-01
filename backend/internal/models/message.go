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
	ID              string            `gorm:"type:text;primaryKey" json:"id"`
	Content         string            `gorm:"column:encrypted_content;not null" json:"content"`
	KeyFragment     string            `gorm:"column:key_fragment;not null" json:"-"`
	ManagementToken string            `gorm:"column:management_token;not null" json:"-"`
	RecipientEmail  string            `gorm:"not null" json:"recipient_email"`
	TriggerDuration int               `gorm:"not null" json:"trigger_duration"`
	LastSeen        time.Time         `gorm:"not null;default:CURRENT_TIMESTAMP" json:"last_seen"`
	Status          MessageStatus     `gorm:"default:'active'" json:"status"`
	Reminders       []MessageReminder `gorm:"foreignKey:MessageID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"reminders"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	DeletedAt       gorm.DeletedAt    `gorm:"index" json:"-"`
	AttachmentCount int64             `gorm:"-" json:"attachment_count"`
}

// BeforeCreate hook to generate UUID before creating
func (m *Message) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	if m.ManagementToken == "" {
		m.ManagementToken = uuid.NewString()
	}
	return nil
}
