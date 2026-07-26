// Command post is the ingestion service for FB Post Search: it accepts new posts
// (and later likes) and will index them for search. Skeleton for now.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
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

func createPostHandler(w http.ResponseWriter, r *http.Request) {
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

	post := Post{
		ID:        time.Now().Format("20060102150405.000000"),
		Content:   req.Content,
		User:      req.User,
		CreatedAt: time.Now(),
	}

	// TODO: persist the post and add it to the search index (inverted index in
	// Redis: term -> posting lists, plus recency + like-count sorted indexes).

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(post)
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

	http.HandleFunc("/posts", createPostHandler)
	http.HandleFunc("/health", healthHandler)

	log.Printf("post service starting on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
