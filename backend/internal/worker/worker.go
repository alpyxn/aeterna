package worker

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/alpyxn/aeterna/backend/internal/database"
	"github.com/alpyxn/aeterna/backend/internal/models"
	"github.com/alpyxn/aeterna/backend/internal/services"
)

var settingsService = services.SettingsService{}
var emailService = services.EmailService{}
var webhookService = services.WebhookService{}
var webhookStore = services.WebhookStore{}
var workerFileService = services.FileService{}

func Start() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		checkReminders()
		checkHeartbeats()
	}
}

func checkReminders() {
	settings, err := settingsService.Get()
	if err != nil || settings.OwnerEmail == "" || settings.SMTPHost == "" {
		return // No owner email or SMTP configured
	}

	var messages []models.Message

	// Find active messages where 50% of time has passed and reminder not sent
	// last_seen + (trigger_duration * 0.5 minutes) < now AND reminder_sent = false
	err = database.DB.Where(
		"status = ? AND reminder_sent = ? AND datetime(last_seen, '+' || CAST(CAST(trigger_duration * 0.5 AS INTEGER) AS TEXT) || ' minutes') < datetime('now')",
		models.StatusActive, false,
	).Find(&messages).Error

	if err != nil {
		slog.Error("Error checking reminders", "error", err)
		return
	}

	for _, msg := range messages {
		sendReminderEmail(settings, msg)
	}
}

func sendReminderEmail(settings models.Settings, msg models.Message) {
	// Calculate remaining time
	lastSeen := msg.LastSeen
	triggerTime := lastSeen.Add(time.Duration(msg.TriggerDuration) * time.Minute)
	remaining := time.Until(triggerTime)

	var remainingStr string
	if remaining.Hours() > 24 {
		days := int(remaining.Hours() / 24)
		remainingStr = fmt.Sprintf("%d day(s)", days)
	} else if remaining.Hours() > 1 {
		remainingStr = fmt.Sprintf("%.0f hour(s)", remaining.Hours())
	} else {
		remainingStr = fmt.Sprintf("%.0f minute(s)", remaining.Minutes())
	}

	// Build quick heartbeat link
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:5173"
	}
	quickLink := fmt.Sprintf("%s/api/quick-heartbeat/%s", baseURL, settings.HeartbeatToken)

	subject := "Check-in required"
	body := fmt.Sprintf(`You have a scheduled message that will be sent in %s unless you confirm.

Recipient: %s

To confirm you are available, click the link below:
%s

---
Sent by Aeterna`, remainingStr, msg.RecipientEmail, quickLink)

	err := emailService.SendPlain(settings, settings.OwnerEmail, subject, body)
	if err != nil {
		slog.Error("Failed to send reminder email", "error", err, "owner", settings.OwnerEmail)
		return
	}

	// Mark reminder as sent
	database.DB.Model(&msg).Update("reminder_sent", true)
	slog.Info("Reminder email sent", "owner", settings.OwnerEmail, "message_id", msg.ID)
}

func checkHeartbeats() {
	var messages []models.Message

	// Find active messages where last_seen + trigger_duration < now
	err := database.DB.Where(
		"status = ? AND datetime(last_seen, '+' || CAST(trigger_duration AS TEXT) || ' minutes') < datetime('now')",
		models.StatusActive,
	).Find(&messages).Error
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

	// Load file attachments
	var emailAttachments []services.EmailAttachment
	attachments, err := workerFileService.ListByMessageID(msg.ID)
	if err != nil {
		slog.Error("Failed to load attachments", "error", err, "message_id", msg.ID)
	} else {
		for _, att := range attachments {
			filename, mimeType, data, err := workerFileService.GetDecrypted(att.ID)
			if err != nil {
				slog.Error("Failed to decrypt attachment", "error", err, "attachment_id", att.ID)
				continue
			}
			emailAttachments = append(emailAttachments, services.EmailAttachment{
				Filename: filename,
				MimeType: mimeType,
				Data:     data,
			})
		}
	}

	// Get SMTP settings
	settings, err := settingsService.Get()
	if err != nil {
		slog.Error("Failed to load SMTP settings", "error", err)
	} else if settings.SMTPHost != "" {
		// Send real email with content and attachments
		err := emailService.SendTriggeredMessage(settings, msg, emailAttachments)
		if err != nil {
			slog.Error("Failed to send email", "error", err, "recipient", msg.RecipientEmail)
		} else {
			slog.Info("Email sent successfully", "recipient", msg.RecipientEmail, "attachments", len(emailAttachments))
		}
	} else {
		slog.Info("Mock email", "recipient", msg.RecipientEmail, "content", msg.Content, "attachments", len(emailAttachments))
	}

	webhooks, err := webhookStore.ListEnabled()
	if err != nil {
		slog.Error("Failed to load webhooks", "error", err)
	} else if len(webhooks) > 0 {
		slog.Info("Webhook delivery attempt", "count", len(webhooks), "recipient", msg.RecipientEmail)
		if err := webhookService.SendTriggerWebhooks(webhooks, msg); err != nil {
			slog.Error("Failed to deliver webhook", "error", err, "recipient", msg.RecipientEmail)
		} else {
			slog.Info("Webhook delivered", "count", len(webhooks), "recipient", msg.RecipientEmail)
		}
	}

	// Update Status
	msg.Status = models.StatusTriggered
	database.DB.Save(&msg)

	// Clean up attachment files from disk after successful trigger
	if len(attachments) > 0 {
		if err := workerFileService.DeleteByMessageID(msg.ID); err != nil {
			slog.Error("Failed to clean up attachments", "error", err, "message_id", msg.ID)
		} else {
			slog.Info("Attachments cleaned up", "message_id", msg.ID, "count", len(attachments))
		}
	}

	// Notify owner that the message was delivered
	if settings.OwnerEmail != "" && settings.SMTPHost != "" {
		sendOwnerNotification(settings, msg, webhooks)
	}
}

func sendOwnerNotification(settings models.Settings, msg models.Message, webhooks []models.Webhook) {
	webhookInfo := ""
	if len(webhooks) > 0 {
		webhookInfo = "\n\nTriggered Webhooks:\n"
		for _, w := range webhooks {
			webhookInfo += fmt.Sprintf("- %s\n", w.URL)
		}
	}

	subject := "Message delivered"
	body := fmt.Sprintf(`Your scheduled message has been delivered as planned.

Recipient: %s%s

---

Sent by Aeterna`, msg.RecipientEmail, webhookInfo)

	err := emailService.SendPlain(settings, settings.OwnerEmail, subject, body)
	if err != nil {
		slog.Error("Failed to send owner notification", "error", err, "owner", settings.OwnerEmail)
	} else {
		slog.Info("Owner notified of delivery", "owner", settings.OwnerEmail, "recipient", msg.RecipientEmail)
	}
}
