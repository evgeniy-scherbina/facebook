// Command search is the query service for FB Post Search: it searches posts by
// keyword, sorted by recency or like count. Skeleton for now.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

type SearchResponse struct {
	Query   string `json:"query"`
	Sort    string `json:"sort"`
	Results []Post `json:"results"`
}

type Post struct {
	ID        string `json:"id"`
	Content   string `json:"content"`
	User      string `json:"user,omitempty"`
	Likes     int    `json:"likes"`
	CreatedAt string `json:"created_at"`
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	sort := r.URL.Query().Get("sort")
	if sort == "" {
		sort = "recency" // or "likes"
	}

	// TODO: look up the query terms in the inverted index and return the matching
	// posts from the chosen pre-sorted index (recency list or like-count zset).

	resp := SearchResponse{
		Query:   query,
		Sort:    sort,
		Results: []Post{}, // empty until the index is implemented
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
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

	http.HandleFunc("/search", searchHandler)
	http.HandleFunc("/health", healthHandler)

	log.Printf("search service starting on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
