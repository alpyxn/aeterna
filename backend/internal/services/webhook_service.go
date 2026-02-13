package services

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/alpyxn/aeterna/backend/internal/models"
)

type WebhookService struct{}

type triggerPayload struct {
	Event          string    `json:"event"`
	MessageID      string    `json:"message_id"`
	RecipientEmail string    `json:"recipient_email"`
	Content        string    `json:"content"`
	TriggerDuration int      `json:"trigger_duration"`
	LastSeen       time.Time `json:"last_seen"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
}

func (s WebhookService) SendTriggerWebhooks(webhooks []models.Webhook, msg models.Message) error {
	if len(webhooks) == 0 {
		return nil
	}

	content := msg.Content
	if msg.Content != "" {
		decrypted, err := cryptoService.Decrypt(msg.Content)
		if err != nil {
			return err
		}
		content = decrypted
	}

	payload := triggerPayload{
		Event:           "switch.triggered",
		MessageID:       msg.ID,
		RecipientEmail:  msg.RecipientEmail,
		Content:         content,
		TriggerDuration: msg.TriggerDuration,
		LastSeen:        msg.LastSeen,
		Status:          string(msg.Status),
		CreatedAt:       msg.CreatedAt,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return Internal("Failed to encode webhook payload", err)
	}

	client := &http.Client{Timeout: 6 * time.Second}
	var lastErr error
	for _, hook := range webhooks {
		if hook.URL == "" {
			lastErr = BadRequest("Webhook URL is required", nil)
			continue
		}
		secret := ""
		if hook.Secret != "" {
			decrypted, err := cryptoService.DecryptIfNeeded(hook.Secret)
			if err != nil {
				lastErr = err
				continue
			}
			secret = decrypted
		}

		req, err := http.NewRequest(http.MethodPost, hook.URL, bytes.NewBuffer(body))
		if err != nil {
			lastErr = Internal("Failed to create webhook request", err)
			continue
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Aeterna-Event", payload.Event)

		if secret != "" {
			mac := hmac.New(sha256.New, []byte(secret))
			mac.Write(body)
			signature := hex.EncodeToString(mac.Sum(nil))
			req.Header.Set("X-Aeterna-Signature", signature)
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = Internal("Webhook request failed", err)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			lastErr = Internal("Webhook returned non-2xx status", errors.New(resp.Status))
			continue
		}
	}

	if lastErr != nil {
		return lastErr
	}

	return nil
}
