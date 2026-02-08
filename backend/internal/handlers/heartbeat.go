package handlers

import (
	"crypto/subtle"
	"time"

	"github.com/alpyxn/aeterna/backend/internal/database"
	"github.com/alpyxn/aeterna/backend/internal/models"
	"github.com/alpyxn/aeterna/backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

var settingsService = services.SettingsService{}

// QuickHeartbeat handles heartbeat via token link (no auth required)
func QuickHeartbeat(c *fiber.Ctx) error {
	token := c.Params("token")
	if token == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Token required"})
	}

	// Get settings and verify token
	settings, err := settingsService.Get()
	if err != nil {
		return writeError(c, err)
	}

	if settings.HeartbeatToken == "" || subtle.ConstantTimeCompare([]byte(settings.HeartbeatToken), []byte(token)) != 1 {
		return c.Status(403).JSON(fiber.Map{"error": "Invalid token"})
	}

	// Update all active messages
	now := time.Now().UTC()
	result := database.DB.Model(&models.Message{}).
		Where("status = ?", models.StatusActive).
		Updates(map[string]interface{}{
			"last_seen":     now,
			"reminder_sent": false,
		})

	if result.Error != nil {
		return writeError(c, services.Internal("Failed to update heartbeats", result.Error))
	}

	// Return success HTML page - minimal style
	html := `<!DOCTYPE html>
<html>
<head>
    <title>Confirmed - Aeterna</title>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>
        body { 
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: #fafafa; 
            color: #333; 
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            margin: 0;
        }
        .container {
            text-align: center;
            padding: 2rem;
            max-width: 400px;
        }
        h1 { font-size: 1.25rem; font-weight: 500; margin-bottom: 0.5rem; }
        p { color: #666; font-size: 0.9rem; }
        .footer { margin-top: 2rem; font-size: 0.75rem; color: #999; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Confirmed</h1>
        <p>Your check-in has been recorded.</p>
        <p class="footer">Aeterna</p>
    </div>
</body>
</html>`

	c.Set("Content-Type", "text/html")
	return c.SendString(html)
}

// GetHeartbeatToken returns the heartbeat token for authenticated users
func GetHeartbeatToken(c *fiber.Ctx) error {
	settings, err := settingsService.Get()
	if err != nil {
		return writeError(c, err)
	}

	return c.JSON(fiber.Map{
		"token": settings.HeartbeatToken,
	})
}
