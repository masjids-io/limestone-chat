package database

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewPostgreSQLDB() (*gorm.DB, error) {
	dsn := os.Getenv("DATABASE_URL")

	// --- ADD THESE LOGGING LINES ---
	log.Printf("DEBUG: Value of DATABASE_URL from environment: %s", dsn)
	if dsn == "" {
		log.Println("DEBUG: DATABASE_URL is empty! This is likely the root cause.")
		return nil, fmt.Errorf("DATABASE_URL environment variable is not set or empty")
	}
	// --- END DEBUG LOGGING ---

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Println("Database connection and migration successful!")
	return db, nil
}
