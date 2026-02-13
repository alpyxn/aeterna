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

	var err error
	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to SQLite database at ", dbPath, ": ", err)
	}

	// Enable foreign keys for SQLite
	if err := DB.Exec("PRAGMA foreign_keys = ON;").Error; err != nil {
		log.Fatal("Failed to enable foreign keys: ", err)
	}

	// Enable WAL mode for better concurrent access
	if err := DB.Exec("PRAGMA journal_mode = WAL;").Error; err != nil {
		log.Println("Warning: Failed to enable WAL mode:", err)
	}

	log.Println("Database connection successfully opened:", dbPath)
}
