package services

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"time"

	"github.com/alpyxn/aeterna/backend/internal/models"
)

type EmailService struct{}

var emailCryptoService = CryptoService{}

func (s EmailService) SendTriggeredMessage(settings models.Settings, msg models.Message) error {
	to := msg.RecipientEmail
	subject := "A Message Has Been Left For You"

	content := msg.Content
	if msg.Content != "" {
		decrypted, err := emailCryptoService.Decrypt(msg.Content)
		if err != nil {
			return err
		}
		content = decrypted
	}
	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
</head>
<body style="font-family: Arial, sans-serif; background-color: #0a0a0f; color: #e2e8f0; padding: 40px;">
    <div style="max-width: 600px; margin: 0 auto; background-color: #1a1a2e; border-radius: 12px; padding: 40px; border: 1px solid #334155;">
        <h1 style="color: #22d3ee; margin-bottom: 20px;">A Message For You</h1>
        <p style="font-size: 14px; color: #64748b; margin-bottom: 20px;">
            Someone has left you a message with instructions to deliver it at this time.
        </p>
        <div style="background-color: #0f172a; border-radius: 8px; padding: 24px; border-left: 4px solid #22d3ee; margin-bottom: 30px;">
            <p style="font-size: 16px; line-height: 1.8; margin: 0; white-space: pre-wrap;">%s</p>
        </div>
        <p style="font-size: 12px; color: #475569; margin-top: 40px; text-align: center;">
            This message was sent via Aeterna - Dead Man's Switch
        </p>
    </div>
</body>
</html>`, content)

	from := settings.SMTPFrom
	if from == "" {
		from = settings.SMTPUser
	}
	fromName := settings.SMTPFromName
	if fromName == "" {
		fromName = "Aeterna"
	}

	headers := fmt.Sprintf("From: %s <%s>\r\n", fromName, from)
	headers += fmt.Sprintf("To: %s\r\n", to)
	headers += fmt.Sprintf("Subject: %s\r\n", subject)
	headers += "MIME-Version: 1.0\r\n"
	headers += "Content-Type: text/html; charset=UTF-8\r\n"
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

	auth := smtp.PlainAuth("", settings.SMTPUser, settings.SMTPPass, settings.SMTPHost)
	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("auth failed: %v", err)
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

	auth := smtp.PlainAuth("", settings.SMTPUser, settings.SMTPPass, settings.SMTPHost)
	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("auth failed: %v", err)
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
