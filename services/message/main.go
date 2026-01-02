package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type MessageRequest struct {
	Content string `json:"content"`
	User    string `json:"user,omitempty"`
}

type MessageResponse struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	User      string    `json:"user,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Status    string    `json:"status"`
}

var notificationServiceURL string

func init() {
	notificationServiceURL = os.Getenv("NOTIFICATION_SERVICE_URL")
	if notificationServiceURL == "" {
		notificationServiceURL = "http://localhost:8081"
	}
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

	// Generate a simple ID (in production, use UUID)
	messageID := time.Now().Format("20060102150405.000000")

	// Create message response
	response := MessageResponse{
		ID:        messageID,
		Content:   req.Content,
		User:      req.User,
		Timestamp: time.Now(),
		Status:    "sent",
	}

	// Notify the real-time notification service
	go notifyRealTimeService(response)

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func notifyRealTimeService(message MessageResponse) {
	// Prepare notification payload
	notificationData := map[string]interface{}{
		"id":        message.ID,
		"content":   message.Content,
		"user":      message.User,
		"timestamp": message.Timestamp.Format(time.RFC3339),
		"type":      "message",
	}

	jsonData, err := json.Marshal(notificationData)
	if err != nil {
		log.Printf("Error marshaling notification: %v", err)
		return
	}

	// Send HTTP POST to notification service
	url := notificationServiceURL + "/notify"
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Error notifying real-time service: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Notification service returned status: %d", resp.StatusCode)
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
	log.Printf("Notification service URL: %s", notificationServiceURL)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

