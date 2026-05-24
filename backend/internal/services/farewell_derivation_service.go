package services

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/alpyxn/aeterna/backend/internal/database"
	"github.com/alpyxn/aeterna/backend/internal/models"
	"github.com/alpyxn/aeterna/backend/internal/ports"
)

const defaultFarewellDerivationBatchSize = 50

type FarewellDerivationService struct {
	crypto CryptoService
}

func NewFarewellDerivationService() ports.FarewellDerivationPort {
	return &FarewellDerivationService{crypto: CryptoService{}}
}

func (s *FarewellDerivationService) ProcessPending(batchSize int) (int, error) {
	if batchSize <= 0 {
		batchSize = defaultFarewellDerivationBatchSize
	}

	letters := make([]models.FarewellLetter, 0, batchSize)
	err := database.DB.
		Where("derivatives_pending = ?", true).
		Where("deleted_at IS NULL").
		Order("updated_at ASC").
		Limit(batchSize).
		Find(&letters).Error
	if err != nil {
		return 0, fmt.Errorf("failed to load pending farewell derivations: %w", err)
	}

	processed := 0
	for _, letter := range letters {
		if letter.UserID == "" {
			continue
		}
		if err := s.deriveLetter(letter); err != nil {
			slog.Error("Failed to process farewell derivation", "letter_id", letter.ID, "error", err)
			continue
		}
		processed++
	}

	return processed, nil
}

func (s *FarewellDerivationService) deriveLetter(letter models.FarewellLetter) error {
	contentCipher := letter.RawContent
	if strings.TrimSpace(contentCipher) == "" {
		contentCipher = letter.Content
	}
	if strings.TrimSpace(contentCipher) == "" {
		return nil
	}

	rawMarkdown, err := s.crypto.Decrypt(contentCipher)
	if err != nil {
		return fmt.Errorf("failed to decrypt raw content: %w", err)
	}

	safeMarkdown := sanitizeFarewellMarkdown(rawMarkdown)
	renderedHTML := markdownToHTML(safeMarkdown)
	wordCount := countWordsFromMarkdown(safeMarkdown)

	encryptedSafe, err := s.crypto.Encrypt(safeMarkdown)
	if err != nil {
		return fmt.Errorf("failed to encrypt sanitized content: %w", err)
	}

	encryptedRaw, err := s.crypto.Encrypt(rawMarkdown)
	if err != nil {
		return fmt.Errorf("failed to encrypt raw content: %w", err)
	}

	encryptedHTML, err := s.crypto.Encrypt(renderedHTML)
	if err != nil {
		return fmt.Errorf("failed to encrypt rendered html: %w", err)
	}

	updates := map[string]any{
		"encrypted_content":       encryptedSafe,
		"encrypted_content_raw":   encryptedRaw,
		"encrypted_rendered_html": encryptedHTML,
		"word_count":              wordCount,
		"derivatives_pending":     false,
	}

	if err := database.ForTenant(letter.UserID).
		Model(&models.FarewellLetter{}).
		Where("id = ?", letter.ID).
		Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to persist derived fields: %w", err)
	}

	return nil
}
