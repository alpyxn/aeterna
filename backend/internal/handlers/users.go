package handlers

import (
	"github.com/alpyxn/aeterna/backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

var userAdminService = services.UserAdminService{}

// ListUsers returns all accounts (primary administrator only).
func ListUsers(c *fiber.Ctx) error {
	actorID, err := currentUserID(c)
	if err != nil {
		return writeError(c, err)
	}
	users, err := userAdminService.List(actorID)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(users)
}

// DeleteUser removes a non-primary user (primary administrator only).
func DeleteUser(c *fiber.Ctx) error {
	actorID, err := currentUserID(c)
	if err != nil {
		return writeError(c, err)
	}
	targetID := c.Params("id")
	if err := userAdminService.Delete(actorID, targetID); err != nil {
		return writeError(c, err)
	}
	return c.JSON(fiber.Map{"success": true})
}
