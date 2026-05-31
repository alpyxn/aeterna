package migrations

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/alpyxn/aeterna/backend/internal/config"
	"github.com/alpyxn/aeterna/backend/internal/models"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func uploadsBaseDir(dbPath string) string {
	return filepath.Join(filepath.Dir(dbPath), "uploads")
}

// MigrateLegacyToMultitenant assigns a single legacy user to existing rows when upgrading
// from single-tenant installs. Safe to call on every startup (idempotent).
func MigrateLegacyToMultitenant(db *gorm.DB, cfg config.Config) error {
	dbPath := cfg.Database.Path
	masterPassword := cfg.Auth.MasterPassword
	var userCount int64
	if err := db.Model(&models.User{}).Count(&userCount).Error; err != nil {
		return err
	}
	if userCount > 0 {
		return backfillOrphanUserIDs(db)
	}

	var settings models.Settings
	err := db.First(&settings).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if strings.TrimSpace(settings.UserID) != "" {
		return nil
	}

	email := strings.TrimSpace(settings.OwnerEmail)
	if email == "" {
		email = "legacy@aeterna.local"
	}
	email = strings.ToLower(email)

	pwdHash := settings.MasterPasswordHash
	if pwdHash == "" {
		if env := masterPassword; env != "" {
			h, err := bcrypt.GenerateFromPassword([]byte(env), bcrypt.DefaultCost)
			if err != nil {
				return err
			}
			pwdHash = string(h)
		} else {
			h, err := bcrypt.GenerateFromPassword([]byte(uuid.NewString()), bcrypt.DefaultCost)
			if err != nil {
				return err
			}
			pwdHash = string(h)
		}
	}

	user := models.User{
		Email:        email,
		PasswordHash: pwdHash,
	}
	if err := db.Create(&user).Error; err != nil {
		return err
	}

	if err := db.Model(&settings).Update("user_id", user.ID).Error; err != nil {
		return err
	}

	uid := user.ID
	if err := db.Model(&models.Message{}).Where("user_id = ? OR TRIM(COALESCE(user_id, '')) = ?", "", "").Update("user_id", uid).Error; err != nil {
		return err
	}
	if err := db.Model(&models.Webhook{}).Where("user_id = ? OR TRIM(COALESCE(user_id, '')) = ?", "", "").Update("user_id", uid).Error; err != nil {
		return err
	}
	if err := db.Model(&models.Attachment{}).Where("user_id = ? OR TRIM(COALESCE(user_id, '')) = ?", "", "").Update("user_id", uid).Error; err != nil {
		return err
	}

	if err := migrateLegacyUploadPaths(db, uid, dbPath); err != nil {
		log.Println("Warning: legacy upload path migration:", err)
	}

	log.Println("Multi-tenant migration: assigned legacy user", user.ID, "email", user.Email)
	return nil
}

func backfillOrphanUserIDs(db *gorm.DB) error {
	var first models.User
	if err := db.Order("created_at ASC").First(&first).Error; err != nil {
		return err
	}
	uid := first.ID
	q := "(user_id IS NULL OR TRIM(COALESCE(user_id, '')) = '')"
	if err := db.Model(&models.Message{}).Where(q).Update("user_id", uid).Error; err != nil {
		return err
	}
	if err := db.Model(&models.Webhook{}).Where(q).Update("user_id", uid).Error; err != nil {
		return err
	}
	return db.Model(&models.Attachment{}).Where(q).Update("user_id", uid).Error
}

func migrateLegacyUploadPaths(db *gorm.DB, legacyUserID, dbPath string) error {
	base := uploadsBaseDir(dbPath)
	entries, err := os.ReadDir(base)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if name == legacyUserID {
			continue
		}
		var msg models.Message
		if err := db.First(&msg, "id = ?", name).Error; err != nil {
			continue
		}
		if msg.UserID != legacyUserID {
			continue
		}
		oldDir := filepath.Join(base, name)
		newDir := filepath.Join(base, legacyUserID, name)
		if err := os.MkdirAll(filepath.Dir(newDir), 0700); err != nil {
			return err
		}
		if err := os.Rename(oldDir, newDir); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}

		var attachments []models.Attachment
		if err := db.Where("message_id = ?", name).Find(&attachments).Error; err != nil {
			return err
		}
		for _, att := range attachments {
			newPath := filepath.Join(newDir, filepath.Base(att.StoragePath))
			if err := db.Model(&att).Update("storage_path", newPath).Error; err != nil {
				return err
			}
		}
	}
	return nil
}
