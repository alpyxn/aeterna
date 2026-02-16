package services

import (
	"html"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

type ValidationService struct{}

const (
	MaxContentLength     = 50000
	MinContentLength     = 1
	MaxEmailLength       = 254
	MaxFileSize          = 10 * 1024 * 1024 // 10 MB
	MaxTotalAttachSize   = 25 * 1024 * 1024 // 25 MB
	MaxAttachmentsPerMsg = 5
)

var AllowedExtensions = map[string]bool{
	".pdf":  true,
	".txt":  true,
	".doc":  true,
	".docx": true,
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".webp": true,
	".zip":  true,
}

var AllowedMIMEPrefixes = []string{
	"application/pdf",
	"text/plain",
	"application/msword",
	"application/vnd.openxmlformats",
	"image/jpeg",
	"image/png",
	"image/gif",
	"image/webp",
	"application/zip",
	"application/octet-stream", // fallback for unknown binary types like .docx
}

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)

func (s ValidationService) ValidateEmail(email string) error {
	email = strings.TrimSpace(email)

	if email == "" {
		return BadRequest("Email is required", nil)
	}

	if len(email) > MaxEmailLength {
		return BadRequest("Email address is too long", nil)
	}

	if !emailRegex.MatchString(email) {
		return BadRequest("Invalid email format", nil)
	}

	// Check for common dangerous patterns
	lowerEmail := strings.ToLower(email)
	dangerousPatterns := []string{"<script", "javascript:", "data:", "vbscript:"}
	for _, pattern := range dangerousPatterns {
		if strings.Contains(lowerEmail, pattern) {
			return BadRequest("Invalid email format", nil)
		}
	}

	return nil
}

func (s ValidationService) ValidateContent(content string) error {
	if len(content) < MinContentLength {
		return BadRequest("Content is required", nil)
	}

	if len(content) > MaxContentLength {
		return BadRequest("Content exceeds maximum length of 50000 characters", nil)
	}

	return nil
}

func (s ValidationService) SanitizeContent(content string) string {
	// HTML escape to prevent XSS
	sanitized := html.EscapeString(content)
	return sanitized
}

func (s ValidationService) ValidatePassword(password string) error {
	if len(password) < 8 {
		return BadRequest("Password must be at least 8 characters", nil)
	}

	if len(password) > 128 {
		return BadRequest("Password exceeds maximum length", nil)
	}

	var (
		hasUpper   bool
		hasLower   bool
		hasNumber  bool
		hasSpecial bool
	)

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	if !hasUpper {
		return BadRequest("Password must contain at least one uppercase letter", nil)
	}
	if !hasLower {
		return BadRequest("Password must contain at least one lowercase letter", nil)
	}
	if !hasNumber {
		return BadRequest("Password must contain at least one number", nil)
	}
	if !hasSpecial {
		return BadRequest("Password must contain at least one special character (!@#$%^&* etc.)", nil)
	}

	return nil
}

// ValidateTriggerDuration validates the trigger duration in minutes
func (s ValidationService) ValidateTriggerDuration(duration int) error {
	if duration < 1 {
		return BadRequest("Duration must be at least 1 minute", nil)
	}
	if duration > 525600 {
		return BadRequest("Duration cannot exceed 1 year (525600 minutes)", nil)
	}
	return nil
}

// ValidateFile validates a file upload: extension, MIME type, and size
func (s ValidationService) ValidateFile(filename string, size int64, data []byte) error {
	if size == 0 {
		return BadRequest("File is empty", nil)
	}
	if size > MaxFileSize {
		return BadRequest("File exceeds maximum size of 10 MB", nil)
	}

	// Check extension
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		return BadRequest("File must have an extension", nil)
	}
	if !AllowedExtensions[ext] {
		return BadRequest("File type not allowed. Allowed: PDF, TXT, DOC, DOCX, JPG, PNG, GIF, WEBP, ZIP", nil)
	}

	// Check MIME type via content sniffing (first 512 bytes)
	detectedMIME := http.DetectContentType(data)
	mimeAllowed := false
	for _, prefix := range AllowedMIMEPrefixes {
		if strings.HasPrefix(detectedMIME, prefix) {
			mimeAllowed = true
			break
		}
	}
	if !mimeAllowed {
		return BadRequest("File content type not allowed", nil)
	}

	return nil
}

// SanitizeFilename cleans a filename to prevent path traversal and other attacks
func (s ValidationService) SanitizeFilename(filename string) string {
	// Extract just the base name (no directory path)
	filename = filepath.Base(filename)

	// Remove null bytes and control characters
	cleaned := strings.Map(func(r rune) rune {
		if r == 0 || unicode.IsControl(r) {
			return -1
		}
		return r
	}, filename)

	// Replace path separators
	cleaned = strings.ReplaceAll(cleaned, "/", "_")
	cleaned = strings.ReplaceAll(cleaned, "\\", "_")

	// Remove leading dots (hidden files)
	cleaned = strings.TrimLeft(cleaned, ".")

	if cleaned == "" {
		cleaned = "unnamed_file"
	}

	// Limit length
	if len(cleaned) > 255 {
		ext := filepath.Ext(cleaned)
		name := cleaned[:255-len(ext)]
		cleaned = name + ext
	}

	return cleaned
}
