package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/redis/go-redis/v9"
)

type MessageRequest struct {
	Content string `json:"content"`
	User    string `json:"user,omitempty"`
	Room    string `json:"room,omitempty"`
}

type MessageResponse struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	User      string    `json:"user,omitempty"`
	Room      string    `json:"room"`
	Timestamp time.Time `json:"timestamp"`
	Status    string    `json:"status"`
}

// channelPrefix + roomID is the Redis Pub/Sub channel for a room.
const channelPrefix = "comments:"

// defaultRoom is used when a request omits the room.
const defaultRoom = "general"

var rdb *redis.Client

func init() {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "redis:6379"
	}
	rdb = redis.NewClient(&redis.Options{Addr: redisAddr})
	log.Printf("Redis publisher configured for %s (channel prefix %q)", redisAddr, channelPrefix)
}

func sendMessageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req MessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		http.Error(w, "Content is required", http.StatusBadRequest)
		return
	}

	room := req.Room
	if room == "" {
		room = defaultRoom
	}

	// Generate a simple ID (in production, use UUID)
	messageID := time.Now().Format("20060102150405.000000")

	// Create message response
	response := MessageResponse{
		ID:        messageID,
		Content:   req.Content,
		User:      req.User,
		Room:      room,
		Timestamp: time.Now(),
		Status:    "sent",
	}

	// Publish the notification so every real-time-ntfn replica can fan it out
	go publishNotification(response)

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func publishNotification(message MessageResponse) {
	// Payload matches the Notification shape the real-time-ntfn subscribers decode.
	notificationData := map[string]interface{}{
		"id":        message.ID,
		"content":   message.Content,
		"user":      message.User,
		"room":      message.Room,
		"timestamp": message.Timestamp.Format(time.RFC3339),
		"type":      "message",
	}

	jsonData, err := json.Marshal(notificationData)
	if err != nil {
		log.Printf("Error marshaling notification: %v", err)
		return
	}

	// Publish to the room's channel; only replicas with viewers in that room are
	// subscribed to it, and they fan it out to their own SSE clients.
	channel := channelPrefix + message.Room
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Publish(ctx, channel, jsonData).Err(); err != nil {
		log.Printf("Error publishing notification to %s: %v", channel, err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	
	// Try multiple possible paths for index.html
	// In development: two levels up from services/message/
	// In Docker: in the same directory as the binary
	possiblePaths := []string{
		filepath.Join("..", "..", "index.html"), // Development path
		"index.html",                             // Docker path
		"/root/index.html",                       // Absolute Docker path
	}
	
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			http.ServeFile(w, r, path)
			return
		}
	}
	
	http.Error(w, "index.html not found", http.StatusNotFound)
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/messages", sendMessageHandler)
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/", indexHandler)

	log.Printf("Message service starting on :%s", port)
	log.Printf("UI available at http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

