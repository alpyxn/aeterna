package services

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"

	"github.com/alpyxn/aeterna/backend/internal/database"
	"github.com/alpyxn/aeterna/backend/internal/models"
	"gorm.io/gorm"
)

type CryptoService struct{}

const cryptoPrefix = "enc:"

func (s CryptoService) getOrCreateKey() (string, error) {
	var settings models.Settings
	result := database.DB.First(&settings)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			key, err := generateKey()
			if err != nil {
				return "", Internal("Failed to generate encryption key", err)
			}
			settings = models.Settings{
				ID:            1,
				EncryptionKey: key,
			}
			if err := database.DB.Create(&settings).Error; err != nil {
				return "", Internal("Failed to save encryption key", err)
			}
			return key, nil
		}
		return "", Internal("Failed to fetch settings", result.Error)
	}

	if settings.EncryptionKey == "" {
		key, err := generateKey()
		if err != nil {
			return "", Internal("Failed to generate encryption key", err)
		}
		settings.EncryptionKey = key
		if err := database.DB.Save(&settings).Error; err != nil {
			return "", Internal("Failed to save encryption key", err)
		}
		return key, nil
	}

	return settings.EncryptionKey, nil
}

func generateKey() (string, error) {
	buf := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf), nil
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
