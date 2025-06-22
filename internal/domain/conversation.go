package domain

import (
	"database/sql"
	"fmt"
	"github.com/google/uuid"
	"time"

	"gorm.io/gorm"
)

type ConversationType string

const (
	ConversationTypePrivate ConversationType = "private"
	ConversationTypeGroup   ConversationType = "group"
)

func (ct ConversationType) IsValid() bool {
	switch ct {
	case ConversationTypePrivate, ConversationTypeGroup:
		return true
	}
	return false
}

type ConversationPurpose string

const (
	ConversationPurposeNikkah         ConversationPurpose = "nikkah_service"
	ConversationPurposeRevertService  ConversationPurpose = "revert_service"
	ConversationPurposeGeneralSupport ConversationPurpose = "general_support"
	ConversationPurposeAdminSupport   ConversationPurpose = "admin_support"
)

func (cp ConversationPurpose) IsValid() bool {
	switch cp {
	case ConversationPurposeNikkah, ConversationPurposeRevertService,
		ConversationPurposeGeneralSupport, ConversationPurposeAdminSupport:
		return true
	}
	return false
}

type Conversation struct {
	ID            uuid.UUID                 `gorm:"primaryKey;type:char(36)" json:"id"`
	CreatorID     uuid.UUID                 `gorm:"column:creator_id;not null;type:char(36)" json:"creator_id"`
	Type          ConversationType          `gorm:"column:type;type:varchar(20);not null" json:"type"`
	Purpose       ConversationPurpose       `gorm:"column:purpose;type:varchar(50);not null" json:"purpose"`
	Name          sql.NullString            `gorm:"column:name" json:"name"`
	Description   sql.NullString            `gorm:"column:description" json:"description"`
	Creator       User                      `gorm:"foreignKey:CreatorID;references:ID"`
	LastMessageID sql.NullString            `gorm:"column:last_message_id" json:"last_message_id"`
	CreatedAt     time.Time                 `json:"created_at"`
	UpdatedAt     time.Time                 `json:"updated_at"`
	DeletedAt     gorm.DeletedAt            `gorm:"index" json:"-"`
	Participants  []ConversationParticipant `gorm:"foreignKey:ConversationID" json:"participants"`
	Messages      []Message                 `gorm:"foreignKey:ConversationID" json:"-"`
}

func (c *Conversation) BeforeSave(tx *gorm.DB) (err error) {
	if !c.Type.IsValid() {
		return fmt.Errorf("invalid conversation type: %s", c.Type)
	}
	if !c.Purpose.IsValid() {
		return fmt.Errorf("invalid conversation purpose: %s", c.Purpose)
	}
	return nil
}
