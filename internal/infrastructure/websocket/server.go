package websocket

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/masjids-io/limestone-chat/internal/application/services"
	"github.com/masjids-io/limestone-chat/internal/domain"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

type Hub struct {
	clients     map[uuid.UUID]map[*Client]bool
	broadcast   chan *domain.Message
	register    chan *Client
	unregister  chan *Client
	mu          sync.RWMutex
	chatService services.ChatService
	db          *gorm.DB
}

type Client struct {
	hub            *Hub
	conn           *websocket.Conn
	send           chan []byte
	userID         uuid.UUID
	conversationID uuid.UUID
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// TODO: config check origin (in production, restrict this to frontend domains)
		return true
	},
}

func NewHub(chatSvc services.ChatService, database *gorm.DB) *Hub {
	hub := &Hub{
		broadcast:   make(chan *domain.Message),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		clients:     make(map[uuid.UUID]map[*Client]bool),
		chatService: chatSvc,
		db:          database,
	}
	go hub.run()
	return hub
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if _, ok := h.clients[client.conversationID]; !ok {
				h.clients[client.conversationID] = make(map[*Client]bool)
			}
			h.clients[client.conversationID][client] = true
			h.mu.Unlock()
			log.Printf("Client %s connected to conversation %s. Total clients in this conversation: %d\n", client.userID.String(), client.conversationID.String(), len(h.clients[client.conversationID]))

		case client := <-h.unregister:
			h.mu.Lock()
			if clientsInConv, ok := h.clients[client.conversationID]; ok {
				if _, ok := clientsInConv[client]; ok {
					delete(clientsInConv, client)
					close(client.send)
					if len(clientsInConv) == 0 {
						delete(h.clients, client.conversationID)
					}
					log.Printf("Client %s disconnected from conversation %s. Remaining clients in this conversation: %d\n", client.userID.String(), client.conversationID.String(), len(h.clients[client.conversationID]))
				}
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			if clientsInConv, ok := h.clients[message.ConversationID]; ok {
				responseMessage := map[string]interface{}{
					"id":                  message.ID.String(),
					"conversation_id":     message.ConversationID.String(),
					"sender_id":           message.SenderID.String(),
					"content":             message.Content,
					"type":                message.MessageType,
					"media_url":           message.MediaURL.String,
					"metadata":            json.RawMessage(message.Metadata),
					"reply_to_message_id": message.ReplyToMessageID.String,
					"created_at":          message.CreatedAt.Format(time.RFC3339),
				}
				if !message.MediaURL.Valid {
					responseMessage["media_url"] = nil
				}
				if !message.ReplyToMessageID.Valid {
					responseMessage["reply_to_message_id"] = nil
				}

				responseBytes, err := json.Marshal(responseMessage)
				if err != nil {
					log.Printf("Error marshaling saved message for broadcast in Hub: %v\n", err)
					continue
				}

				for client := range clientsInConv {
					select {
					case client.send <- responseBytes:
					default:
						close(client.send)
						delete(clientsInConv, client)
						log.Printf("Client %s's send channel blocked, unregistering.", client.userID.String())
					}
				}
			} else {
				log.Printf("No active clients for conversation %s to broadcast message %s.\n", message.ConversationID.String(), message.ID.String())
			}
			h.mu.RUnlock()
		}
	}
}

