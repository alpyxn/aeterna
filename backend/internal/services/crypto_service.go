package services

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"io"
	"os"
)

type CryptoService struct{}

const cryptoPrefix = "enc:"

func (s CryptoService) getOrCreateKey() (string, error) {
	envKey := os.Getenv("ENCRYPTION_KEY")
	if envKey == "" {
		return "", Internal("ENCRYPTION_KEY environment variable is required", nil)
	}
	
	// Validate key length (should be base64 encoded 32 bytes)
	decoded, err := base64.StdEncoding.DecodeString(envKey)
	if err != nil || len(decoded) != 32 {
		return "", Internal("Invalid ENCRYPTION_KEY format (must be base64 encoded 32 bytes)", nil)
	}
	
	return envKey, nil
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

func (s CryptoService) GenerateToken(length int) (string, error) {
	buf := make([]byte, length)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return "", Internal("Failed to generate token", err)
	}
	return base64.URLEncoding.EncodeToString(buf), nil
}

