package services

import (
	"crypto/tls"
	"errors"
	"net/smtp"
	"strings"

	"github.com/alpyxn/aeterna/backend/internal/database"
	"github.com/alpyxn/aeterna/backend/internal/models"
	"gorm.io/gorm"
)

type SettingsService struct{}

func (s SettingsService) Get() (models.Settings, error) {
	var settings models.Settings
	result := database.DB.First(&settings)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return models.Settings{}, nil
		}
		return models.Settings{}, Internal("Failed to fetch settings", result.Error)
	}
	if settings.SMTPPass != "" {
		decrypted, err := cryptoService.DecryptIfNeeded(settings.SMTPPass)
		if err != nil {
			return models.Settings{}, err
		}
		settings.SMTPPass = decrypted
	}
	if settings.WebhookSecret != "" {
		decrypted, err := cryptoService.DecryptIfNeeded(settings.WebhookSecret)
		if err != nil {
			return models.Settings{}, err
		}
		settings.WebhookSecret = decrypted
	}
	return settings, nil
}

func (s SettingsService) Save(req models.Settings) error {
	req.WebhookURL = strings.TrimSpace(req.WebhookURL)
	if req.WebhookEnabled && req.WebhookURL == "" {
		return BadRequest("Webhook URL is required", nil)
	}
	if req.WebhookURL != "" {
		validatedURL, err := validateWebhookURL(req.WebhookURL)
		if err != nil {
			return err
		}
		req.WebhookURL = validatedURL
	}
	if req.SMTPPass != "" {
		encrypted, err := cryptoService.EncryptIfNeeded(req.SMTPPass)
		if err != nil {
			return err
		}
		req.SMTPPass = encrypted
	}
	if req.WebhookSecret != "" {
		encrypted, err := cryptoService.EncryptIfNeeded(req.WebhookSecret)
		if err != nil {
			return err
		}
		req.WebhookSecret = encrypted
	}
	var existing models.Settings
	result := database.DB.First(&existing)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			req.ID = 1
			if err := database.DB.Create(&req).Error; err != nil {
				return Internal("Failed to save settings", err)
			}
			return nil
		}
		return Internal("Failed to fetch settings", result.Error)
	}

	existing.SMTPHost = req.SMTPHost
	existing.SMTPPort = req.SMTPPort
	existing.SMTPUser = req.SMTPUser
	// Only update password if a new one is provided
	if req.SMTPPass != "" {
		existing.SMTPPass = req.SMTPPass
	}
	existing.SMTPFrom = req.SMTPFrom
	existing.SMTPFromName = req.SMTPFromName
	existing.WebhookURL = req.WebhookURL
	// Only update webhook secret if a new one is provided
	if req.WebhookSecret != "" {
		existing.WebhookSecret = req.WebhookSecret
	}
	existing.WebhookEnabled = req.WebhookEnabled
	existing.OwnerEmail = req.OwnerEmail

	if err := database.DB.Save(&existing).Error; err != nil {
		return Internal("Failed to save settings", err)
	}

	return nil
}

func (s SettingsService) TestSMTP(req models.Settings) error {
	if req.SMTPHost == "" || req.SMTPPort == "" {
		return BadRequest("SMTP host and port are required", nil)
	}
	if req.SMTPUser == "" || req.SMTPPass == "" {
		return BadRequest("SMTP username and password are required for test", nil)
	}

	addr := req.SMTPHost + ":" + req.SMTPPort
	tlsConfig := &tls.Config{ServerName: req.SMTPHost}

	var client *smtp.Client
	var err error

	if req.SMTPPort == "465" {
		conn, dialErr := tls.Dial("tcp", addr, tlsConfig)
		if dialErr != nil {
			return BadRequest("Failed to connect (SSL)", dialErr)
		}
		client, err = smtp.NewClient(conn, req.SMTPHost)
		if err != nil {
			conn.Close()
			return BadRequest("Failed to create client", err)
		}
	} else {
		client, err = smtp.Dial(addr)
		if err != nil {
			return BadRequest("Failed to connect", err)
		}

		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(tlsConfig); err != nil {
				client.Close()
				return BadRequest("STARTTLS failed", err)
			}
		} else if req.SMTPPort == "587" {
			client.Close()
			return BadRequest("Server does not support STARTTLS on port 587", nil)
		}
	}
	defer client.Close()

	// Try PLAIN auth first, then LOGIN auth as fallback (for Yandex and others)
	auth := smtp.PlainAuth("", req.SMTPUser, req.SMTPPass, req.SMTPHost)
	if err := client.Auth(auth); err != nil {
		// Try LOGIN auth as fallback
		loginAuth := LoginAuth(req.SMTPUser, req.SMTPPass)
		if loginErr := client.Auth(loginAuth); loginErr != nil {
			return BadRequest("Authentication failed", err)
		}
	}

	return nil
}

// LoginAuth implements LOGIN authentication mechanism
type loginAuth struct {
	username, password string
}

func LoginAuth(username, password string) smtp.Auth {
	return &loginAuth{username, password}
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte{}, nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		switch string(fromServer) {
		case "Username:":
			return []byte(a.username), nil
		case "Password:":
			return []byte(a.password), nil
		default:
			return nil, errors.New("unknown LOGIN challenge")
		}
	}
	return nil, nil
}
