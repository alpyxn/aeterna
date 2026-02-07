package services

import (
	"html"
	"regexp"
	"strings"
	"unicode"
)

type ValidationService struct{}

const (
	MaxContentLength = 50000
	MinContentLength = 1
	MaxEmailLength   = 254
)

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
