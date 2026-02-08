package services

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/smtp"
	"strings"
	"time"

	"github.com/alpyxn/aeterna/backend/internal/models"
)

type EmailService struct{}

var emailCryptoService = CryptoService{}

// sanitizeEmailHeader removes newlines to prevent header injection
func sanitizeEmailHeader(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", "")
	return s
}

func (s EmailService) SendTriggeredMessage(settings models.Settings, msg models.Message) error {
	to := msg.RecipientEmail
	subject := "A message for you"

	content := msg.Content
	if msg.Content != "" {
		decrypted, err := emailCryptoService.Decrypt(msg.Content)
		if err != nil {
			return err
		}
		content = decrypted
	}
	body := fmt.Sprintf(`Someone has arranged for this message to be delivered to you.

---

%s

---

Sent by Aeterna`, content)

	return s.SendPlain(settings, to, subject, body)
}

// SendPlain sends a plain text email
func (s EmailService) SendPlain(settings models.Settings, to, subject, body string) error {
	from := settings.SMTPFrom
	if from == "" {
		from = settings.SMTPUser
	}
	fromName := settings.SMTPFromName
	if fromName == "" {
		fromName = "Aeterna"
	}

	// Sanitize headers to prevent header injection
	from = sanitizeEmailHeader(from)
	fromName = sanitizeEmailHeader(fromName)
	to = sanitizeEmailHeader(to)
	subject = sanitizeEmailHeader(subject)

	headers := fmt.Sprintf("From: %s <%s>\r\n", fromName, from)
	headers += fmt.Sprintf("To: %s\r\n", to)
	headers += fmt.Sprintf("Subject: %s\r\n", subject)
	headers += "MIME-Version: 1.0\r\n"
	headers += "Content-Type: text/plain; charset=UTF-8\r\n"
	headers += "\r\n"

	message := []byte(headers + body)
	addr := settings.SMTPHost + ":" + settings.SMTPPort

	if settings.SMTPPort == "465" {
		return s.sendWithRetry(func() error {
			return s.sendEmailSSL(settings, addr, from, to, message)
		})
	}
	return s.sendWithRetry(func() error {
		return s.sendEmailSTARTTLS(settings, addr, from, to, message)
	})
}


func (s EmailService) sendWithRetry(sendFn func() error) error {
	const maxAttempts = 3
	baseDelay := 500 * time.Millisecond

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := sendFn(); err == nil {
			return nil
		} else {
			lastErr = err
		}

		if attempt < maxAttempts {
			backoff := baseDelay * time.Duration(1<<(attempt-1))
			time.Sleep(backoff)
		}
	}

	return lastErr
}

// authWithFallback tries PLAIN auth first, then LOGIN auth as fallback
func authWithFallback(client *smtp.Client, username, password, host string) error {
	// Try PLAIN auth first
	auth := smtp.PlainAuth("", username, password, host)
	if err := client.Auth(auth); err != nil {
		// Try LOGIN auth as fallback (for Yandex and others)
		loginAuth := &emailLoginAuth{username, password}
		if loginErr := client.Auth(loginAuth); loginErr != nil {
			return fmt.Errorf("auth failed (PLAIN: %v, LOGIN: %v)", err, loginErr)
		}
	}
	return nil
}

func (s EmailService) sendEmailSSL(settings models.Settings, addr, from, to string, message []byte) error {
	tlsConfig := &tls.Config{ServerName: settings.SMTPHost}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("TLS dial failed: %v", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, settings.SMTPHost)
	if err != nil {
		return fmt.Errorf("SMTP client failed: %v", err)
	}
	defer client.Quit()

	if err = authWithFallback(client, settings.SMTPUser, settings.SMTPPass, settings.SMTPHost); err != nil {
		return err
	}

	if err = client.Mail(from); err != nil {
		return fmt.Errorf("MAIL FROM failed: %v", err)
	}

	if err = client.Rcpt(to); err != nil {
		return fmt.Errorf("RCPT TO failed: %v", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("DATA failed: %v", err)
	}

	_, err = w.Write(message)
	if err != nil {
		return fmt.Errorf("write failed: %v", err)
	}

	return w.Close()
}

func (s EmailService) sendEmailSTARTTLS(settings models.Settings, addr, from, to string, message []byte) error {
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("dial failed: %v", err)
	}
	defer client.Quit()

	tlsConfig := &tls.Config{ServerName: settings.SMTPHost}
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err = client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("STARTTLS failed: %v", err)
		}
	}

	if err = authWithFallback(client, settings.SMTPUser, settings.SMTPPass, settings.SMTPHost); err != nil {
		return err
	}

	if err = client.Mail(from); err != nil {
		return fmt.Errorf("MAIL FROM failed: %v", err)
	}

	if err = client.Rcpt(to); err != nil {
		return fmt.Errorf("RCPT TO failed: %v", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("DATA failed: %v", err)
	}

	_, err = w.Write(message)
	if err != nil {
		return fmt.Errorf("write failed: %v", err)
	}

	return w.Close()
}

// emailLoginAuth implements LOGIN authentication mechanism
type emailLoginAuth struct {
	username, password string
}

func (a *emailLoginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte{}, nil
}

func (a *emailLoginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
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

