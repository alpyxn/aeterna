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
// GET: Shows a page with a button to send heartbeat
// POST: Actually triggers the heartbeat
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

	// Handle POST request - trigger heartbeat
	if c.Method() == "POST" {
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

		// Return success HTML page
		html := `<!DOCTYPE html>
<html>
<head>
    <title>Heartbeat Confirmed - Aeterna</title>
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
        <h1>✓ Heartbeat Confirmed</h1>
        <p>Your check-in has been recorded.</p>
        <p class="footer">Aeterna</p>
    </div>
</body>
</html>
`
		c.Set("Content-Type", "text/html")
		return c.SendString(html)
	}

	// Handle GET request - show page with button
	html := `<!DOCTYPE html>
<html>
<head>
    <title>Send Heartbeat - Aeterna</title>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>
        body { 
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: #333; 
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            margin: 0;
            padding: 1rem;
        }
        .container {
            background: white;
            border-radius: 12px;
            box-shadow: 0 10px 40px rgba(0,0,0,0.1);
            text-align: center;
            padding: 3rem 2rem;
            max-width: 400px;
            width: 100%;
        }
        h1 { 
            font-size: 1.5rem; 
            font-weight: 600; 
            margin-bottom: 0.5rem;
            color: #1a1a1a;
        }
        p { 
            color: #666; 
            font-size: 0.95rem;
            margin-bottom: 2rem;
            line-height: 1.5;
        }
        .button {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            border: none;
            padding: 1rem 2rem;
            font-size: 1rem;
            font-weight: 600;
            border-radius: 8px;
            cursor: pointer;
            width: 100%;
            transition: transform 0.2s, box-shadow 0.2s;
            box-shadow: 0 4px 12px rgba(102, 126, 234, 0.4);
        }
        .button:hover {
            transform: translateY(-2px);
            box-shadow: 0 6px 20px rgba(102, 126, 234, 0.5);
        }
        .button:active {
            transform: translateY(0);
        }
        .button:disabled {
            opacity: 0.6;
            cursor: not-allowed;
            transform: none;
        }
        .footer { 
            margin-top: 2rem; 
            font-size: 0.75rem; 
            color: #999; 
        }
        .loading {
            display: none;
            margin-top: 1rem;
            color: #667eea;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Send Heartbeat</h1>
        <p>Click the button below to confirm you are available and reset your dead man's switch timer.</p>
        <form id="heartbeatForm" method="POST">
            <button type="submit" class="button" id="heartbeatButton">
                Send Heartbeat
            </button>
            <div class="loading" id="loading">Sending...</div>
        </form>
        <p class="footer">Aeterna</p>
    </div>
    <script>
        document.getElementById('heartbeatForm').addEventListener('submit', function(e) {
            e.preventDefault();
            const button = document.getElementById('heartbeatButton');
            const loading = document.getElementById('loading');
            
            button.disabled = true;
            loading.style.display = 'block';
            
            fetch(window.location.href, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                }
            })
            .then(response => {
                if (response.ok) {
                    return response.text();
                }
                throw new Error('Failed to send heartbeat');
            })
            .then(html => {
                document.body.innerHTML = html;
            })
            .catch(error => {
                button.disabled = false;
                loading.style.display = 'none';
                alert('Error: ' + error.message);
            });
        });
    </script>
</body>
</html>
`
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
