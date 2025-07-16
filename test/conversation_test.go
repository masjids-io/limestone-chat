package test

import (
	"github.com/masjids-io/limestone-chat/internal/domain"
	"testing"
)

func TestConversationPurpose_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		purpose  domain.ConversationPurpose
		expected bool
	}{
		{
			name:     "Valid Purpose: Nikkah",
			purpose:  domain.ConversationPurposeNikkah,
			expected: true,
		},
		{
			name:     "Valid Purpose: Revert Service",
			purpose:  domain.ConversationPurposeRevertService,
			expected: true,
		},
		{
			name:     "Valid Purpose: General Support",
			purpose:  domain.ConversationPurposeGeneralSupport,
			expected: true,
		},
		{
			name:     "Valid Purpose: Admin Support",
			purpose:  domain.ConversationPurposeAdminSupport,
			expected: true,
		},
		{
			name:     "Invalid Purpose: Unknown Value",
			purpose:  domain.ConversationPurpose("unknown"),
			expected: false,
		},
		{
			name:     "Invalid Purpose: Empty String",
			purpose:  domain.ConversationPurpose(""),
			expected: false,
		},
		{
			name:     "Invalid Purpose: Other Valid String",
			purpose:  domain.ConversationPurpose("some_other_valid_string"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.purpose.IsValid()
			if got != tt.expected {
				t.Errorf("IsValid() for purpose %q got = %v, want %v", tt.purpose, got, tt.expected)
			}
		})
	}
}

// Anda juga perlu memastikan definisi ConversationPurpose ada.
// Sebagai contoh, jika Anda memiliki definisi di internal/domain/conversation.go:
/*
package domain

type ConversationPurpose string

const (
	ConversationPurposeNikkah        ConversationPurpose = "NIKKAH"
	ConversationPurposeRevertService ConversationPurpose = "REVERT_SERVICE"
	ConversationPurposeGeneralSupport ConversationPurpose = "GENERAL_SUPPORT"
	ConversationPurposeAdminSupport   ConversationPurpose = "ADMIN_SUPPORT"
)

func (cp ConversationPurpose) IsValid() bool {
    switch cp {
    case ConversationPurposeNikkah, ConversationPurposeRevertService,
       ConversationPurposeGeneralSupport, ConversationPurposeAdminSupport:
       return true
    }
    return false
}
*/
