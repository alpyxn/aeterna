package models

// Settings is the database model - sensitive fields hidden in JSON responses
type Settings struct {
	ID                 uint   `gorm:"primaryKey"`
	SMTPHost           string `gorm:"column:smtp_host" json:"smtp_host"`
	SMTPPort           string `gorm:"column:smtp_port" json:"smtp_port"`
	SMTPUser           string `gorm:"column:smtp_user" json:"smtp_user"`
	SMTPPass           string `gorm:"column:smtp_pass" json:"-"` // Hidden from API responses
	SMTPFrom           string `gorm:"column:smtp_from" json:"smtp_from"`
	SMTPFromName       string `gorm:"column:smtp_from_name" json:"smtp_from_name"`
	MasterPasswordHash string `gorm:"column:master_password_hash" json:"-"`
	RecoveryKeyHash    string `gorm:"column:recovery_key_hash" json:"-"`
	WebhookURL         string `gorm:"column:webhook_url" json:"webhook_url"`
	WebhookSecret      string `gorm:"column:webhook_secret" json:"-"` // Hidden from API responses
	WebhookEnabled     bool   `gorm:"column:webhook_enabled;default:0" json:"webhook_enabled"`
	OwnerEmail         string `gorm:"column:owner_email" json:"owner_email"`
	HeartbeatToken     string `gorm:"column:heartbeat_token" json:"-"`
}

// SettingsRequest is used for receiving settings from API (includes sensitive fields)
type SettingsRequest struct {
	SMTPHost       string `json:"smtp_host"`
	SMTPPort       string `json:"smtp_port"`
	SMTPUser       string `json:"smtp_user"`
	SMTPPass       string `json:"smtp_pass"` // Accepted from API requests
	SMTPFrom       string `json:"smtp_from"`
	SMTPFromName   string `json:"smtp_from_name"`
	WebhookURL     string `json:"webhook_url"`
	WebhookSecret  string `json:"webhook_secret"` // Accepted from API requests
	WebhookEnabled bool   `json:"webhook_enabled"`
	OwnerEmail     string `json:"owner_email"`
}

// ToSettings converts SettingsRequest to Settings model
func (r SettingsRequest) ToSettings() Settings {
	return Settings{
		SMTPHost:       r.SMTPHost,
		SMTPPort:       r.SMTPPort,
		SMTPUser:       r.SMTPUser,
		SMTPPass:       r.SMTPPass,
		SMTPFrom:       r.SMTPFrom,
		SMTPFromName:   r.SMTPFromName,
		WebhookURL:     r.WebhookURL,
		WebhookSecret:  r.WebhookSecret,
		WebhookEnabled: r.WebhookEnabled,
		OwnerEmail:     r.OwnerEmail,
	}
}
