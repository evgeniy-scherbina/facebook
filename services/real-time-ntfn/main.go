package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

type Notification struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	User      string    `json:"user,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
}

type Client struct {
	id       string
	notify   chan Notification
	done     chan struct{}
	mu       sync.Mutex
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan Notification
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

func newHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan Notification),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("SSE client connected. Total clients: %d", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.notify)
				close(client.done)
			}
			h.mu.Unlock()
			log.Printf("SSE client disconnected. Total clients: %d", len(h.clients))

		case notification := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.notify <- notification:
				default:
					// Client's channel is full, skip
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (c *Client) serveSSE(w http.ResponseWriter) {
	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

	// Send initial connection message
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Send a ping to establish connection
	_, err := w.Write([]byte(": ping\n\n"))
	if err != nil {
		return
	}
	flusher.Flush()

	// Send notifications as they arrive
	for {
		select {
		case notification, ok := <-c.notify:
			if !ok {
				return
			}

			// Serialize notification to JSON
			jsonData, err := json.Marshal(notification)
			if err != nil {
				log.Printf("Error marshaling notification: %v", err)
				continue
			}

			// Write SSE formatted message
			_, err = w.Write([]byte("data: " + string(jsonData) + "\n\n"))
			if err != nil {
				return
			}
			flusher.Flush()

		case <-c.done:
			return

		case <-time.After(30 * time.Second):
			// Send keep-alive ping
			_, err := w.Write([]byte(": keepalive\n\n"))
			if err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func serveSSE(hub *Hub, w http.ResponseWriter, r *http.Request) {
	clientID := r.URL.Query().Get("client_id")
	if clientID == "" {
		clientID = time.Now().Format("20060102150405.000000")
	}

	client := &Client{
		id:     clientID,
		notify: make(chan Notification, 256),
		done:   make(chan struct{}),
	}

	hub.register <- client
	defer func() {
		hub.unregister <- client
	}()

	// Handle client disconnect
	ctx := r.Context()
	go func() {
		<-ctx.Done()
		close(client.done)
	}()

	client.serveSSE(w)
}

func notifyHandler(hub *Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var notification Notification
		if err := json.NewDecoder(r.Body).Decode(&notification); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Broadcast to all connected clients
		hub.broadcast <- notification

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Notification sent"))
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	hub := newHub()
	go hub.run()

	http.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		serveSSE(hub, w, r)
	})
	http.HandleFunc("/notify", notifyHandler(hub))
	http.HandleFunc("/health", healthHandler)

	log.Printf("Real-time notification service starting on :%s", port)
	log.Printf("SSE endpoint: http://localhost:%s/events", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

