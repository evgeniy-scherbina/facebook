package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
)

var sumServiceURL, mulServiceURL string

func init() {
	sumServiceURL = os.Getenv("SUM_SERVICE_URL")
	if sumServiceURL == "" {
		sumServiceURL = "http://sum-service"
	}
	mulServiceURL = os.Getenv("MUL_SERVICE_URL")
	if mulServiceURL == "" {
		mulServiceURL = "http://mul-service"
	}
}

func calcHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	aStr := r.URL.Query().Get("a")
	bStr := r.URL.Query().Get("b")
	if aStr == "" || bStr == "" {
		http.Error(w, "query params 'a' and 'b' are required", http.StatusBadRequest)
		return
	}

	a, err := strconv.ParseFloat(aStr, 64)
	if err != nil {
		http.Error(w, "query param 'a' must be a number", http.StatusBadRequest)
		return
	}
	b, err := strconv.ParseFloat(bStr, 64)
	if err != nil {
		http.Error(w, "query param 'b' must be a number", http.StatusBadRequest)
		return
	}

	query := "?a=" + aStr + "&b=" + bStr

	var sumResult, mulResult map[string]interface{}
	var sumErr, mulErr error
	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()
		resp, err := http.Get(sumServiceURL + query)
		if err != nil {
			sumErr = err
			return
		}
		defer resp.Body.Close()
		sumErr = json.NewDecoder(resp.Body).Decode(&sumResult)
	}()
	go func() {
		defer wg.Done()
		resp, err := http.Get(mulServiceURL + query)
		if err != nil {
			mulErr = err
			return
		}
		defer resp.Body.Close()
		mulErr = json.NewDecoder(resp.Body).Decode(&mulResult)
	}()

	wg.Wait()

	if sumErr != nil {
		log.Printf("sum service error: %v", sumErr)
		http.Error(w, "sum service unavailable", http.StatusBadGateway)
		return
	}
	if mulErr != nil {
		log.Printf("mul service error: %v", mulErr)
		http.Error(w, "mul service unavailable", http.StatusBadGateway)
		return
	}

	result := map[string]interface{}{
		"a":   a,
		"b":   b,
		"sum": sumResult["sum"],
		"mul": mulResult["mul"],
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
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

	http.HandleFunc("/", calcHandler)
	http.HandleFunc("/health", healthHandler)

	log.Printf("Calc service listening on :%s (sum=%s mul=%s)", port, sumServiceURL, mulServiceURL)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
