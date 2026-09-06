package services

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/alpyxn/aeterna/backend/internal/models"
)

type WebhookService struct{}

type triggerPayload struct {
	Event           string    `json:"event"`
	MessageID       string    `json:"message_id"`
	RecipientEmail  string    `json:"recipient_email"`
	RecipientEmails []string  `json:"recipient_emails"`
	Content         string    `json:"content"`
	TriggerDuration int       `json:"trigger_duration"`
	LastSeen        time.Time `json:"last_seen"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
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
		RecipientEmails: ParseRecipientEmails(msg.RecipientEmail),
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

	client := newWebhookHTTPClient()
	var lastErr error
	for _, hook := range webhooks {
		if hook.URL == "" {
			lastErr = BadRequest("Webhook URL is required", nil)
			continue
		}
		// Re-validate the stored URL at delivery time (scheme, credentials,
		// literal private IPs). Host resolution is re-checked at dial time.
		if _, err := validateWebhookURL(hook.URL, ""); err != nil {
			lastErr = err
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

// newWebhookHTTPClient returns an HTTP client hardened against SSRF for
// outbound webhook delivery: redirects are not followed, and every TCP
// connection is dialed only after the resolved IP has been checked against the
// private/loopback/link-local deny-list at connect time. Dialing the resolved
// IP directly also closes the DNS-rebinding TOCTOU between resolution and dial.
func newWebhookHTTPClient() *http.Client {
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil {
				return nil, err
			}
			if len(ips) == 0 {
				return nil, errors.New("webhook host resolved to no addresses")
			}
			for _, ip := range ips {
				if err := validateWebhookIP(ip.IP.String()); err != nil {
					return nil, errors.New("webhook host resolves to a disallowed IP address")
				}
			}
			var lastErr error
			for _, ip := range ips {
				conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip.IP.String(), port))
				if err == nil {
					return conn, nil
				}
				lastErr = err
			}
			return nil, lastErr
		},
	}
	return &http.Client{
		Timeout:   6 * time.Second,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}
