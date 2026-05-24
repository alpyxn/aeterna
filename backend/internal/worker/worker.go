package worker

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/alpyxn/aeterna/backend/internal/config"
	"github.com/alpyxn/aeterna/backend/internal/database"
	"github.com/alpyxn/aeterna/backend/internal/models"
	"github.com/alpyxn/aeterna/backend/internal/ports"
	"github.com/alpyxn/aeterna/backend/internal/services"
)

// Worker runs the background goroutine that checks heartbeats, reminders, and farewell letters.
type Worker struct {
	settings           ports.SettingsServicePort
	webhooks           ports.WebhookStorePort
	files              ports.FileServicePort
	farewellDerivation ports.FarewellDerivationPort
	email              services.EmailService
	webhook            services.WebhookService
	cfg                config.Config
}

func New(
	settings ports.SettingsServicePort,
	webhooks ports.WebhookStorePort,
	files ports.FileServicePort,
	farewellDerivation ports.FarewellDerivationPort,
	cfg config.Config,
) *Worker {
	return &Worker{
		settings:           settings,
		webhooks:           webhooks,
		files:              files,
		farewellDerivation: farewellDerivation,
		cfg:                cfg,
	}
}

func (w *Worker) Start() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		w.checkFarewellDerivatives()
		w.checkReminders()
		w.checkHeartbeats()
		w.checkFarewellLetters()
	}
}

func (w *Worker) checkFarewellDerivatives() {
	if w.farewellDerivation == nil {
		return
	}

	processed, err := w.farewellDerivation.ProcessPending(50)
	if err != nil {
		slog.Error("Error checking farewell derivatives", "error", err)
		return
	}
	if processed > 0 {
		slog.Info("Farewell derivatives processed", "count", processed)
	}
}

func (w *Worker) checkReminders() {
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
		settings, err := w.settings.Get(msg.UserID)
		if err != nil || settings.OwnerEmail == "" || settings.SMTPHost == "" {
			continue
		}
		w.sendReminderEmail(settings, msg, req)
	}
}

func (w *Worker) sendReminderEmail(settings models.Settings, msg models.Message, reminder models.MessageReminder) {
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

	quickLink := fmt.Sprintf("%s/api/quick-heartbeat/%s", w.cfg.Worker.BaseURL, settings.HeartbeatToken)

	subject := "Check-in required"
	body := fmt.Sprintf(`You have a scheduled message that will be sent in %s unless you confirm.

Recipient: %s

To confirm you are available, click the link below:
%s

---
Sent by Aeterna`, remainingStr, formatRecipients(msg.RecipientEmail), quickLink)

	err := w.email.SendPlain(settings, []string{settings.OwnerEmail}, subject, body)
	if err != nil {
		slog.Error("Failed to send reminder email", "error", err, "owner", settings.OwnerEmail)
		return
	}

	if err := database.DB.Model(&reminder).Update("sent", true).Error; err != nil {
		slog.Error("Failed to mark reminder as sent", "error", err, "reminder_id", reminder.ID)
	}
	slog.Info("Reminder email sent", "owner", settings.OwnerEmail, "message_id", msg.ID, "minutes_before", reminder.MinutesBefore)
}

func (w *Worker) checkHeartbeats() {
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
		w.triggerSwitch(msg)
	}
}

