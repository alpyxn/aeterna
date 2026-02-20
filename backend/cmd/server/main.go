package main

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/alpyxn/aeterna/backend/internal/database"
	"github.com/alpyxn/aeterna/backend/internal/handlers"
	"github.com/alpyxn/aeterna/backend/internal/logging"
	"github.com/alpyxn/aeterna/backend/internal/middleware"
	"github.com/alpyxn/aeterna/backend/internal/models"
	"github.com/alpyxn/aeterna/backend/internal/services"
	"github.com/alpyxn/aeterna/backend/internal/worker"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/google/uuid"
)

func main() {
	// Parse CLI flags
	encryptionKeyFile := flag.String("encryption-key-file", "", "Path to file containing encryption key (fallback, must have 0600 permissions)")
	flag.Parse()

	// Initialize logging first
	logging.Init()

	// Initialize encryption key manager (CRITICAL - must happen before any encryption operations)
	services.InitKeyManager(*encryptionKeyFile)

	// Validate encryption key is available (fail fast if not)
	// The InitKeyManager already tries to load the key, so we just need to verify it worked
	// by attempting to use the crypto service
	cryptoService := services.CryptoService{}
	_, err := cryptoService.Encrypt("test")
	if err != nil {
		log.Fatalf("FATAL: Failed to initialize encryption key: %v\n\n"+
			"Please configure one of the following:\n"+
			"  1. Docker Secrets: mount key at /run/secrets/encryption_key\n"+
			"  2. Secure file: use --encryption-key-file flag (file must have 0600 permissions)\n"+
			"\n"+
			"For more information, see: https://github.com/alpyxn/aeterna/blob/main/README.md", err)
	}
	if os.Getenv("ENV") == "production" {
		if os.Getenv("DATABASE_PATH") == "" {
			log.Fatal("DATABASE_PATH must be set in production")
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

	// Auto Migrate - GORM handles schema creation and updates for SQLite
	// SQLite doesn't support ALTER TABLE ADD COLUMN IF NOT EXISTS, so we rely on AutoMigrate
	if err := database.DB.AutoMigrate(&models.Message{}, &models.Settings{}, &models.Webhook{}, &models.Attachment{}); err != nil {
		log.Fatal("Failed to migrate database: ", err)
	}

	// SQLite migration: Update existing records with default values if needed
	// These are safe operations that work with SQLite

	// Ensure key_fragment has default value for existing records
	database.DB.Exec("UPDATE messages SET key_fragment = 'local' WHERE key_fragment IS NULL OR key_fragment = '';")

	// Ensure management_token is set for existing records (BeforeCreate hook handles new ones)
	// For SQLite, we need to update in Go since SQLite doesn't have uuid generation
	var messagesWithoutToken []models.Message
	database.DB.Where("management_token IS NULL OR management_token = ''").Find(&messagesWithoutToken)
	for i := range messagesWithoutToken {
		messagesWithoutToken[i].ManagementToken = uuid.NewString()
		database.DB.Save(&messagesWithoutToken[i])
	}

	// Ensure encrypted_content is not null for existing records
	database.DB.Exec("UPDATE messages SET encrypted_content = '' WHERE encrypted_content IS NULL;")

	// Ensure webhook_enabled has default value
	database.DB.Exec("UPDATE settings SET webhook_enabled = 0 WHERE webhook_enabled IS NULL;")

	// Ensure reminder_sent has default value
	database.DB.Exec("UPDATE messages SET reminder_sent = 0 WHERE reminder_sent IS NULL;")

	// Create uploads directory
	if err := services.EnsureUploadsDir(); err != nil {
		log.Fatal("Failed to create uploads directory: ", err)
	}

	app := fiber.New(fiber.Config{
		BodyLimit: 12 * 1024 * 1024, // 12MB limit for file uploads
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
	// GET: Shows page with button, POST: Triggers heartbeat
	api.Get("/quick-heartbeat/:token", handlers.QuickHeartbeat)
	api.Post("/quick-heartbeat/:token", handlers.QuickHeartbeat)

	// Protected Management
	mgmt := api.Group("/", middleware.MasterAuth)
	mgmt.Post("/messages", handlers.CreateMessage)
	mgmt.Get("/messages", handlers.ListMessages)
	mgmt.Delete("/messages/:id", handlers.DeleteMessage)
	mgmt.Put("/messages/:id", handlers.UpdateMessage)
	mgmt.Post("/heartbeat", handlers.Heartbeat)
	mgmt.Post("/messages/:id/attachments", handlers.UploadAttachment)
	mgmt.Get("/messages/:id/attachments", handlers.ListAttachments)
	mgmt.Delete("/messages/:id/attachments/:attachmentId", handlers.DeleteAttachment)
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
