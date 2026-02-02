package worker

import (
	"log/slog"
	"time"

	"github.com/alpyxn/aeterna/backend/internal/database"
	"github.com/alpyxn/aeterna/backend/internal/models"
	"github.com/alpyxn/aeterna/backend/internal/services"
)

var settingsService = services.SettingsService{}
var emailService = services.EmailService{}

func Start() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		checkHeartbeats()
	}
}

func checkHeartbeats() {
	var messages []models.Message
	
	// Find active messages where last_seen + trigger_duration < now
	err := database.DB.Where("status = ? AND last_seen < NOW() - (trigger_duration * INTERVAL '1 minute')", models.StatusActive).Find(&messages).Error
	if err != nil {
		slog.Error("Error checking heartbeats", "error", err)
		return
	}

	for _, msg := range messages {
		triggerSwitch(msg)
	}
}

func triggerSwitch(msg models.Message) {
	slog.Warn("Switch triggered", "recipient", msg.RecipientEmail, "id", msg.ID)
	
	// Get SMTP settings
	settings, err := settingsService.Get()
	if err != nil {
		slog.Error("Failed to load SMTP settings", "error", err)
	} else if settings.SMTPHost != "" {
		// Send real email with content directly
		err := emailService.SendTriggeredMessage(settings, msg)
		if err != nil {
			slog.Error("Failed to send email", "error", err, "recipient", msg.RecipientEmail)
		} else {
			slog.Info("Email sent successfully", "recipient", msg.RecipientEmail)
		}
	} else {
		slog.Info("Mock email", "recipient", msg.RecipientEmail, "content", msg.Content)
	}

	// Update Status
	msg.Status = models.StatusTriggered
	database.DB.Save(&msg)
}
