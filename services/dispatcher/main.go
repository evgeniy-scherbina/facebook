// Command dispatcher routes /events SSE connections to RTN pods by consistent
// hashing on the room, replacing nginx's upstream-hash-by. It watches the RTN
// EndpointSlices to keep its ring in sync, and reverse-proxies each connection
// to the pod that owns its room.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/evgeniy-scherbina/k8s/internal/discovery"
	"github.com/evgeniy-scherbina/k8s/internal/hashring"
	"github.com/evgeniy-scherbina/k8s/internal/proxy"
)

func main() {
	port := getenv("PORT", "8080")
	namespace := getenv("NAMESPACE", "chat-app")
	rtnService := getenv("RTN_SERVICE", "real-time-ntfn-service")

	ring := hashring.New(0)

	// Cancel on SIGTERM/SIGINT so both discovery and the server shut down.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Discovery keeps the ring in sync with the live RTN pods (watches
	// EndpointSlices). Fatal on error: without discovery the dispatcher is
	// useless, so let the pod crashloop/restart.
	go func() {
		if err := discovery.Watch(ctx, ring, namespace, rtnService); err != nil {
			log.Fatalf("discovery: %v", err)
		}
	}()

	mux := http.NewServeMux()
	mux.Handle("/events", proxy.New(ring))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	srv := &http.Server{Addr: ":" + port, Handler: mux}

	// Graceful shutdown: on signal, stop accepting and drain briefly.
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	log.Printf("dispatcher listening on :%s (namespace=%s, rtn=%s)", port, namespace, rtnService)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server: %v", err)
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
