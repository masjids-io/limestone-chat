package main

import (
	"context"
	"github.com/lpernett/godotenv"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/masjids-io/limestone-chat/internal/application/services"
	"github.com/masjids-io/limestone-chat/internal/infrastructure/database"
	"github.com/masjids-io/limestone-chat/internal/infrastructure/websocket"
	"github.com/masjids-io/limestone-chat/internal/interfaces/api"
)

func loadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

func main() {
	loadEnv()
	db, err := database.NewPostgreSQLDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	chatService := services.NewChatService(db)
	chatHub := websocket.NewHub(chatService, db)

	webSocketHandler := api.NewWebSocketHandler(chatService, chatHub)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", webSocketHandler.ServeChatWs)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Limestone Chat Service is running. Connect to /ws?purpose=<your_purpose>"))
	})

	server := &http.Server{
		Addr:         ":8082",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("Server starting on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited gracefully.")
}
