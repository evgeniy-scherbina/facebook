package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
)

func mulHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	a, err := strconv.ParseFloat(r.URL.Query().Get("a"), 64)
	if err != nil {
		http.Error(w, "query param 'a' must be a number", http.StatusBadRequest)
		return
	}
	b, err := strconv.ParseFloat(r.URL.Query().Get("b"), 64)
	if err != nil {
		http.Error(w, "query param 'b' must be a number", http.StatusBadRequest)
		return
	}

	resp := map[string]interface{}{
		"a":   a,
		"b":   b,
		"mul": a * b,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/", mulHandler)
	http.HandleFunc("/health", healthHandler)

	log.Printf("Mul service listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
