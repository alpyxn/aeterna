package services

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"
)

// KeySource defines the interface for retrieving encryption keys
type KeySource interface {
	// GetKey retrieves the encryption key from the source
	GetKey() (string, error)
	// Name returns the name of the key source for logging
	Name() string
	// Available checks if this key source is available/configured
	Available() bool
}

// KeySourceManager manages key sources and tries them in priority order
type KeySourceManager struct {
	sources []KeySource
}

// NewKeySourceManager creates a new manager with automatic source detection
func NewKeySourceManager(encryptionKeyFile string) *KeySourceManager {
	manager := &KeySourceManager{
		sources: []KeySource{},
	}

	// 1. Docker Secrets (production - auto-detected)
	if _, err := os.Stat("/run/secrets/encryption_key"); err == nil {
		manager.sources = append(manager.sources, &DockerSecretKeySource{
			path: "/run/secrets/encryption_key",
		})
	}

	// 2. Secure File (fallback / development)
	if encryptionKeyFile != "" {
		manager.sources = append(manager.sources, &FileKeySource{
			path: encryptionKeyFile,
		})
	}

	return manager
}

// GetKey tries all available sources in priority order
func (m *KeySourceManager) GetKey() (string, error) {
	var lastErr error
	var triedSources []string

	for _, source := range m.sources {
		if !source.Available() {
			continue
		}

		key, err := source.GetKey()
		if err == nil {
			// Validate key format
			if _, err := ValidateKeyFormat(key); err != nil {
				return "", fmt.Errorf("invalid key format from %s: %w", source.Name(), err)
			}
			return key, nil
		}

		triedSources = append(triedSources, source.Name())
		lastErr = err
	}

	if len(triedSources) == 0 {
		return "", fmt.Errorf("no encryption key source available. Use Docker secrets or --encryption-key-file flag")
	}

	return "", fmt.Errorf("failed to retrieve encryption key from any source (tried: %s). Last error: %w", strings.Join(triedSources, ", "), lastErr)
}

// GetSourceName returns the name of the source that successfully provided the key
func (m *KeySourceManager) GetSourceName() string {
	for _, source := range m.sources {
		if !source.Available() {
			continue
		}
		key, err := source.GetKey()
		if err == nil {
			if _, validateErr := ValidateKeyFormat(key); validateErr == nil {
				return source.Name()
			}
		}
	}
	return "unknown"
}

// DockerSecretKeySource retrieves key from Docker secrets
type DockerSecretKeySource struct {
	path string
}

func (s *DockerSecretKeySource) Name() string { return "Docker Secrets" }
func (s *DockerSecretKeySource) Available() bool {
	_, err := os.Stat(s.path)
	return err == nil
}

func (s *DockerSecretKeySource) GetKey() (string, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return "", fmt.Errorf("failed to read Docker secret: %w", err)
	}

	key := strings.TrimSpace(string(data))
	return key, nil
}

// FileKeySource retrieves key from a secure file
type FileKeySource struct {
	path string
}

func (s *FileKeySource) Name() string { return "Secure File" }
func (s *FileKeySource) Available() bool {
	_, err := os.Stat(s.path)
	return err == nil
}

func (s *FileKeySource) GetKey() (string, error) {
	// Validate file permissions (must be 0600)
	info, err := os.Stat(s.path)
	if err != nil {
		return "", fmt.Errorf("failed to stat key file: %w", err)
	}

	mode := info.Mode().Perm()
	if mode != 0600 {
		return "", fmt.Errorf("key file %s has insecure permissions %04o (must be 0600)", s.path, mode)
	}

	data, err := os.ReadFile(s.path)
	if err != nil {
		return "", fmt.Errorf("failed to read key file: %w", err)
	}

	key := strings.TrimSpace(string(data))
	return key, nil
}

// ValidateKeyFormat validates that the key is base64 encoded 32 bytes
// Returns the decoded key bytes if valid, or an error if invalid
func ValidateKeyFormat(key string) ([]byte, error) {
	if key == "" {
		return nil, fmt.Errorf("key is empty")
	}

	decoded, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return nil, fmt.Errorf("key is not valid base64: %w", err)
	}

	if len(decoded) != 32 {
		return nil, fmt.Errorf("key length is %d bytes (must be 32 bytes)", len(decoded))
	}

	return decoded, nil
}

// GenerateKey generates a new encryption key
func GenerateKey() (string, error) {
	buf := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return "", fmt.Errorf("failed to generate random key: %w", err)
	}
	return base64.StdEncoding.EncodeToString(buf), nil
}
