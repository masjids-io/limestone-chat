package domain

import (
	"time"

	"github.com/google/uuid"
)

type Gender string

type User struct {
	ID             uuid.UUID `gorm:"type:char(36);primaryKey" json:"id"`
	Email          string    `gorm:"type:varchar(320);unique;not null"`
	Username       string    `gorm:"type:varchar(255);unique;not null"`
	HashedPassword string    `gorm:"type:varchar(60);not null"`
	IsVerified     bool      `gorm:"default:false" json:"is_verified"`
	FirstName      string    `gorm:"type:varchar(255);not null" json:"first_name"`
	LastName       string    `gorm:"type:varchar(255);not null" json:"last_name"`
	PhoneNumber    string    `gorm:"type:varchar(255);not null" json:"phone_number"`
	Gender         string    `gorm:"type:varchar(255);not null" json:"gender"`
	CreatedAt      time.Time `gorm:"autoCreateTime"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime"`
}
