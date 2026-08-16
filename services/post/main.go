// Command post is the ingestion service for FB Post Search: it accepts new posts
// and indexes them for search.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/evgeniy-scherbina/k8s/internal/tokenize"
)

// Redis key layout
//
//	post:{id}              STRING  the post JSON. Stand-in for the primary post
//	                               store (source of truth); the real design keeps
//	                               posts in a database and only IDs in the index.
//	idx:recency:{term}     LIST    post IDs for that term, newest first (LPUSH).
//	                               The list IS the recency ordering — no sorting
//	                               at query time.
//	post:next_id           STRING  INCR counter used to mint post IDs.
const (
	postKeyPrefix  = "post:"
	recencyKeyFmt  = "idx:recency:%s"
	postIDCounter  = "post:next_id"
	redisOpTimeout = 5 * time.Second
)

type CreatePostRequest struct {
	Content string `json:"content"`
	User    string `json:"user,omitempty"`
}

type Post struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	User      string    `json:"user,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type server struct {
	rdb *redis.Client
}

func (s *server) createPostHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreatePostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Content == "" {
		http.Error(w, "content is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), redisOpTimeout)
	defer cancel()

	// INCR gives collision-free, monotonically increasing IDs — safe across
	// concurrent requests and multiple post-service replicas.
	n, err := s.rdb.Incr(ctx, postIDCounter).Result()
	if err != nil {
		log.Printf("minting post id: %v", err)
		http.Error(w, "storage unavailable", http.StatusServiceUnavailable)
		return
	}

	post := Post{
		ID:        fmt.Sprintf("%d", n),
		Content:   req.Content,
		User:      req.User,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.indexPost(ctx, post); err != nil {
		log.Printf("indexing post %s: %v", post.ID, err)
		http.Error(w, "failed to index post", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(post)
}

// indexPost stores the post and adds its ID to the recency list of every term it
// contains. All writes go in one pipeline: a single round trip instead of
// 1+N, which matters on a write-heavy path.
func (s *server) indexPost(ctx context.Context, post Post) error {
	body, err := json.Marshal(post)
	if err != nil {
		return fmt.Errorf("marshal post: %w", err)
	}

	terms := tokenize.Terms(post.Content)

	pipe := s.rdb.Pipeline()
	pipe.Set(ctx, postKeyPrefix+post.ID, body, 0)
	for _, term := range terms {
		// LPUSH => newest first, so a search can read the head of the list and
		// already have results in recency order.
		pipe.LPush(ctx, fmt.Sprintf(recencyKeyFmt, term), post.ID)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("redis pipeline: %w", err)
	}

	log.Printf("indexed post %s under %d term(s): %v", post.ID, len(terms), terms)
	return nil
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "redis:6379"
	}

	s := &server{rdb: redis.NewClient(&redis.Options{Addr: redisAddr})}

	http.HandleFunc("/posts", s.createPostHandler)
	http.HandleFunc("/health", healthHandler)

	log.Printf("post service starting on :%s (redis %s)", port, redisAddr)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
