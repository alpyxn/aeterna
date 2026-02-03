package services

import (
	"errors"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/alpyxn/aeterna/backend/internal/database"
	"github.com/alpyxn/aeterna/backend/internal/models"
	"gorm.io/gorm"
)

type WebhookStore struct{}

func (s WebhookStore) List() ([]models.Webhook, error) {
	var items []models.Webhook
	if err := database.DB.Order("created_at ASC").Find(&items).Error; err != nil {
		return nil, Internal("Failed to fetch webhooks", err)
	}
	for i := range items {
		items[i].Secret = ""
	}
	return items, nil
}

func (s WebhookStore) ListEnabled() ([]models.Webhook, error) {
	var items []models.Webhook
	if err := database.DB.Where("enabled = ?", true).Find(&items).Error; err != nil {
		return nil, Internal("Failed to fetch webhooks", err)
	}
	return items, nil
}

func (s WebhookStore) Create(item models.Webhook) (models.Webhook, error) {
	item.URL = strings.TrimSpace(item.URL)
	if item.URL == "" {
		return models.Webhook{}, BadRequest("Webhook URL is required", nil)
	}
	validatedURL, err := validateWebhookURL(item.URL)
	if err != nil {
		return models.Webhook{}, err
	}
	item.URL = validatedURL
	item.Secret = strings.TrimSpace(item.Secret)
	if item.Secret != "" {
		encrypted, err := cryptoService.EncryptIfNeeded(item.Secret)
		if err != nil {
			return models.Webhook{}, err
		}
		item.Secret = encrypted
	}
	if err := database.DB.Create(&item).Error; err != nil {
		return models.Webhook{}, Internal("Failed to create webhook", err)
	}
	item.Secret = ""
	return item, nil
}

func (s WebhookStore) Update(id string, input models.Webhook) (models.Webhook, error) {
	parsedID, err := strconv.Atoi(id)
	if err != nil {
		return models.Webhook{}, BadRequest("Invalid webhook id", err)
	}
	var existing models.Webhook
	if err := database.DB.First(&existing, parsedID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Webhook{}, NotFound("Webhook not found", err)
		}
		return models.Webhook{}, Internal("Failed to fetch webhook", err)
	}
	input.URL = strings.TrimSpace(input.URL)
	if input.URL == "" {
		return models.Webhook{}, BadRequest("Webhook URL is required", nil)
	}
	validatedURL, err := validateWebhookURL(input.URL)
	if err != nil {
		return models.Webhook{}, err
	}
	secret := strings.TrimSpace(input.Secret)
	if secret != "" {
		encrypted, err := cryptoService.EncryptIfNeeded(secret)
		if err != nil {
			return models.Webhook{}, err
		}
		secret = encrypted
	} else {
		secret = existing.Secret
	}

	existing.URL = validatedURL
	existing.Secret = secret
	existing.Enabled = input.Enabled

	if err := database.DB.Save(&existing).Error; err != nil {
		return models.Webhook{}, Internal("Failed to update webhook", err)
	}
	existing.Secret = ""
	return existing, nil
}

func (s WebhookStore) Delete(id string) error {
	parsedID, err := strconv.Atoi(id)
	if err != nil {
		return BadRequest("Invalid webhook id", err)
	}
	if err := database.DB.Delete(&models.Webhook{}, parsedID).Error; err != nil {
		return Internal("Failed to delete webhook", err)
	}
	return nil
}

func validateWebhookURL(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", BadRequest("Webhook URL is required", nil)
	}

	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", BadRequest("Invalid webhook URL", err)
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "https" {
		return "", BadRequest("Webhook URL must use https", nil)
	}

	if parsed.User != nil {
		return "", BadRequest("Webhook URL must not include credentials", nil)
	}

	if parsed.Fragment != "" {
		return "", BadRequest("Webhook URL must not include fragments", nil)
	}

	hostname := strings.ToLower(parsed.Hostname())
	if hostname == "" {
		return "", BadRequest("Invalid webhook URL host", nil)
	}
	if err := enforceWebhookAllowlist(hostname); err != nil {
		return "", err
	}
	if hostname == "localhost" || strings.HasSuffix(hostname, ".localhost") || strings.HasSuffix(hostname, ".local") {
		return "", BadRequest("Webhook URL host is not allowed", nil)
	}

	if ip := net.ParseIP(hostname); ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
			return "", BadRequest("Webhook URL host is not allowed", nil)
		}
	}

	return parsed.String(), nil
}

func enforceWebhookAllowlist(hostname string) error {
	rawAllowlist := strings.TrimSpace(os.Getenv("WEBHOOK_ALLOWLIST_HOSTS"))
	if rawAllowlist == "" {
		return nil
	}

	host := strings.ToLower(strings.TrimSpace(hostname))
	if host == "" {
		return BadRequest("Invalid webhook URL host", nil)
	}

	for _, entry := range strings.Split(rawAllowlist, ",") {
		candidate := strings.ToLower(strings.TrimSpace(entry))
		if candidate == "" {
			continue
		}
		if host == candidate {
			return nil
		}
		if strings.HasPrefix(candidate, ".") && strings.HasSuffix(host, candidate) {
			return nil
		}
	}

	return BadRequest("Webhook URL host is not allowlisted", nil)
}
