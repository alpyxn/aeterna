package services

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"io"
	"log/slog"
	"sync"
)

type CryptoService struct{}

const cryptoPrefix = "enc:"

var (
	keyManager     *KeySourceManager
	keyManagerOnce sync.Once
	cachedKey      string
	cachedKeyOnce  sync.Once
	keySourceName  string
)

// InitKeyManager initializes the key manager with the given encryption key file path
// This should be called once at application startup
func InitKeyManager(encryptionKeyFile string) {
	keyManagerOnce.Do(func() {
		keyManager = NewKeySourceManager(encryptionKeyFile)
		// Try to get the key once to cache it and log which source was used
		key, err := keyManager.GetKey()
		if err == nil {
			cachedKey = key
			keySourceName = keyManager.GetSourceName()
			slog.Info("Encryption key loaded", "source", keySourceName)
		}
	})
}

func (s CryptoService) getOrCreateKey() (string, error) {
	// Use cached key if available (thread-safe)
	if cachedKey != "" {
		return cachedKey, nil
	}

	// If not cached, try to get from manager
	if keyManager == nil {
		return "", Internal("Encryption key manager not initialized. Call InitKeyManager() at startup.", nil)
	}

	key, err := keyManager.GetKey()
	if err != nil {
		return "", Internal("Failed to retrieve encryption key", err)
	}

	// Cache the key for future use
	cachedKeyOnce.Do(func() {
		cachedKey = key
		keySourceName = keyManager.GetSourceName()
		slog.Info("Encryption key loaded", "source", keySourceName)
	})

	return key, nil
}

func (s CryptoService) Encrypt(plaintext string) (string, error) {
	keyBase64, err := s.getOrCreateKey()
	if err != nil {
		return "", err
	}

	key, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		return "", Internal("Invalid encryption key", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", Internal("Failed to create cipher", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", Internal("Failed to create GCM", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", Internal("Failed to generate nonce", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (s CryptoService) Decrypt(encoded string) (string, error) {
	keyBase64, err := s.getOrCreateKey()
	if err != nil {
		return "", err
	}

	key, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		return "", Internal("Invalid encryption key", err)
	}

	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", Internal("Invalid ciphertext", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", Internal("Failed to create cipher", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", Internal("Failed to create GCM", err)
	}

	if len(data) < gcm.NonceSize() {
		return "", Internal("Invalid ciphertext length", nil)
	}

	nonce := data[:gcm.NonceSize()]
	ciphertext := data[gcm.NonceSize():]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", Internal("Failed to decrypt message", err)
	}

	return string(plaintext), nil
}

// EncryptBytes encrypts raw binary data and returns the ciphertext as bytes (nonce prepended)
func (s CryptoService) EncryptBytes(plaintext []byte) ([]byte, error) {
	keyBase64, err := s.getOrCreateKey()
	if err != nil {
		return nil, err
	}

	key, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		return nil, Internal("Invalid encryption key", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, Internal("Failed to create cipher", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, Internal("Failed to create GCM", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, Internal("Failed to generate nonce", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// DecryptBytes decrypts raw binary ciphertext (nonce prepended) and returns the plaintext bytes
func (s CryptoService) DecryptBytes(ciphertext []byte) ([]byte, error) {
	keyBase64, err := s.getOrCreateKey()
	if err != nil {
		return nil, err
	}

	key, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		return nil, Internal("Invalid encryption key", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, Internal("Failed to create cipher", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, Internal("Failed to create GCM", err)
	}

	if len(ciphertext) < gcm.NonceSize() {
		return nil, Internal("Invalid ciphertext length", nil)
	}

	nonce := ciphertext[:gcm.NonceSize()]
	data := ciphertext[gcm.NonceSize():]

	plaintext, err := gcm.Open(nil, nonce, data, nil)
	if err != nil {
		return nil, Internal("Failed to decrypt data", err)
	}

	return plaintext, nil
}

func (s CryptoService) EncryptIfNeeded(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	if len(plaintext) >= len(cryptoPrefix) && plaintext[:len(cryptoPrefix)] == cryptoPrefix {
		return plaintext, nil
	}
	enc, err := s.Encrypt(plaintext)
	if err != nil {
		return "", err
	}
	return cryptoPrefix + enc, nil
}

func (s CryptoService) DecryptIfNeeded(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	if len(value) >= len(cryptoPrefix) && value[:len(cryptoPrefix)] == cryptoPrefix {
		return s.Decrypt(value[len(cryptoPrefix):])
	}
	return value, nil
}

func (s CryptoService) GenerateToken(length int) (string, error) {
	buf := make([]byte, length)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return "", Internal("Failed to generate token", err)
	}
	return base64.URLEncoding.EncodeToString(buf), nil
}
