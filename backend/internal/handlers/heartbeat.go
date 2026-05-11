package handlers

import (
	"github.com/alpyxn/aeterna/backend/internal/ports"
	"github.com/alpyxn/aeterna/backend/internal/services"
	"github.com/gofiber/fiber/v2"
	"strings"
)

// HeartbeatHandlers groups quick-heartbeat and token route handlers.
type HeartbeatHandlers struct {
	messages ports.MessageServicePort
	settings ports.SettingsServicePort
}

func NewHeartbeatHandlers(messages ports.MessageServicePort, settings ports.SettingsServicePort) *HeartbeatHandlers {
	return &HeartbeatHandlers{messages: messages, settings: settings}
}

// QuickHeartbeat handles token-based heartbeat (no session auth required).
func (h *HeartbeatHandlers) QuickHeartbeat(c *fiber.Ctx) error {
	token := c.Params("token")
	if token == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Token required"})
	}

	settings, err := h.settings.GetByHeartbeatToken(token)
	if err != nil {
		return writeError(c, err)
	}

	userID := settings.UserID

	if c.Method() == "POST" {
		if err := h.messages.BulkHeartbeat(userID); err != nil {
			return writeError(c, services.Internal("Failed to update heartbeats", err))
		}

		html := `<!DOCTYPE html>
<html>
<head>
    <title>Heartbeat Confirmed - Aeterna</title>
    <meta charset="UTF-8">
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
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.SendString(html)
	}

	autoSend := c.Query("auto") == "1"
	autoScript := ""
	autoLoadingStyle := "display: none;"
	if autoSend {
		autoLoadingStyle = "display: block;"
		autoScript = `
        window.addEventListener('load', function() {
            submitHeartbeat();
        });
`
	}

	autoSubmitCall := `
        function submitHeartbeat() {
            const button = document.getElementById('heartbeatButton');
            const loading = document.getElementById('loading');
            const postURL = new URL(window.location.href);
            postURL.searchParams.delete('auto');
            
            button.disabled = true;
            loading.style.display = 'block';
            
            fetch(postURL.toString(), {
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
        }
`

	html := strings.NewReplacer(
		"__AUTO_LOADING_STYLE__", autoLoadingStyle,
		"__AUTO_SUBMIT_CALL__", autoSubmitCall,
		"__AUTO_SCRIPT__", autoScript,
	).Replace(`<!DOCTYPE html>
<html>
<head>
    <title>Send Heartbeat - Aeterna</title>
    <meta charset="UTF-8">
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
            __AUTO_LOADING_STYLE__
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
__AUTO_SUBMIT_CALL__
__AUTO_SCRIPT__
        document.getElementById('heartbeatForm').addEventListener('submit', function(e) {
            e.preventDefault();
            submitHeartbeat();
        });
    </script>
</body>
</html>
`)
	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(html)
}

// GetToken returns the quick-heartbeat token for the authenticated user.
func (h *HeartbeatHandlers) GetToken(c *fiber.Ctx) error {
	userID, err := currentUserID(c)
	if err != nil {
		return writeError(c, err)
	}
	settings, err := h.settings.Get(userID)
	if err != nil {
		return writeError(c, err)
	}

	return c.JSON(fiber.Map{
		"token": settings.HeartbeatToken,
	})
}
