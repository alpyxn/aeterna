package services

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/alpyxn/aeterna/backend/internal/database"
	"github.com/alpyxn/aeterna/backend/internal/models"
	"gorm.io/gorm"
)

// UserAdminService manages accounts for the primary administrator only.
type UserAdminService struct{}

// UserListItem is a safe view of a user for admin listing.
type UserListItem struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
	IsPrimary bool   `json:"is_primary"`
}

// List returns all accounts when the actor is the primary (first) user.
func (s UserAdminService) List(actorUserID string) ([]UserListItem, error) {
	if !IsFirstUser(actorUserID) {
		return nil, NewAPIError(403, "forbidden", "Only the primary administrator can list users.", nil)
	}

	var first models.User
	if err := database.DB.Order("created_at ASC, id ASC").First(&first).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []UserListItem{}, nil
		}
		return nil, Internal("Failed to resolve primary user", err)
	}

	var users []models.User
	if err := database.DB.Order("created_at ASC, id ASC").Find(&users).Error; err != nil {
		return nil, Internal("Failed to list users", err)
	}

	out := make([]UserListItem, 0, len(users))
	for _, u := range users {
		out = append(out, UserListItem{
			ID:        u.ID,
			Email:     u.Email,
			CreatedAt: u.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
			IsPrimary: u.ID == first.ID,
		})
	}
	return out, nil
}

// Delete removes a non-primary user and all tenant data when the actor is primary.
func (s UserAdminService) Delete(actorUserID, targetUserID string) error {
	if !IsFirstUser(actorUserID) {
		return NewAPIError(403, "forbidden", "Only the primary administrator can delete users.", nil)
	}
	if targetUserID == "" {
		return BadRequest("User id is required", nil)
	}
	if actorUserID == targetUserID {
		return NewAPIError(400, "cannot_delete_self", "You cannot delete your own account.", nil)
	}
	if IsFirstUser(targetUserID) {
		return NewAPIError(400, "cannot_delete_primary", "The primary administrator account cannot be deleted.", nil)
	}

	var target models.User
	if err := database.DB.First(&target, "id = ?", targetUserID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return NotFound("User not found", err)
		}
		return Internal("Failed to load user", err)
	}
	_ = target

	var msgs []models.Message
	if err := database.DB.Unscoped().Where("user_id = ?", targetUserID).Find(&msgs).Error; err != nil {
		return Internal("Failed to list messages for user", err)
	}

	fs := FileService{}
	for _, msg := range msgs {
		if err := fs.DeleteByMessageID(targetUserID, msg.ID); err != nil {
			return err
		}
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		for _, msg := range msgs {
			if err := tx.Unscoped().Where("message_id = ?", msg.ID).Delete(&models.MessageReminder{}).Error; err != nil {
				return Internal("Failed to delete reminders", err)
			}
			if err := tx.Unscoped().Delete(&msg).Error; err != nil {
				return Internal("Failed to delete message", err)
			}
		}
		if err := tx.Unscoped().Where("user_id = ?", targetUserID).Delete(&models.Webhook{}).Error; err != nil {
			return Internal("Failed to delete webhooks", err)
		}
		if err := tx.Unscoped().Where("user_id = ?", targetUserID).Delete(&models.Settings{}).Error; err != nil {
			return Internal("Failed to delete settings", err)
		}
		if err := tx.Unscoped().Where("user_id = ?", targetUserID).Delete(&models.Attachment{}).Error; err != nil {
			return Internal("Failed to delete attachment records", err)
		}
		if err := tx.Unscoped().Delete(&models.User{}, "id = ?", targetUserID).Error; err != nil {
			return Internal("Failed to delete user", err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	removeUserUploadsDir(targetUserID)
	return nil
}

func removeUserUploadsDir(userID string) {
	_ = os.RemoveAll(filepath.Join(GetUploadsDir(), userID))
}
