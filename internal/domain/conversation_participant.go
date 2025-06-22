package domain

import (
	"database/sql"
	"github.com/google/uuid"
	"time"
)

type ConversationParticipant struct {
	ConversationID uuid.UUID    `gorm:"column:conversation_id;primaryKey;type:char(36)" json:"conversation_id"`
	UserID         uuid.UUID    `gorm:"column:user_id;primaryKey;type:char(36)" json:"user_id"`
	JoinedAt       time.Time    `gorm:"column:joined_at;not null" json:"joined_at"`
	LeftAt         sql.NullTime `gorm:"column:left_at" json:"left_at"`
	Role           string       `gorm:"column:role;type:varchar(50);default:'member'" json:"role"`

	LastReadMessageID sql.NullString `gorm:"column:last_read_message_id" json:"last_read_message_id"`
	LastReadMessage   *Message       `gorm:"foreignKey:LastReadMessageID;references:ID"`

	Conversation Conversation `gorm:"foreignKey:ConversationID;references:ID"`
	User         User         `gorm:"foreignKey:UserID;references:ID"`
}
