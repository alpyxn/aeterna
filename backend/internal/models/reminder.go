package models

// MessageReminder defines a scheduled reminder for a specific Message
type MessageReminder struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	MessageID string `gorm:"type:text;index;not null;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"message_id"`

	// How many minutes before triggering the switch to send this reminder
	MinutesBefore int  `gorm:"not null" json:"minutes_before"`
	Sent          bool `gorm:"default:0" json:"sent"`
}