func (w *Worker) triggerSwitch(msg models.Message) {
	slog.Warn("Switch triggered", "recipient", formatRecipients(msg.RecipientEmail), "id", msg.ID)

	settings, err := w.settings.Get(msg.UserID)
	if err != nil {
		slog.Error("Failed to load SMTP settings", "error", err, "user_id", msg.UserID)
		settings = models.Settings{}
	}

	var emailAttachments []services.EmailAttachment
	attachments, err := w.files.ListByMessageID(msg.UserID, msg.ID)
	if err != nil {
		slog.Error("Failed to load attachments", "error", err, "message_id", msg.ID)
	} else {
		for _, att := range attachments {
			filename, mimeType, data, err := w.files.GetDecrypted(msg.UserID, att.ID)
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
		err := w.email.SendTriggeredMessage(settings, msg, emailAttachments)
		if err != nil {
			slog.Error("Failed to send email", "error", err, "recipient", formatRecipients(msg.RecipientEmail))
		} else {
			slog.Info("Email sent successfully", "recipient", formatRecipients(msg.RecipientEmail), "attachments", len(emailAttachments))
		}
	} else {
		slog.Info("Mock email", "recipient", formatRecipients(msg.RecipientEmail), "attachments", len(emailAttachments))
	}

	webhooks, err := w.webhooks.ListEnabledForUser(msg.UserID)
	if err != nil {
		slog.Error("Failed to load webhooks", "error", err)
	} else if len(webhooks) > 0 {
		slog.Info("Webhook delivery attempt", "count", len(webhooks), "recipient", formatRecipients(msg.RecipientEmail))
		if err := w.webhook.SendTriggerWebhooks(webhooks, msg); err != nil {
			slog.Error("Failed to deliver webhook", "error", err, "recipient", formatRecipients(msg.RecipientEmail))
		} else {
			slog.Info("Webhook delivered", "count", len(webhooks), "recipient", formatRecipients(msg.RecipientEmail))
		}
	}

	now := time.Now().UTC()
	msg.Status = models.StatusTriggered
	msg.TriggeredAt = &now
	if err := database.ForTenant(msg.UserID).Save(&msg).Error; err != nil {
		slog.Error("Failed to persist triggered status", "error", err, "message_id", msg.ID)
	}

	if len(attachments) > 0 {
		if err := w.files.DeleteByMessageID(msg.UserID, msg.ID); err != nil {
			slog.Error("Failed to clean up attachments", "error", err, "message_id", msg.ID)
		} else {
			slog.Info("Attachments cleaned up", "message_id", msg.ID, "count", len(attachments))
		}
	}

	if settings.OwnerEmail != "" && settings.SMTPHost != "" {
		w.sendOwnerNotification(settings, msg, webhooks)
	}
}

func (w *Worker) sendOwnerNotification(settings models.Settings, msg models.Message, webhooks []models.Webhook) {
	webhookInfo := ""
	if len(webhooks) > 0 {
		webhookInfo = "\n\nTriggered Webhooks:\n"
		for _, wh := range webhooks {
			webhookInfo += fmt.Sprintf("- %s\n", wh.URL)
		}
	}

	subject := "Message delivered"
	body := fmt.Sprintf(`Your scheduled message has been delivered as planned.

Recipient: %s%s

---

Sent by Aeterna`, formatRecipients(msg.RecipientEmail), webhookInfo)

	err := w.email.SendPlain(settings, []string{settings.OwnerEmail}, subject, body)
	if err != nil {
		slog.Error("Failed to send owner notification", "error", err, "owner", settings.OwnerEmail)
	} else {
		slog.Info("Owner notified of delivery", "owner", settings.OwnerEmail, "recipient", formatRecipients(msg.RecipientEmail))
	}
}

func (w *Worker) checkFarewellLetters() {
	var letters []models.FarewellLetter

	err := database.DB.Table("farewell_letters").
		Select("farewell_letters.*").
		Joins("JOIN messages ON messages.id = farewell_letters.message_id").
		Where("farewell_letters.status = ?", models.FarewellStatusPending).
		Where("messages.status = ?", models.StatusTriggered).
		Where("messages.triggered_at IS NOT NULL").
		Where("datetime(messages.triggered_at, '+' || CAST(farewell_letters.delay_minutes AS TEXT) || ' minutes') <= datetime('now')").
		Where("farewell_letters.deleted_at IS NULL").
		Find(&letters).Error

	if err != nil {
		slog.Error("Error checking farewell letters", "error", err)
		return
	}

	for _, letter := range letters {
		if letter.UserID == "" {
			continue
		}
		w.sendFarewellLetter(letter)
	}
}

func (w *Worker) sendFarewellLetter(letter models.FarewellLetter) {
	settings, err := w.settings.Get(letter.UserID)
	if err != nil || settings.SMTPHost == "" {
		slog.Error("SMTP not configured for farewell letter", "letter_id", letter.ID, "user_id", letter.UserID)
		return
	}

	decryptedSafeMarkdown, err := services.CryptoService{}.Decrypt(letter.Content)
	if err != nil {
		slog.Error("Failed to decrypt farewell letter content", "letter_id", letter.ID, "error", err)
		return
	}

	var renderedHTML string
	if strings.TrimSpace(letter.RenderedHTML) != "" {
		decryptedHTML, htmlErr := services.CryptoService{}.Decrypt(letter.RenderedHTML)
		if htmlErr != nil {
			slog.Warn("Failed to decrypt pre-rendered farewell HTML, using markdown fallback", "letter_id", letter.ID, "error", htmlErr)
		} else {
			renderedHTML = decryptedHTML
		}
	}

	rawAttachments, err := w.files.ListFarewellAttachmentsByLetterID(letter.UserID, letter.ID)
	if err != nil {
		slog.Error("Failed to load farewell attachments", "letter_id", letter.ID, "error", err)
	}

	var emailAttachments []services.EmailAttachment
	for _, att := range rawAttachments {
		filename, mimeType, data, err := w.files.GetFarewellAttachmentDecrypted(letter.UserID, att.ID)
		if err != nil {
			slog.Error("Failed to decrypt farewell attachment", "attachment_id", att.ID, "error", err)
			continue
		}
		emailAttachments = append(emailAttachments, services.EmailAttachment{
			Filename: filename,
			MimeType: mimeType,
			Data:     data,
		})
	}

	if err := w.email.SendFarewellLetterPreRendered(
		settings,
		letter.RecipientEmail,
		letter.Subject,
		decryptedSafeMarkdown,
		renderedHTML,
		emailAttachments,
	); err != nil {
		slog.Error("Failed to send farewell letter", "letter_id", letter.ID, "recipient", letter.RecipientEmail, "error", err)
		return
	}

	now := time.Now().UTC()
	if err := database.ForTenant(letter.UserID).Model(&letter).Updates(map[string]any{
		"status":  models.FarewellStatusSent,
		"sent_at": now,
	}).Error; err != nil {
		slog.Error("Failed to mark farewell letter as sent", "error", err, "letter_id", letter.ID)
	}

	if len(rawAttachments) > 0 {
		if err := w.files.DeleteFarewellAttachmentsByLetterID(letter.UserID, letter.ID); err != nil {
			slog.Error("Failed to clean up farewell attachments", "letter_id", letter.ID, "error", err)
		}
	}

	slog.Info("Farewell letter sent", "letter_id", letter.ID, "recipient", letter.RecipientEmail)
}

func formatRecipients(value string) string {
	recipients := services.ParseRecipientEmails(value)
	if len(recipients) == 0 {
		return strings.TrimSpace(value)
	}
	return strings.Join(recipients, ", ")
}
