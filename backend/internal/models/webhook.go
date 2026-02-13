package models

import "time"

type Webhook struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	URL       string    `gorm:"not null" json:"url"`
	Secret    string    `gorm:"not null" json:"secret"`
	Enabled   bool      `gorm:"default:1" json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
