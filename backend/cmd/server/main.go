package main

import (
	"log"
	"os"
	"time"

	"github.com/alpyxn/aeterna/backend/internal/database"
	"github.com/alpyxn/aeterna/backend/internal/handlers"
	"github.com/alpyxn/aeterna/backend/internal/logging"
	"github.com/alpyxn/aeterna/backend/internal/middleware"
	"github.com/alpyxn/aeterna/backend/internal/models"
	"github.com/alpyxn/aeterna/backend/internal/worker"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

func main() {
	logging.Init()
	if os.Getenv("ENV") == "production" {
		if os.Getenv("DATABASE_URL") == "" {
			log.Fatal("DATABASE_URL must be set in production")
		}
		if os.Getenv("ALLOWED_ORIGINS") == "" {
			log.Fatal("ALLOWED_ORIGINS must be set in production")
		}
		// Allow * only in simple mode (HTTP-only, IP-based access)
		if os.Getenv("ALLOWED_ORIGINS") == "*" && os.Getenv("PROXY_MODE") != "simple" {
			log.Fatal("ALLOWED_ORIGINS cannot be '*' in production (unless using simple mode)")
		}
	}
	// Initialize Database
	database.Connect()
	
	// Auto Migrate
	database.DB.AutoMigrate(&models.Message{}, &models.Settings{}, &models.Webhook{})

	// Ensure legacy schema constraints don't block inserts
	database.DB.Exec("ALTER TABLE messages ADD COLUMN IF NOT EXISTS key_fragment TEXT;")
	database.DB.Exec("ALTER TABLE messages ALTER COLUMN key_fragment SET DEFAULT 'local';")
	database.DB.Exec("UPDATE messages SET key_fragment = 'local' WHERE key_fragment IS NULL;")
	database.DB.Exec("ALTER TABLE messages ALTER COLUMN key_fragment SET NOT NULL;")

	database.DB.Exec("ALTER TABLE messages ADD COLUMN IF NOT EXISTS management_token UUID;")
	database.DB.Exec("ALTER TABLE messages ALTER COLUMN management_token SET DEFAULT gen_random_uuid();")
	database.DB.Exec("UPDATE messages SET management_token = gen_random_uuid() WHERE management_token IS NULL;")
	database.DB.Exec("ALTER TABLE messages ALTER COLUMN management_token SET NOT NULL;")

	// Legacy content column may still be NOT NULL in some schemas
	database.DB.Exec("ALTER TABLE messages ADD COLUMN IF NOT EXISTS content TEXT;")
	database.DB.Exec("ALTER TABLE messages ALTER COLUMN content SET DEFAULT ''; ")
	database.DB.Exec("UPDATE messages SET content = '' WHERE content IS NULL;")
	database.DB.Exec("ALTER TABLE messages ALTER COLUMN content SET NOT NULL;")

	// Ensure settings table has encryption key column
	database.DB.Exec("ALTER TABLE settings ADD COLUMN IF NOT EXISTS encryption_key TEXT;")
	// Webhook settings columns
	database.DB.Exec("ALTER TABLE settings ADD COLUMN IF NOT EXISTS webhook_url TEXT;")
	database.DB.Exec("ALTER TABLE settings ADD COLUMN IF NOT EXISTS webhook_secret TEXT;")
	database.DB.Exec("ALTER TABLE settings ADD COLUMN IF NOT EXISTS webhook_enabled BOOLEAN DEFAULT FALSE;")
	database.DB.Exec("UPDATE settings SET webhook_enabled = FALSE WHERE webhook_enabled IS NULL;")

	// Owner email and heartbeat token columns
	database.DB.Exec("ALTER TABLE settings ADD COLUMN IF NOT EXISTS owner_email TEXT;")
	database.DB.Exec("ALTER TABLE settings ADD COLUMN IF NOT EXISTS heartbeat_token TEXT;")

	// Reminder sent column for messages
	database.DB.Exec("ALTER TABLE messages ADD COLUMN IF NOT EXISTS reminder_sent BOOLEAN DEFAULT FALSE;")
	database.DB.Exec("UPDATE messages SET reminder_sent = FALSE WHERE reminder_sent IS NULL;")

	app := fiber.New(fiber.Config{
		BodyLimit: 1 * 1024 * 1024, // 1MB limit to prevent DoS
	})

	// Middleware
	app.Use(requestid.New())
	app.Use(logger.New(logger.Config{
		Format: "{\"time\":\"${time}\",\"ip\":\"${ip}\",\"status\":${status},\"method\":\"${method}\",\"path\":\"${path}\",\"latency\":\"${latency}\",\"req_id\":\"${locals:requestid}\"}\n",
	}))
	
	// Security headers middleware
	app.Use(middleware.SecurityHeaders)
	
	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	if allowedOrigins == "" {
		allowedOrigins = "http://localhost:5173"
	}
	
	// For simple mode (ALLOWED_ORIGINS=*), use dynamic origin to avoid Fiber CORS panic
	if allowedOrigins == "*" {
		app.Use(cors.New(cors.Config{
			AllowOriginsFunc: func(origin string) bool {
				return true // Allow all origins in simple mode
			},
			AllowHeaders:     "Origin, Content-Type, Accept",
			AllowMethods:     "GET, POST, PUT, DELETE, OPTIONS",
			AllowCredentials: true,
		}))
	} else {
		app.Use(cors.New(cors.Config{
			AllowOrigins:     allowedOrigins,
			AllowHeaders:     "Origin, Content-Type, Accept",
			AllowMethods:     "GET, POST, PUT, DELETE, OPTIONS",
			AllowCredentials: true,
		}))
	}
	app.Use(limiter.New(limiter.Config{
		Max:        120,
		Expiration: 1 * time.Minute,
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(429).JSON(fiber.Map{
				"error": "Too many requests",
				"code":  "rate_limited",
			})
		},
	}))

	// Note: CSRF Protection is provided by SameSite=Lax cookies
	// Additional CSRF token middleware removed as frontend doesn't support it
	// and SameSite provides sufficient protection for same-site origins

	// Routes
	api := app.Group("/api")
	
	// Public Reveal
	api.Get("/messages/:id", handlers.GetMessage)
	api.Get("/setup/status", handlers.SetupStatus)
	api.Post("/setup", handlers.SetupMasterPassword)
	
	// Auth endpoints with brute-force protection
	api.Post("/auth/verify", middleware.AuthRateLimiter, handlers.VerifyMasterPassword)
	api.Get("/auth/session", handlers.SessionStatus)
	api.Post("/auth/logout", handlers.Logout)

	// Quick heartbeat (no auth, token-based)
	api.Get("/quick-heartbeat/:token", handlers.QuickHeartbeat)

	// Protected Management
	mgmt := api.Group("/", middleware.MasterAuth)
	mgmt.Post("/messages", handlers.CreateMessage)
	mgmt.Get("/messages", handlers.ListMessages)
	mgmt.Delete("/messages/:id", handlers.DeleteMessage)
	mgmt.Put("/messages/:id", handlers.UpdateMessage)
	mgmt.Post("/heartbeat", handlers.Heartbeat)
	mgmt.Get("/webhooks", handlers.ListWebhooks)
	mgmt.Post("/webhooks", handlers.CreateWebhook)
	mgmt.Put("/webhooks/:id", handlers.UpdateWebhook)
	mgmt.Delete("/webhooks/:id", handlers.DeleteWebhook)
	
	// Settings
	mgmt.Get("/settings", handlers.GetSettings)
	mgmt.Post("/settings", handlers.SaveSettings)
	mgmt.Post("/settings/test", handlers.TestSMTP)
	mgmt.Get("/heartbeat-token", handlers.GetHeartbeatToken)

	// Start Background Worker
	go worker.Start()

	log.Fatal(app.Listen(":3000"))
}
