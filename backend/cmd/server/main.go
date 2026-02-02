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
		if os.Getenv("ALLOWED_ORIGINS") == "*" {
			log.Fatal("ALLOWED_ORIGINS cannot be '*' in production")
		}
	}
	// Initialize Database
	database.Connect()
	
	// Auto Migrate
	database.DB.AutoMigrate(&models.Message{}, &models.Settings{})

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

	app := fiber.New()

	// Middleware
	app.Use(requestid.New())
	app.Use(logger.New(logger.Config{
		Format: "{\"time\":\"${time}\",\"ip\":\"${ip}\",\"status\":${status},\"method\":\"${method}\",\"path\":\"${path}\",\"latency\":\"${latency}\",\"req_id\":\"${locals:requestid}\"}\n",
	}))
	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	if allowedOrigins == "" {
		allowedOrigins = "http://localhost:5173"
	}
	app.Use(cors.New(cors.Config{
		AllowOrigins: allowedOrigins,
		AllowHeaders: "Origin, Content-Type, Accept, X-Master-Key",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))
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

	// Routes
	api := app.Group("/api")
	
	// Public Reveal
	api.Get("/messages/:id", handlers.GetMessage)
	api.Get("/setup/status", handlers.SetupStatus)
	api.Post("/setup", handlers.SetupMasterPassword)
	api.Post("/auth/verify", handlers.VerifyMasterPassword)

	// Protected Management
	mgmt := api.Group("/", middleware.MasterAuth)
	mgmt.Post("/messages", handlers.CreateMessage)
	mgmt.Get("/messages", handlers.ListMessages)
	mgmt.Delete("/messages/:id", handlers.DeleteMessage)
	mgmt.Post("/heartbeat", handlers.Heartbeat)
	
	// Settings
	mgmt.Get("/settings", handlers.GetSettings)
	mgmt.Post("/settings", handlers.SaveSettings)
	mgmt.Post("/settings/test", handlers.TestSMTP)

	// Start Background Worker
	go worker.Start()

	log.Fatal(app.Listen(":3000"))
}
