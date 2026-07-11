package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// channelPrefix + roomID is the Redis Pub/Sub channel for a room.
const channelPrefix = "comments:"

type Notification struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	User      string    `json:"user,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
	Room      string    `json:"room,omitempty"`
}

type Client struct {
	id        string
	room      string
	notify    chan Notification
	done      chan struct{}
	closeOnce sync.Once
}

// Room holds the local SSE clients for a single room plus THIS replica's Redis
// subscription to that room's channel. A Room exists on a replica only while it
// has at least one local client — that is what drives subscribe/unsubscribe.
type Room struct {
	name    string
	clients map[*Client]bool
	pubsub  *redis.PubSub
}

type Hub struct {
	mu    sync.Mutex
	rooms map[string]*Room
	rdb   *redis.Client
}

func newHub(rdb *redis.Client) *Hub {
	return &Hub{
		rooms: make(map[string]*Room),
		rdb:   rdb,
	}
}

// addClient registers a client into its room. If this is the FIRST local client
// for that room, the replica subscribes to the room's Redis channel and starts
// consuming it — so a replica only listens on channels for rooms it serves.
func (h *Hub) addClient(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	room, ok := h.rooms[c.room]
	if !ok {
		pubsub := h.rdb.Subscribe(context.Background(), channelPrefix+c.room)
		room = &Room{
			name:    c.room,
			clients: make(map[*Client]bool),
			pubsub:  pubsub,
		}
		h.rooms[c.room] = room
		go h.consume(room)
		log.Printf("SUBSCRIBE %s%s  (rooms on this replica: %d)", channelPrefix, c.room, len(h.rooms))
	}
	room.clients[c] = true
	log.Printf("client joined room %q  (clients in room: %d)", c.room, len(room.clients))
}

// removeClient unregisters a client. If it was the LAST client in the room, the
// replica unsubscribes from the room's channel and drops the room.
func (h *Hub) removeClient(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	room, ok := h.rooms[c.room]
	if !ok {
		return
	}
	delete(room.clients, c)
	// Signal this client's SSE loop to stop. We deliberately do NOT close
	// c.notify, so a concurrent deliver() can never send on a closed channel.
	c.closeOnce.Do(func() { close(c.done) })

	if len(room.clients) == 0 {
		_ = room.pubsub.Close() // closes the channel => consume() goroutine exits
		delete(h.rooms, c.room)
		log.Printf("UNSUBSCRIBE %s%s  (rooms on this replica: %d)", channelPrefix, c.room, len(h.rooms))
	}
}

// consume reads messages from a room's Redis subscription and fans them out to
// that room's local SSE clients. It exits when the room's pubsub is closed.
func (h *Hub) consume(room *Room) {
	for msg := range room.pubsub.Channel() {
		var n Notification
		if err := json.Unmarshal([]byte(msg.Payload), &n); err != nil {
			log.Printf("bad notification on %s: %v", msg.Channel, err)
			continue
		}
		h.deliver(room.name, n)
	}
}

// deliver sends a notification to all local clients currently in the room.
func (h *Hub) deliver(roomName string, n Notification) {
	h.mu.Lock()
	room, ok := h.rooms[roomName]
	if !ok {
		h.mu.Unlock()
		return
	}
	targets := make([]*Client, 0, len(room.clients))
	for c := range room.clients {
		targets = append(targets, c)
	}
	h.mu.Unlock()

	for _, c := range targets {
		select {
		case c.notify <- n:
		default: // client's buffer is full, drop
		}
	}
}

func (c *Client) serveSSE(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Establish the stream.
	if _, err := w.Write([]byte(": ping\n\n")); err != nil {
		return
	}
	flusher.Flush()

	for {
		select {
		case notification := <-c.notify:
			jsonData, err := json.Marshal(notification)
			if err != nil {
				log.Printf("Error marshaling notification: %v", err)
				continue
			}
			if _, err := w.Write([]byte("data: " + string(jsonData) + "\n\n")); err != nil {
				return
			}
			flusher.Flush()

		case <-c.done:
			return

		case <-time.After(30 * time.Second):
			if _, err := w.Write([]byte(": keepalive\n\n")); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func serveSSE(hub *Hub, w http.ResponseWriter, r *http.Request) {
	room := r.URL.Query().Get("room")
	if room == "" {
		room = "general"
	}

	clientID := r.URL.Query().Get("client_id")
	if clientID == "" {
		clientID = time.Now().Format("20060102150405.000000")
	}

	client := &Client{
		id:     clientID,
		room:   room,
		notify: make(chan Notification, 256),
		done:   make(chan struct{}),
	}

	hub.addClient(client)
	defer hub.removeClient(client)

	// Stop the client when the HTTP connection closes.
	go func() {
		<-r.Context().Done()
		client.closeOnce.Do(func() { close(client.done) })
	}()

	client.serveSSE(w)
}

// debugRoomsHandler reports which rooms THIS replica is subscribed to and how
// many clients each has — used to observe co-location later (each pod should
// hold a small, disjoint set of rooms once the ingress hashes /events by room).
func debugRoomsHandler(hub *Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hub.mu.Lock()
		rooms := make(map[string]int, len(hub.rooms))
		for name, room := range hub.rooms {
			rooms[name] = len(room.clients)
		}
		hub.mu.Unlock()

		host, _ := os.Hostname()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"pod":   host,
			"rooms": rooms,
		})
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

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "redis:6379"
	}
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	hub := newHub(rdb)

	http.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		serveSSE(hub, w, r)
	})
	http.HandleFunc("/debug/rooms", debugRoomsHandler(hub))
	http.HandleFunc("/health", healthHandler)

	log.Printf("Real-time notification service starting on :%s", port)
	log.Printf("Redis %s, channel prefix %q", redisAddr, channelPrefix)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
