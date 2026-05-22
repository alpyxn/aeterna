package ports

import (
	"time"

	"github.com/alpyxn/aeterna/backend/internal/models"
)

// AuthServicePort covers authentication and session management.
type AuthServicePort interface {
	IsConfigured() (bool, error)
	RegisterFirstUser(email, password, ownerEmail string) (recoveryKey string, user models.User, err error)
	RegisterAdditionalUser(email, password, ownerEmail string) (recoveryKey string, user models.User, err error)
	Login(email, password string) (models.User, error)
	IssueSessionToken(userID string) (string, time.Time, error)
	VerifySessionToken(token string) (userID string, err error)
	ResetPasswordWithRecovery(email, recoveryKey, newPassword string) (newRecoveryKey string, err error)
	AdditionalRegistrationOpen() (bool, error)
}

// MessageServicePort covers switch lifecycle and heartbeat operations.
type MessageServicePort interface {
	Create(userID, content string, recipientEmails []string, triggerDuration int, reminders []int) (models.Message, error)
	GetPublicByID(id string) (models.Message, error)
	GetByID(userID, id string) (models.Message, error)
	List(userID string) ([]models.Message, error)
	Heartbeat(userID, id string) (models.Message, error)
	BulkHeartbeat(userID string) error
	Delete(userID, id string) error
	Update(userID, id, content string, recipientEmails []string, triggerDuration int, reminders []int) (models.Message, error)
}

// FileServicePort covers attachment storage for switches and farewell letters.
type FileServicePort interface {
	Upload(userID, messageID, filename, mimeType string, data []byte) (models.Attachment, error)
	Delete(userID, attachmentID string) error
	DeleteByMessageID(userID, messageID string) error
	GetDecrypted(userID, attachmentID string) (filename, mimeType string, data []byte, err error)
	ListByMessageID(userID, messageID string) ([]models.Attachment, error)
	CountByMessageID(userID, messageID string) (int64, error)
	UploadFarewellAttachment(userID, letterID, filename, mimeType string, data []byte) (models.FarewellAttachment, error)
	ListFarewellAttachmentsByLetterID(userID, letterID string) ([]models.FarewellAttachment, error)
	CountFarewellAttachmentsByLetterID(userID, letterID string) (int64, error)
	DeleteFarewellAttachment(userID, attachmentID string) error
	DeleteFarewellAttachmentsByLetterID(userID, letterID string) error
	GetFarewellAttachmentDecrypted(userID, attachmentID string) (filename, mimeType string, data []byte, err error)
}

// FarewellServicePort covers farewell letter CRUD scoped to a switch.
type FarewellServicePort interface {
	Create(userID, messageID, recipientEmail, subject, content string, delayMinutes int) (models.FarewellLetter, error)
	List(userID, messageID string) ([]models.FarewellLetter, error)
	Update(userID, messageID, id, recipientEmail, subject, content string, delayMinutes int) (models.FarewellLetter, error)
	Delete(userID, messageID, id string) error
	CancelPending(userID, messageID, id string) error
	CancelPendingByMessageID(userID, messageID string) (int64, error)
}

// SettingsServicePort covers per-user SMTP and heartbeat token configuration.
type SettingsServicePort interface {
	Get(userID string) (models.Settings, error)
	GetByHeartbeatToken(token string) (models.Settings, error)
	Save(userID string, req models.Settings) error
	TestSMTP(req models.Settings) error
}

// ApplicationSettingsServicePort covers the global (singleton) application settings.
type ApplicationSettingsServicePort interface {
	Get() (models.ApplicationSettings, error)
	SetAllowRegistration(actorUserID string, allow bool) error
	CanManageRegistration(userID string) bool
}

// WebhookStorePort covers webhook CRUD for a tenant.
type WebhookStorePort interface {
	List(userID string) ([]models.Webhook, error)
	ListEnabledForUser(userID string) ([]models.Webhook, error)
	Create(userID string, item models.Webhook) (models.Webhook, error)
	Update(userID, id string, input models.Webhook) (models.Webhook, error)
	Delete(userID, id string) error
}

// UserAdminServicePort covers administrative user account management.
type UserAdminServicePort interface {
	List(actorUserID string) ([]models.UserListItem, error)
	Delete(actorUserID, targetUserID string) error
}
