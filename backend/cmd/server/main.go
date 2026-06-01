package main

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/alpyxn/aeterna/backend/internal/config"
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
	cfg := config.Load()

	logging.Init(cfg)

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

	sqliteEnc := database.SQLiteEncryptionConfig{
		Enabled:     cfg.Database.EncryptionEnabled,
		AutoMigrate: cfg.Database.EncryptionAutoMigrate,
	}

	if sqliteEnc.Enabled {
		sqlitePassphrase, err := services.PrepareSQLiteEncryptionPassphrase(cfg.Database.EncryptionKDFContextFile)
		if err != nil {
			log.Fatal("Failed to prepare SQLite encryption key material: ", err)
		}
		sqliteEnc.Passphrase = sqlitePassphrase
	} else if _, statErr := os.Stat(cfg.Database.EncryptionKDFContextFile); statErr == nil {
		// If a context file exists, derive passphrase so plain-mode auto-migrate can decrypt legacy encrypted DBs.
		sqlitePassphrase, err := services.PrepareSQLiteEncryptionPassphrase(cfg.Database.EncryptionKDFContextFile)
		if err != nil {
			log.Fatal("Failed to derive SQLite passphrase from existing context: ", err)
		}
		sqliteEnc.Passphrase = sqlitePassphrase
	}

	database.Connect(cfg, sqliteEnc)

	if err := database.RunPreAutoMigrate(database.DB, cfg); err != nil {
		log.Fatal("Failed to run pre-auto migrations: ", err)
	}

	if err := database.DB.AutoMigrate(
		&models.User{},
		&models.RefreshSession{},
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

	if err := database.RunMigrations(database.DB, cfg); err != nil {
		log.Fatal("Failed to run startup migrations: ", err)
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
	database.DB.Exec("UPDATE farewell_letters SET encrypted_content_raw = encrypted_content WHERE encrypted_content_raw IS NULL OR encrypted_content_raw = '';")
	database.DB.Exec("UPDATE farewell_letters SET encrypted_rendered_html = '' WHERE encrypted_rendered_html IS NULL;")
	database.DB.Exec("UPDATE farewell_letters SET derivatives_pending = 1 WHERE derivatives_pending IS NULL;")

	if err := services.EnsureUploadsDir(cfg.Database.Path); err != nil {
		log.Fatal("Failed to create uploads directory: ", err)
	}

	// --- Composition root: wire services ---
	authSvc := services.NewAuthService(cfg)
	messageSvc := services.MessageService{}
	fileSvc := services.NewFileService(cfg)
	farewellSvc := services.FarewellService{}
	settingsSvc := services.NewSettingsService(cfg)
	appSettingsSvc := services.ApplicationSettingsService{}
	webhookStore := services.NewWebhookStore(cfg)
	userAdminSvc := services.NewUserAdminService(cfg)
	farewellDerivationSvc := services.NewFarewellDerivationService()
	eventStreamSvc := services.NewEventStreamService()

	// Decorate mutating services with event emission.
	messageSvcWithEvents := services.NewNotifyingMessageService(messageSvc, eventStreamSvc)
	fileSvcWithEvents := services.NewNotifyingFileService(fileSvc, eventStreamSvc)
	farewellSvcWithEvents := services.NewNotifyingFarewellService(farewellSvc, eventStreamSvc)
	settingsSvcWithEvents := services.NewNotifyingSettingsService(settingsSvc, eventStreamSvc)
	webhookStoreWithEvents := services.NewNotifyingWebhookStore(webhookStore, eventStreamSvc)

	// --- Wire handlers ---
	authH := handlers.NewAuthHandlers(authSvc, cfg)
	messageH := handlers.NewMessageHandlers(messageSvcWithEvents)
	heartbeatH := handlers.NewHeartbeatHandlers(messageSvcWithEvents, settingsSvc)
	attachH := handlers.NewAttachmentHandlers(fileSvcWithEvents)
	settingsH := handlers.NewSettingsHandlers(settingsSvcWithEvents, appSettingsSvc)
	webhookH := handlers.NewWebhookHandlers(webhookStoreWithEvents)
	farewellH := handlers.NewFarewellHandlers(farewellSvcWithEvents, fileSvcWithEvents)
	usersH := handlers.NewUserHandlers(userAdminSvc)
	eventsH := handlers.NewEventsHandlers(eventStreamSvc)

	// --- Wire worker ---
	w := worker.New(settingsSvc, webhookStore, fileSvc, farewellDerivationSvc, cfg)

	app := fiber.New(fiber.Config{
		BodyLimit: 25 * 1024 * 1024,
	})

	app.Use(handlers.AttachRuntimeFlags(cfg.IsProduction()))
	app.Use(requestid.New())
	app.Use(logger.New(logger.Config{
		Format: "{\"time\":\"${time}\",\"ip\":\"${ip}\",\"status\":${status},\"method\":\"${method}\",\"path\":\"${path}\",\"latency\":\"${latency}\",\"req_id\":\"${locals:requestid}\"}\n",
	}))
	app.Use(middleware.SecurityHeaders(cfg))

	allowedOrigins := cfg.AllowedOriginsOrDefault()

	if allowedOrigins == "*" {
		app.Use(cors.New(cors.Config{
			AllowOriginsFunc: func(origin string) bool { return true },
			AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
			AllowMethods:     "GET, POST, PUT, DELETE, OPTIONS",
			AllowCredentials: true,
		}))
	} else {
		app.Use(cors.New(cors.Config{
			AllowOrigins:     allowedOrigins,
			AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
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
	apiV2 := app.Group("/api/v2")

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

	// Public routes (v2, token-oriented for mobile clients)
	apiV2.Get("/messages/:id", messageH.GetPublic)
	apiV2.Get("/setup/status", authH.SetupStatus)
	apiV2.Post("/setup", authH.SetupMasterPasswordV2)
	apiV2.Post("/auth/register", middleware.AuthRateLimiter, authH.RegisterV2)
	apiV2.Post("/auth/login", middleware.AuthRateLimiter, authH.LoginV2)
	apiV2.Post("/auth/reset-password", middleware.AuthRateLimiter, authH.ResetMasterPasswordV2)
	apiV2.Get("/auth/session", authH.SessionStatusV2)
	apiV2.Post("/auth/refresh", middleware.AuthRateLimiter, authH.RefreshV2)
	apiV2.Post("/auth/logout", authH.LogoutV2)

	// Protected routes
	mgmt := api.Group("/", middleware.MasterAuth(authSvc, cfg))
	registerProtectedRoutes(mgmt, messageH, attachH, farewellH, webhookH, settingsH, heartbeatH, usersH, eventsH)

	// Protected routes (v2, accepts Authorization: Bearer <token>)
	mgmtV2 := apiV2.Group("/", middleware.MasterAuthV2(authSvc, cfg))
	registerProtectedRoutes(mgmtV2, messageH, attachH, farewellH, webhookH, settingsH, heartbeatH, usersH, eventsH)

	go w.Start()

	log.Fatal(app.Listen(":3000"))
}

func registerProtectedRoutes(
	group fiber.Router,
	messageH *handlers.MessageHandlers,
	attachH *handlers.AttachmentHandlers,
	farewellH *handlers.FarewellHandlers,
	webhookH *handlers.WebhookHandlers,
	settingsH *handlers.SettingsHandlers,
	heartbeatH *handlers.HeartbeatHandlers,
	usersH *handlers.UserHandlers,
	eventsH *handlers.EventsHandlers,
) {
	group.Post("/messages", messageH.Create)
	group.Get("/messages", messageH.List)
	group.Delete("/messages/:id", messageH.Delete)
	group.Put("/messages/:id", messageH.Update)
	group.Post("/heartbeat", messageH.Heartbeat)

	group.Post("/messages/:id/attachments", attachH.Upload)
	group.Get("/messages/:id/attachments", attachH.List)
	group.Delete("/messages/:id/attachments/:attachmentId", attachH.Delete)

	group.Get("/messages/:id/farewell-letters", farewellH.List)
	group.Post("/messages/:id/farewell-letters", farewellH.Create)
	group.Put("/messages/:id/farewell-letters/:letterId", farewellH.Update)
	group.Delete("/messages/:id/farewell-letters/:letterId", farewellH.Delete)
	group.Post("/messages/:id/farewell-letters/cancel-pending", farewellH.CancelAllPending)
	group.Post("/messages/:id/farewell-letters/:letterId/cancel", farewellH.CancelPending)
	group.Post("/messages/:id/farewell-letters/:letterId/attachments", farewellH.UploadAttachment)
	group.Get("/messages/:id/farewell-letters/:letterId/attachments", farewellH.ListAttachments)
	group.Delete("/messages/:id/farewell-letters/:letterId/attachments/:attachmentId", farewellH.DeleteAttachment)

	group.Get("/webhooks", webhookH.List)
	group.Post("/webhooks", webhookH.Create)
	group.Put("/webhooks/:id", webhookH.Update)
	group.Delete("/webhooks/:id", webhookH.Delete)

	group.Get("/settings", settingsH.Get)
	group.Post("/settings", settingsH.Save)
	group.Post("/settings/test", settingsH.TestSMTP)
	group.Get("/heartbeat-token", heartbeatH.GetToken)

	group.Get("/users", usersH.List)
	group.Delete("/users/:id", usersH.Delete)
	group.Get("/events", eventsH.Stream)
}
