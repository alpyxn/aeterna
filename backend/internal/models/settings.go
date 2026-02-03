package models

type Settings struct {
	ID           uint   `gorm:"primaryKey"`
	SMTPHost     string `gorm:"column:smtp_host" json:"smtp_host"`
	SMTPPort     string `gorm:"column:smtp_port" json:"smtp_port"`
	SMTPUser     string `gorm:"column:smtp_user" json:"smtp_user"`
	SMTPPass     string `gorm:"column:smtp_pass" json:"smtp_pass"`
	SMTPFrom     string `gorm:"column:smtp_from" json:"smtp_from"`
	SMTPFromName string `gorm:"column:smtp_from_name" json:"smtp_from_name"`
	MasterPasswordHash string `gorm:"column:master_password_hash" json:"-"`
	EncryptionKey string `gorm:"column:encryption_key" json:"-"`
	WebhookURL    string `gorm:"column:webhook_url" json:"webhook_url"`
	WebhookSecret string `gorm:"column:webhook_secret" json:"webhook_secret"`
	WebhookEnabled bool  `gorm:"column:webhook_enabled;default:false" json:"webhook_enabled"`
}
