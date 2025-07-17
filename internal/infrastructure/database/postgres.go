package database

import (
	"fmt"
	"github.com/masjids-io/limestone-chat/internal/domain"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewPostgreSQLDB() (*gorm.DB, error) {
	dsn := os.Getenv("DATABASE_URL")

	if dsn == "" {
		log.Println("DEBUG: DATABASE_URL is empty! This is likely the root cause.")
		return nil, fmt.Errorf("DATABASE_URL environment variable is not set or empty")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying SQL DB: %w", err)
	}
	if err = sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Running GORM AutoMigrate...")
	err = db.AutoMigrate(&domain.Conversation{}, &domain.Message{}, &domain.MessageRead{}, &domain.IncomingChatMessage{}, &domain.ConversationParticipant{}) // Ganti dengan model Anda yang sebenarnya
	if err != nil {
		return nil, fmt.Errorf("failed to auto migrate database: %w", err)
	}

	log.Println("GORM AutoMigrate completed successfully!")

	return db, nil
}
