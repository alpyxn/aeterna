package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FarewellLetterStatus string

const (
	FarewellStatusPending FarewellLetterStatus = "pending"
	FarewellStatusSent    FarewellLetterStatus = "sent"
)

type FarewellLetter struct {
	ID                 string               `gorm:"type:text;primaryKey" json:"id"`
	UserID             string               `gorm:"type:text;index" json:"-"`
	MessageID          string               `gorm:"type:text;not null;index" json:"message_id"`
	RecipientEmail     string               `gorm:"not null" json:"recipient_email"`
	Subject            string               `gorm:"not null" json:"subject"`
	Content            string               `gorm:"column:encrypted_content;not null" json:"content"`
	RawContent         string               `gorm:"column:encrypted_content_raw;not null;default:''" json:"-"`
	RenderedHTML       string               `gorm:"column:encrypted_rendered_html;not null;default:''" json:"-"`
	WordCount          int                  `gorm:"not null;default:0" json:"word_count"`
	DerivativesPending bool                 `gorm:"column:derivatives_pending;not null;default:1" json:"-"`
	DelayMinutes       int                  `gorm:"not null" json:"delay_minutes"`
	Status             FarewellLetterStatus `gorm:"default:'pending'" json:"status"`
	SentAt             *time.Time           `json:"sent_at,omitempty"`
	AttachmentCount    int64                `gorm:"-" json:"attachment_count"`
	CreatedAt          time.Time            `json:"created_at"`
	UpdatedAt          time.Time            `json:"updated_at"`
	DeletedAt          gorm.DeletedAt       `gorm:"index" json:"-"`
}

func (f *FarewellLetter) BeforeCreate(tx *gorm.DB) error {
	if f.ID == "" {
		f.ID = uuid.NewString()
	}
	return nil
}

// BeforeDelete cascades the delete to associated FarewellAttachments,
// mirroring the soft/hard mode of the parent operation.
// When called from a batch context (f.ID == ""), the parent Message.BeforeDelete handles it.
//
// Uses a fresh session so the parent statement's Where/Select clauses don't leak
// into the cascade query.
func (f *FarewellLetter) BeforeDelete(tx *gorm.DB) error {
	if f.ID == "" {
		return nil
	}
	sess := tx.Session(&gorm.Session{NewDB: true})
	if tx.Statement.Unscoped {
		sess = sess.Unscoped()
	}
	return sess.Where("letter_id = ?", f.ID).Delete(&FarewellAttachment{}).Error
}
