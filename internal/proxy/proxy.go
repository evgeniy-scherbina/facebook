// Package proxy is the dispatcher's HTTP handler: it consistent-hashes each
// /events?room= request to the owning RTN pod and reverse-proxies the SSE
// stream to it.
package proxy

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

const defaultRoom = "general"

// Getter is the subset of *hashring.Ring the handler needs.
type Getter interface {
	Get(key string) (member string, ok bool)
}

// Handler routes SSE connections to the RTN pod that owns their room.
type Handler struct {
	ring Getter
}

func New(ring Getter) *Handler {
	return &Handler{ring: ring}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	room := r.URL.Query().Get("room")
	if room == "" {
		room = defaultRoom
	}

	member, ok := h.ring.Get(room)
	if !ok {
		// Empty ring => no RTN pods known yet (discovery hasn't populated it).
		http.Error(w, "no upstream available", http.StatusServiceUnavailable)
		return
	}

	// Reverse-proxy to the chosen pod. NewSingleHostReverseProxy keeps the
	// incoming path and query (so ?room= reaches RTN, which subscribes to it).
	target := &url.URL{Scheme: "http", Host: member}
	rp := httputil.NewSingleHostReverseProxy(target)
	// FlushInterval < 0 => flush after every write, so SSE events stream through
	// immediately instead of being buffered until the (never-ending) response ends.
	rp.FlushInterval = -1
	rp.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, err error) {
		// If the chosen pod is gone/unreachable, the client's EventSource will
		// reconnect and re-hash over the (by then) updated ring.
		log.Printf("proxy: room %q -> %s failed: %v", room, member, err)
		http.Error(w, "upstream error", http.StatusBadGateway)
	}
	rp.ServeHTTP(w, r)
}
