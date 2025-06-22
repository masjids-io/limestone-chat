package services

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/masjids-io/limestone-chat/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ChatService interface {
	SendMessage(senderID uuid.UUID, conversationID uuid.UUID, content string, messageType string, mediaURL string, metadata []byte, replyToMessageID *uuid.UUID) (*domain.Message, error)
	GetMessagesByConversation(conversationID uuid.UUID, limit, offset int) ([]domain.Message, error)
	MarkMessageAsRead(messageID uuid.UUID, readerID uuid.UUID) error
}

type chatService struct {
	db *gorm.DB
}

func NewChatService(db *gorm.DB) ChatService {
	return &chatService{db: db}
}

func (s *chatService) SendMessage(senderID uuid.UUID, conversationID uuid.UUID, content string, messageType string, mediaURL string, metadata []byte, replyToMessageID *uuid.UUID) (*domain.Message, error) {
	var conversation domain.Conversation
	if err := s.db.First(&conversation, "id = ?", conversationID).Error; err != nil {
		return nil, fmt.Errorf("conversation not found: %w", err)
	}

	newMessage := domain.Message{
		ID:             uuid.New(),
		ConversationID: conversationID,
		SenderID:       senderID,
		Content:        content,
		MessageType:    messageType,
		CreatedAt:      now(),
		UpdatedAt:      now(),
	}

	if mediaURL != "" {
		newMessage.MediaURL.String = mediaURL
		newMessage.MediaURL.Valid = true
	}
	if metadata != nil {
		newMessage.Metadata = metadata
	}
	if replyToMessageID != nil {
		newMessage.ReplyToMessageID.String = replyToMessageID.String()
		newMessage.ReplyToMessageID.Valid = true
	}

	if err := s.db.Create(&newMessage).Error; err != nil {
		return nil, fmt.Errorf("failed to save message: %w", err)
	}

	s.db.Model(&conversation).Update("last_message_id", newMessage.ID.String())

	log.Printf("Message sent: %v\n", newMessage.ID)
	return &newMessage, nil
}

func (s *chatService) GetMessagesByConversation(conversationID uuid.UUID, limit, offset int) ([]domain.Message, error) {
	var messages []domain.Message
	if err := s.db.Where("conversation_id = ?", conversationID).Limit(limit).Offset(offset).Order("created_at ASC").Find(&messages).Error; err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	return messages, nil
}

func (s *chatService) MarkMessageAsRead(messageID uuid.UUID, readerID uuid.UUID) error {
	var message domain.Message
	if err := s.db.First(&message, "id = ?", messageID).Error; err != nil {
		return fmt.Errorf("message not found: %w", err)
	}

	var participant domain.ConversationParticipant
	err := s.db.Where("conversation_id = ? AND user_id = ?", message.ConversationID, readerID).First(&participant).Error
	if err != nil {
		return fmt.Errorf("reader %s is not a participant of conversation %s: %w", readerID, message.ConversationID, err)
	}

	messageRead := domain.MessageRead{
		MessageID: messageID,
		ReaderID:  readerID,
		ReadAt:    now(),
	}
	if err := s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "message_id"}, {Name: "reader_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{"read_at": now()}),
	}).Create(&messageRead).Error; err != nil {
		return fmt.Errorf("failed to mark message as read: %w", err)
	}

	participant.LastReadMessageID.String = messageID.String()
	participant.LastReadMessageID.Valid = true
	if err := s.db.Save(&participant).Error; err != nil {
		log.Printf("Warning: Failed to update last read message ID for participant %s: %v", readerID, err)
	}

	return nil
}

func now() time.Time {
	return time.Now()
}
