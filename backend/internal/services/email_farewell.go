package services

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"mime"
	"strings"

	"github.com/alpyxn/aeterna/backend/internal/models"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

// markdownToHTML converts Markdown content to an HTML string.
func markdownToHTML(md string) string {
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse([]byte(md))

	// Harden rendered HTML:
	// - Drop raw HTML from user markdown
	// - Allow only safe links
	// - Add rel attributes for external links
	opts := html.RendererOptions{
		Flags: html.CommonFlags |
			html.HrefTargetBlank |
			html.SkipHTML |
			html.Safelink |
			html.NofollowLinks |
			html.NoreferrerLinks |
			html.NoopenerLinks,
	}
	renderer := html.NewRenderer(opts)

	return string(markdown.Render(doc, renderer))
}

// SendFarewellLetter sends a farewell letter as a multipart HTML email with optional attachments.
// The content is written in Markdown and rendered to HTML; plain text is sent as fallback.
func (s EmailService) SendFarewellLetter(settings models.Settings, recipientEmail, subject, mdContent string, attachments []EmailAttachment) error {
	htmlBody := markdownToHTML(mdContent)
	plainBody := markdownToPlainText(mdContent)
	if plainBody == "" {
		plainBody = mdContent
	}
	return s.sendFarewellLetterWithBodies(settings, recipientEmail, subject, plainBody, htmlBody, attachments)
}

func (s EmailService) SendFarewellLetterPreRendered(settings models.Settings, recipientEmail, subject, safeMarkdown, renderedHTML string, attachments []EmailAttachment) error {
	plainBody := markdownToPlainText(safeMarkdown)
	if plainBody == "" {
		plainBody = safeMarkdown
	}
	htmlBody := renderedHTML
	if strings.TrimSpace(htmlBody) == "" {
		htmlBody = markdownToHTML(safeMarkdown)
	}
	return s.sendFarewellLetterWithBodies(settings, recipientEmail, subject, plainBody, htmlBody, attachments)
}

func (s EmailService) sendFarewellLetterWithBodies(settings models.Settings, recipientEmail, subject, plainBody, htmlBody string, attachments []EmailAttachment) error {
	from := settings.SMTPFrom
	if from == "" {
		from = settings.SMTPUser
	}
	fromName := settings.SMTPFromName
	if fromName == "" {
		fromName = "Aeterna"
	}

	from = sanitizeEmailHeader(from)
	fromName = sanitizeEmailHeader(fromName)
	subject = sanitizeEmailHeader(subject)
	recipient := sanitizeEmailHeader(recipientEmail)

	outerBoundary := "==AeternaFarewellMixed=="
	altBoundary := "==AeternaFarewellAlt=="

	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("From: %s <%s>\r\n", fromName, from))
	buf.WriteString(fmt.Sprintf("To: %s\r\n", recipient))
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	buf.WriteString("MIME-Version: 1.0\r\n")

	if len(attachments) > 0 {
		// multipart/mixed wrapping multipart/alternative + attachments
		buf.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\r\n\r\n", outerBoundary))

		buf.WriteString(fmt.Sprintf("--%s\r\n", outerBoundary))
		buf.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n\r\n", altBoundary))

		writeAlternativeParts(&buf, altBoundary, plainBody, htmlBody)

		buf.WriteString(fmt.Sprintf("--%s--\r\n", altBoundary))
		buf.WriteString("\r\n")

		for _, att := range attachments {
			buf.WriteString(fmt.Sprintf("--%s\r\n", outerBoundary))
			buf.WriteString(fmt.Sprintf("Content-Type: %s; name=\"%s\"\r\n", att.MimeType, mime.QEncoding.Encode("utf-8", att.Filename)))
			buf.WriteString("Content-Transfer-Encoding: base64\r\n")
			buf.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n\r\n", mime.QEncoding.Encode("utf-8", att.Filename)))
			encoded := base64.StdEncoding.EncodeToString(att.Data)
			for i := 0; i < len(encoded); i += 76 {
				end := i + 76
				if end > len(encoded) {
					end = len(encoded)
				}
				buf.WriteString(encoded[i:end])
				buf.WriteString("\r\n")
			}
		}
		buf.WriteString(fmt.Sprintf("--%s--\r\n", outerBoundary))
	} else {
		// multipart/alternative only
		buf.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n\r\n", altBoundary))
		writeAlternativeParts(&buf, altBoundary, plainBody, htmlBody)
		buf.WriteString(fmt.Sprintf("--%s--\r\n", altBoundary))
	}

	message := buf.Bytes()
	return s.sendRaw(settings, from, []string{recipient}, message)
}

func writeAlternativeParts(buf *bytes.Buffer, boundary, plainBody, htmlBody string) {
	buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	buf.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	buf.WriteString("Content-Transfer-Encoding: 7bit\r\n\r\n")
	buf.WriteString(plainBody)
	buf.WriteString("\r\n")

	buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	buf.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	buf.WriteString("Content-Transfer-Encoding: 7bit\r\n\r\n")
	buf.WriteString(htmlBody)
	buf.WriteString("\r\n")
}