func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request, userID uuid.UUID) {
	purposeStr := r.URL.Query().Get("purpose")
	if purposeStr == "" {
		http.Error(w, "Conversation purpose is required", http.StatusBadRequest)
		return
	}

	purpose := domain.ConversationPurpose(purposeStr)
	if !purpose.IsValid() {
		http.Error(w, "Invalid conversation purpose", http.StatusBadRequest)
		return
	}

	partnerIDStr := r.URL.Query().Get("partner_id")
	if partnerIDStr == "" {
		http.Error(w, "Partner ID is required for this conversation type", http.StatusBadRequest)
		return
	}

	partnerID, err := uuid.Parse(partnerIDStr)
	if err != nil {
		http.Error(w, "Invalid partner ID format", http.StatusBadRequest)
		return
	}

	if userID == partnerID {
		http.Error(w, "Cannot chat with yourself", http.StatusBadRequest)
		return
	}

	var conversationID uuid.UUID
	var existingConversation domain.Conversation
	err = hub.db.
		Joins("JOIN conversation_participants cp1 ON conversations.id = cp1.conversation_id").
		Joins("JOIN conversation_participants cp2 ON conversations.id = cp2.conversation_id").
		Where("cp1.user_id = ? AND cp2.user_id = ? AND conversations.purpose = ?", userID, partnerID, purpose).
		First(&existingConversation).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Printf("No existing conversation found for user %s and partner %s with purpose %s. Creating new one.\n", userID.String(), partnerID.String(), purpose)

			newConversation := domain.Conversation{
				ID:        uuid.New(),
				CreatorID: userID,
				Type:      domain.ConversationTypePrivate,
				Purpose:   purpose,
				Name:      sql.NullString{String: fmt.Sprintf("Chat for %s & %s - %s", userID.String()[:8], partnerID.String()[:8], purpose), Valid: true},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			tx := hub.db.Begin()
			if tx.Error != nil {
				log.Printf("Failed to begin transaction for new conversation: %v", tx.Error)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			if err := tx.Create(&newConversation).Error; err != nil {
				tx.Rollback()
				log.Printf("Failed to create new conversation: %v", err)
				http.Error(w, "Failed to create conversation", http.StatusInternalServerError)
				return
			}

			participant1 := domain.ConversationParticipant{
				ConversationID: newConversation.ID,
				UserID:         userID,
				JoinedAt:       time.Now(),
				Role:           "member",
			}
			if err := tx.Create(&participant1).Error; err != nil {
				tx.Rollback()
				log.Printf("Failed to add user %s as participant: %v", userID.String(), err)
				http.Error(w, "Failed to add participant", http.StatusInternalServerError)
				return
			}

			participant2 := domain.ConversationParticipant{
				ConversationID: newConversation.ID,
				UserID:         partnerID,
				JoinedAt:       time.Now(),
				Role:           "member",
			}
			if err := tx.Create(&participant2).Error; err != nil {
				tx.Rollback()
				log.Printf("Failed to add partner user %s as participant: %v", partnerID.String(), err)
				http.Error(w, "Failed to add partner participant", http.StatusInternalServerError)
				return
			}

			if err := tx.Commit().Error; err != nil {
				log.Printf("Failed to commit transaction: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			conversationID = newConversation.ID
			log.Printf("New conversation %s created successfully for users %s and %s with purpose %s.\n", conversationID.String(), userID.String(), partnerID.String(), purpose)

		} else {
			log.Printf("Error finding existing conversation: %v\n", err)
			http.Error(w, "Error finding conversation", http.StatusInternalServerError)
			return
		}
	} else {
		conversationID = existingConversation.ID
		log.Printf("Found existing conversation %s for users %s and %s with purpose %s.\n", conversationID.String(), userID.String(), partnerID.String(), purpose)

		var currentParticipant domain.ConversationParticipant
		err := hub.db.Where("conversation_id = ? AND user_id = ?", conversationID, userID).First(&currentParticipant).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				newParticipant := domain.ConversationParticipant{
					ConversationID: conversationID,
					UserID:         userID,
					JoinedAt:       time.Now(),
					Role:           "member",
				}
				if err := hub.db.Create(&newParticipant).Error; err != nil {
					log.Printf("Failed to add reconnecting user %s as participant to existing conversation %s: %v", userID.String(), conversationID.String(), err)
					http.Error(w, "Failed to add reconnecting participant", http.StatusInternalServerError)
					return
				}
				log.Printf("User %s re-added as participant to existing conversation %s.\n", userID.String(), conversationID.String())
			} else {
				log.Printf("Error checking participant status for user %s in conversation %s: %v", userID.String(), conversationID.String(), err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
		}

		var partnerParticipant domain.ConversationParticipant
		err = hub.db.Where("conversation_id = ? AND user_id = ?", conversationID, partnerID).First(&partnerParticipant).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				newPartnerParticipant := domain.ConversationParticipant{
					ConversationID: conversationID,
					UserID:         partnerID,
					JoinedAt:       time.Now(),
					Role:           "member",
				}
				if err := hub.db.Create(&newPartnerParticipant).Error; err != nil {
					log.Printf("Failed to add missing partner %s as participant to existing conversation %s: %v", partnerID.String(), conversationID.String(), err)
					http.Error(w, "Failed to add missing partner participant", http.StatusInternalServerError)
					return
				}
				log.Printf("Partner %s added as participant to existing conversation %s.\n", partnerID.String(), conversationID.String())
			} else {
				log.Printf("Error checking partner participant status for user %s in conversation %s: %v", partnerID.String(), conversationID.String(), err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
		}
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := &Client{
		hub:            hub,
		conn:           conn,
		send:           make(chan []byte, 256),
		userID:         userID,
		conversationID: conversationID,
	}
	client.hub.register <- client

	log.Printf("Incoming WebSocket connection from User ID: %s to Conversation ID: %s (Partner: %s)\n", client.userID.String(), client.conversationID.String(), partnerID.String())

	go client.writePump()
	go client.readPump()
}

type IncomingChatMessage struct {
	Type             string                 `json:"type"`
	Content          string                 `json:"content"`
	MediaURL         string                 `json:"media_url"`
	Metadata         map[string]interface{} `json:"metadata"`
	ReplyToMessageID *uuid.UUID             `json:"reply_to_message_id"`
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				log.Printf("Error getting next writer for client %s: %v\n", c.userID.String(), err)
				return
			}

			if _, err := w.Write(message); err != nil {
				log.Printf("Error writing message to client %s: %v\n", c.userID.String(), err)
				return
			}

			/*
			   n := len(c.send)
			   for i := 0; i < n; i++ {
			      if _, err := w.Write(<-c.send); err != nil {
			         log.Printf("Error writing queued message to client %s: %v\n", c.userID.String(), err)
			         return
			      }
			   }
			*/

			if err := w.Close(); err != nil {
				log.Printf("Error closing writer for client %s: %v\n", c.userID.String(), err)
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Error sending ping to client %s: %v\n", c.userID.String(), err)
				return
			}
		}
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		log.Printf("Received Pong from client %s, resetting read deadline.", c.userID.String())
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, messageBytes, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Error reading message from client %s: %v\n", c.userID.String(), err)
			}
			break
		}

		var incomingMsg IncomingChatMessage
		if err := json.Unmarshal(messageBytes, &incomingMsg); err != nil {
			log.Printf("Error unmarshaling incoming message from %s: %v, raw message: %s\n", c.userID.String(), err, string(messageBytes))
			c.send <- []byte(fmt.Sprintf(`{"error": "Invalid message format: %v"}`, err.Error()))
			continue
		}

		var metadataBytes []byte
		if incomingMsg.Metadata != nil {
			metadataBytes, err = json.Marshal(incomingMsg.Metadata)
			if err != nil {
				log.Printf("Error marshaling metadata for message from %s: %v\n", c.userID.String(), err)
				c.send <- []byte(`{"error": "Failed to process metadata"}`)
				continue
			}
		}

		mediaURL := incomingMsg.MediaURL
		var replyToMessageID *uuid.UUID
		if incomingMsg.ReplyToMessageID != nil {
			replyToMessageID = incomingMsg.ReplyToMessageID
		}

		savedMessage, err := c.hub.chatService.SendMessage(
			c.userID,
			c.conversationID,
			incomingMsg.Content,
			incomingMsg.Type,
			mediaURL,
			metadataBytes,
			replyToMessageID,
		)

		if err != nil {
			log.Printf("Failed to save message from %s to conversation %s: %v\n", c.userID.String(), c.conversationID.String(), err)
			c.send <- []byte(fmt.Sprintf(`{"error": "Failed to send message: %v"}`, err.Error()))
			continue
		}

		log.Printf("Message saved successfully from %s to conversation %s (Msg ID: %s)\n", c.userID.String(), c.conversationID.String(), savedMessage.ID.String())

		c.hub.broadcast <- savedMessage
	}
}
