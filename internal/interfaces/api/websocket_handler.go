package api

import (
	"log"
	"net/http"

	"github.com/masjids-io/limestone-chat/internal/application/services"
	"github.com/masjids-io/limestone-chat/internal/auth"
	"github.com/masjids-io/limestone-chat/internal/infrastructure/websocket"
)

type WebSocketHandler struct {
	chatService services.ChatService
	hub         *websocket.Hub
}

func NewWebSocketHandler(chatSvc services.ChatService, hub *websocket.Hub) *WebSocketHandler {
	return &WebSocketHandler{
		chatService: chatSvc,
		hub:         hub,
	}
}

func (h *WebSocketHandler) ServeChatWs(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.VerifyJWTForWebSocket(r)
	if err != nil {
		log.Printf("WebSocket authentication failed: %v", err)
		http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}

	log.Printf("Incoming WebSocket connection from authenticated User ID: %s\n", userID.String())
	websocket.ServeWs(h.hub, w, r, userID)
}
