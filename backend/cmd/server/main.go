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
	encryptionKeyFile := flag.String("encryption-key-file", "", "Path to file containing encryption key (fallback, must have 0600 permissions)")
	flag.Parse()

	logging.Init()

	services.InitKeyManager(*encryptionKeyFile)

	cryptoSvc := services.CryptoService{}
	_, err := cryptoSvc.Encrypt("test")
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
		if os.Getenv("ALLOWED_ORIGINS") == "*" && os.Getenv("PROXY_MODE") != "simple" {
			log.Fatal("ALLOWED_ORIGINS cannot be '*' in production (unless using simple mode)")
		}
	}

	database.Connect()

	if err := database.DB.AutoMigrate(
		&models.User{},
		&models.Message{},
		&models.MessageReminder{},
		&models.Settings{},
		&models.Webhook{},
		&models.Attachment{},
		&models.ApplicationSettings{},
		&models.FarewellLetter{},
		&models.FarewellAttachment{},
	); err != nil {
		log.Fatal("Failed to migrate database: ", err)
	}

	if err := database.MigrateLegacyToMultitenant(database.DB); err != nil {
		log.Fatal("Failed to migrate to multi-tenant schema: ", err)
	}

	if err := services.EnsureApplicationSettingsRow(); err != nil {
		log.Fatal("Failed to ensure application settings: ", err)
	}

	database.DB.Exec("UPDATE messages SET key_fragment = 'local' WHERE key_fragment IS NULL OR key_fragment = '';")

	var messagesWithoutToken []models.Message
	database.DB.Where("management_token IS NULL OR management_token = ''").Find(&messagesWithoutToken)
	for i := range messagesWithoutToken {
		messagesWithoutToken[i].ManagementToken = uuid.NewString()
		database.DB.Save(&messagesWithoutToken[i])
	}

	database.DB.Exec("UPDATE messages SET encrypted_content = '' WHERE encrypted_content IS NULL;")
	database.DB.Exec("UPDATE settings SET webhook_enabled = 0 WHERE webhook_enabled IS NULL;")
	database.DB.Exec("UPDATE settings SET telegram_enabled = 0 WHERE telegram_enabled IS NULL;")

	if err := services.EnsureUploadsDir(); err != nil {
		log.Fatal("Failed to create uploads directory: ", err)
	}

	// --- Composition root: wire services ---
	authSvc := services.AuthService{}
	messageSvc := services.MessageService{}
	fileSvc := services.FileService{}
	farewellSvc := services.FarewellService{}
	settingsSvc := services.SettingsService{}
	appSettingsSvc := services.ApplicationSettingsService{}
	webhookStore := services.WebhookStore{}
	userAdminSvc := services.UserAdminService{}

	// --- Wire handlers ---
	authH := handlers.NewAuthHandlers(authSvc)
	messageH := handlers.NewMessageHandlers(messageSvc)
	heartbeatH := handlers.NewHeartbeatHandlers(messageSvc, settingsSvc)
	attachH := handlers.NewAttachmentHandlers(fileSvc)
	settingsH := handlers.NewSettingsHandlers(settingsSvc, appSettingsSvc)
	webhookH := handlers.NewWebhookHandlers(webhookStore)
	farewellH := handlers.NewFarewellHandlers(farewellSvc, fileSvc)
	usersH := handlers.NewUserHandlers(userAdminSvc)

	// --- Wire worker ---
	w := worker.New(settingsSvc, webhookStore, fileSvc)

	app := fiber.New(fiber.Config{
		BodyLimit: 25 * 1024 * 1024,
	})

	app.Use(requestid.New())
	app.Use(logger.New(logger.Config{
		Format: "{\"time\":\"${time}\",\"ip\":\"${ip}\",\"status\":${status},\"method\":\"${method}\",\"path\":\"${path}\",\"latency\":\"${latency}\",\"req_id\":\"${locals:requestid}\"}\n",
	}))
	app.Use(middleware.SecurityHeaders)

	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	if allowedOrigins == "" {
		allowedOrigins = "http://localhost:5173"
	}

	if allowedOrigins == "*" {
		app.Use(cors.New(cors.Config{
			AllowOriginsFunc: func(origin string) bool { return true },
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

	// Note: CSRF protection is provided by SameSite=Strict cookies.
	// SameSite=Strict is stronger than Lax and prevents CSRF for same-site origins.

	api := app.Group("/api")

	// Public routes
	api.Get("/messages/:id", messageH.GetPublic)
	api.Get("/setup/status", authH.SetupStatus)
	api.Post("/setup", authH.SetupMasterPassword)
	api.Post("/auth/register", middleware.AuthRateLimiter, authH.Register)
	api.Post("/auth/login", middleware.AuthRateLimiter, authH.Login)
	api.Post("/auth/verify", middleware.AuthRateLimiter, authH.VerifyMasterPassword)
	api.Post("/auth/reset-password", middleware.AuthRateLimiter, authH.ResetMasterPassword)
	api.Get("/auth/session", authH.SessionStatus)
	api.Post("/auth/logout", authH.Logout)
	api.Get("/quick-heartbeat/:token", heartbeatH.QuickHeartbeat)
	api.Post("/quick-heartbeat/:token", heartbeatH.QuickHeartbeat)

	// Protected routes
	mgmt := api.Group("/", middleware.MasterAuth)
	mgmt.Post("/messages", messageH.Create)
	mgmt.Get("/messages", messageH.List)
	mgmt.Delete("/messages/:id", messageH.Delete)
	mgmt.Put("/messages/:id", messageH.Update)
	mgmt.Post("/heartbeat", messageH.Heartbeat)
	mgmt.Post("/messages/:id/attachments", attachH.Upload)
	mgmt.Get("/messages/:id/attachments", attachH.List)
	mgmt.Delete("/messages/:id/attachments/:attachmentId", attachH.Delete)

	mgmt.Get("/messages/:id/farewell-letters", farewellH.List)
	mgmt.Post("/messages/:id/farewell-letters", farewellH.Create)
	mgmt.Put("/messages/:id/farewell-letters/:letterId", farewellH.Update)
	mgmt.Delete("/messages/:id/farewell-letters/:letterId", farewellH.Delete)
	mgmt.Post("/messages/:id/farewell-letters/:letterId/attachments", farewellH.UploadAttachment)
	mgmt.Get("/messages/:id/farewell-letters/:letterId/attachments", farewellH.ListAttachments)
	mgmt.Delete("/messages/:id/farewell-letters/:letterId/attachments/:attachmentId", farewellH.DeleteAttachment)

	mgmt.Get("/webhooks", webhookH.List)
	mgmt.Post("/webhooks", webhookH.Create)
	mgmt.Put("/webhooks/:id", webhookH.Update)
	mgmt.Delete("/webhooks/:id", webhookH.Delete)

	mgmt.Get("/settings", settingsH.Get)
	mgmt.Post("/settings", settingsH.Save)
	mgmt.Post("/settings/test", settingsH.TestSMTP)
	mgmt.Post("/settings/telegram/test", settingsH.TestTelegram)
	mgmt.Get("/heartbeat-token", heartbeatH.GetToken)

	mgmt.Get("/users", usersH.List)
	mgmt.Delete("/users/:id", usersH.Delete)

	go w.Start()

	log.Fatal(app.Listen(":3000"))
}
