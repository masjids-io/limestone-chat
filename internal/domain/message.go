package domain

import (
	"database/sql"
	"encoding/json"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type Message struct {
	ID               uuid.UUID       `gorm:"type:char(36);primaryKey" json:"id"`
	ConversationID   uuid.UUID       `gorm:"column:conversation_id;not null;type:char(36)" json:"conversation_id"`
	SenderID         uuid.UUID       `gorm:"column:sender_id;not null;type:char(36)" json:"sender_id"` // Ubah ke uuid.UUID
	Content          string          `gorm:"column:content" json:"content"`
	MessageType      string          `gorm:"column:message_type;type:varchar(50);not null" json:"message_type"`
	MediaURL         sql.NullString  `gorm:"column:media_url" json:"media_url"`
	Metadata         json.RawMessage `gorm:"column:metadata;type:jsonb" json:"metadata"`
	ReplyToMessageID sql.NullString  `gorm:"column:reply_to_message_id" json:"reply_to_message_id"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
	DeletedAt        gorm.DeletedAt  `gorm:"index" json:"-"`

	Conversation   Conversation `gorm:"foreignKey:ConversationID;references:ID"`
	Sender         User         `gorm:"foreignKey:SenderID;references:ID"`
	ReplyToMessage *Message     `gorm:"foreignKey:ReplyToMessageID;references:ID"`

	MessageReads []MessageRead `gorm:"foreignKey:MessageID" json:"-"`
}

type MessageRead struct {
	MessageID uuid.UUID `gorm:"column:message_id;primaryKey;type:char(36)" json:"message_id"`
	ReaderID  uuid.UUID `gorm:"column:reader_id;primaryKey;type:char(36)" json:"reader_id"`
	ReadAt    time.Time `gorm:"column:read_at;not null" json:"read_at"`

	Message Message `gorm:"foreignKey:MessageID;references:ID"`
	Reader  User    `gorm:"foreignKey:ReaderID;references:ID"`
}

type IncomingChatMessage struct {
	Type             string          `json:"type"`
	Content          string          `json:"content"`
	MediaURL         string          `json:"media_url"`
	Metadata         json.RawMessage `gorm:"column:metadata;type:jsonb" json:"metadata"`
	ReplyToMessageID *uuid.UUID      `json:"reply_to_message_id"`
}
