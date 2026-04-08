package worker

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
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
	var reminders []models.MessageReminder

	err := database.DB.Table("message_reminders").
		Select("message_reminders.*").
		Joins("JOIN messages ON messages.id = message_reminders.message_id").
		Where("messages.status = ?", models.StatusActive).
		Where("message_reminders.sent = ?", false).
		Where("datetime('now') >= datetime(messages.last_seen, '+' || CAST((messages.trigger_duration - message_reminders.minutes_before) AS TEXT) || ' minutes')").
		Find(&reminders).Error

	if err != nil {
		slog.Error("Error checking reminders", "error", err)
		return
	}

	for _, req := range reminders {
		var msg models.Message
		if err := database.DB.First(&msg, "id = ?", req.MessageID).Error; err != nil {
			continue
		}
		if msg.UserID == "" {
			continue
		}
		settings, err := settingsService.Get(msg.UserID)
		if err != nil || settings.OwnerEmail == "" || settings.SMTPHost == "" {
			continue
		}
		sendReminderEmail(settings, msg, req)
	}
}

func sendReminderEmail(settings models.Settings, msg models.Message, reminder models.MessageReminder) {
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
Sent by Aeterna`, remainingStr, formatRecipients(msg.RecipientEmail), quickLink)

	err := emailService.SendPlain(settings, []string{settings.OwnerEmail}, subject, body)
	if err != nil {
		slog.Error("Failed to send reminder email", "error", err, "owner", settings.OwnerEmail)
		return
	}

	database.DB.Model(&reminder).Update("sent", true)
	slog.Info("Reminder email sent", "owner", settings.OwnerEmail, "message_id", msg.ID, "minutes_before", reminder.MinutesBefore)
}

func checkHeartbeats() {
	var messages []models.Message

	err := database.DB.Where(
		"status = ? AND datetime(last_seen, '+' || CAST(trigger_duration AS TEXT) || ' minutes') < datetime('now')",
		models.StatusActive,
	).Find(&messages).Error
	if err != nil {
		slog.Error("Error checking heartbeats", "error", err)
		return
	}

	for _, msg := range messages {
		if msg.UserID == "" {
			continue
		}
		triggerSwitch(msg)
	}
}

func triggerSwitch(msg models.Message) {
	slog.Warn("Switch triggered", "recipient", formatRecipients(msg.RecipientEmail), "id", msg.ID)

	settings, err := settingsService.Get(msg.UserID)
	if err != nil {
		slog.Error("Failed to load SMTP settings", "error", err, "user_id", msg.UserID)
		settings = models.Settings{}
	}

	var emailAttachments []services.EmailAttachment
	attachments, err := workerFileService.ListByMessageID(msg.UserID, msg.ID)
	if err != nil {
		slog.Error("Failed to load attachments", "error", err, "message_id", msg.ID)
	} else {
		for _, att := range attachments {
			filename, mimeType, data, err := workerFileService.GetDecrypted(msg.UserID, att.ID)
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

	if settings.SMTPHost != "" {
		err := emailService.SendTriggeredMessage(settings, msg, emailAttachments)
		if err != nil {
			slog.Error("Failed to send email", "error", err, "recipient", formatRecipients(msg.RecipientEmail))
		} else {
			slog.Info("Email sent successfully", "recipient", formatRecipients(msg.RecipientEmail), "attachments", len(emailAttachments))
		}
	} else {
		slog.Info("Mock email", "recipient", formatRecipients(msg.RecipientEmail), "attachments", len(emailAttachments))
	}

	webhooks, err := webhookStore.ListEnabledForUser(msg.UserID)
	if err != nil {
		slog.Error("Failed to load webhooks", "error", err)
	} else if len(webhooks) > 0 {
		slog.Info("Webhook delivery attempt", "count", len(webhooks), "recipient", formatRecipients(msg.RecipientEmail))
		if err := webhookService.SendTriggerWebhooks(webhooks, msg); err != nil {
			slog.Error("Failed to deliver webhook", "error", err, "recipient", formatRecipients(msg.RecipientEmail))
		} else {
			slog.Info("Webhook delivered", "count", len(webhooks), "recipient", formatRecipients(msg.RecipientEmail))
		}
	}

	msg.Status = models.StatusTriggered
	database.DB.Save(&msg)

	if len(attachments) > 0 {
		if err := workerFileService.DeleteByMessageID(msg.UserID, msg.ID); err != nil {
			slog.Error("Failed to clean up attachments", "error", err, "message_id", msg.ID)
		} else {
			slog.Info("Attachments cleaned up", "message_id", msg.ID, "count", len(attachments))
		}
	}

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

Sent by Aeterna`, formatRecipients(msg.RecipientEmail), webhookInfo)

	err := emailService.SendPlain(settings, []string{settings.OwnerEmail}, subject, body)
	if err != nil {
		slog.Error("Failed to send owner notification", "error", err, "owner", settings.OwnerEmail)
	} else {
		slog.Info("Owner notified of delivery", "owner", settings.OwnerEmail, "recipient", formatRecipients(msg.RecipientEmail))
	}
}

func formatRecipients(value string) string {
	recipients := services.ParseRecipientEmails(value)
	if len(recipients) == 0 {
		return strings.TrimSpace(value)
	}
	return strings.Join(recipients, ", ")
}
