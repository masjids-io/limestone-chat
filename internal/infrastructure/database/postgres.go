package database

import (
	"fmt"
	"github.com/lpernett/godotenv"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewPostgreSQLDB() (*gorm.DB, error) {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	dsn := os.Getenv("DATABASE_URL")
	//if dsn == "" {
	//	dsn = "host=localhost user=user password=password dbname=limestone_chat port=5432 sslmode=disable TimeZone=Asia/Jakarta"
	//	log.Println("DATABASE_URL environment variable not set, using default DSN.")
	//}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Println("Database connection and migration successful!")
	return db, nil
}
