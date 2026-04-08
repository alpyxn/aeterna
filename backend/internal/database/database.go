package database

import (
	"log"
	"os"
	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect() {
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "./data/aeterna.db"
	}

	// Warn if PostgreSQL environment variables are set (Aeterna uses SQLite only)
	if os.Getenv("DB_HOST") != "" || os.Getenv("POSTGRES_HOST") != "" || os.Getenv("DATABASE_URL") != "" {
		log.Println("WARNING: PostgreSQL environment variables detected, but Aeterna uses SQLite only.")
		log.Println("Ignoring PostgreSQL configuration and using SQLite at:", dbPath)
	}

	// Create data directory if it doesn't exist
	dbDir := filepath.Dir(dbPath)
	if dbDir != "." && dbDir != "" {
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			log.Fatal("Failed to create database directory: ", err)
		}
	}

	// DSN query params apply to every new connection in the pool (unlike one-shot PRAGMAs).
	// busy_timeout: writers wait (ms) when another writer holds the lock — important for multi-user HTTP handlers.
	// journal_mode=WAL: readers do not block the writer; multiple readers + one writer at a time.
	// _foreign_keys: enforce FK constraints on each connection.
	dsn := dbPath + "?_busy_timeout=10000&_journal_mode=WAL&_foreign_keys=1"

	var err error
	DB, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to SQLite database at ", dbPath, ": ", err)
	}

	log.Println("Database connection successfully opened:", dbPath)
}
