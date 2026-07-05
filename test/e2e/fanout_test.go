//go:build e2e

// Package e2e holds end-to-end tests that talk to the REAL deployed cluster
// through the public ingress ELB (not in-process fakes).
//
// Run:
//   go test -tags e2e ./test/e2e/ -run TestFanOut -v
//
// Prerequisites:
//   - The chat app is deployed (kubectl apply -k k8s/) and reachable via the ELB below.
//   - To observe the fan-out BUG, scale the notification service to >1 replica:
//       kubectl scale deploy/real-time-ntfn-service -n chat-app --replicas=2
//     With 1 replica the test passes; with 2 replicas and no shared pub/sub it fails.
package e2e

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

// Hardcoded public ingress ELB for the cluster. Update if the ELB is recreated.
const baseURL = "http://a000aed69195e47dd85cfde7487c9291-139361245.us-east-1.elb.amazonaws.com"

// sseHTTPClient has NO timeout — SSE streams are long-lived. Per-read timeouts
// are enforced in waitForMessage instead.
var sseHTTPClient = &http.Client{
	Transport: &http.Transport{
		MaxIdleConns:        200,
		MaxIdleConnsPerHost: 200,
		MaxConnsPerHost:     0, // unlimited concurrent connections to the ELB
	},
}

type sseClient struct {
	resp   *http.Response
	reader *bufio.Reader
}

// connectSSE opens one SSE stream to /events through the ELB. When this returns,
// the server handler has already registered the client in its pod's hub (the
// response headers are flushed together with the initial ping, which is written
// only after registration).
func connectSSE(t *testing.T, base string) *sseClient {
	t.Helper()
	resp, err := sseHTTPClient.Get(base + "/events")
	if err != nil {
		t.Fatalf("connect SSE: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		t.Fatalf("connect SSE: unexpected status %d", resp.StatusCode)
	}
	c := &sseClient{resp: resp, reader: bufio.NewReader(resp.Body)}
	t.Cleanup(func() { _ = resp.Body.Close() })
	return c
}

// waitForMessage reads the stream until a `data:` line contains needle, or timeout.
func (c *sseClient) waitForMessage(needle string, timeout time.Duration) bool {
	found := make(chan bool, 1)
	go func() {
		for {
			line, err := c.reader.ReadString('\n')
			if err != nil {
				found <- false
				return
			}
			if strings.HasPrefix(line, "data:") && strings.Contains(line, needle) {
				found <- true
				return
			}
			// ignore comment (": ping"/keepalive) and unrelated data lines
		}
	}()

	select {
	case ok := <-found:
		return ok
	case <-time.After(timeout):
		_ = c.resp.Body.Close() // unblock the reader goroutine
		return false
	}
}

// postMessage sends one comment through the real message-service (/messages),
// which asynchronously fans it out to a real-time-ntfn replica via /notify.
func postMessage(t *testing.T, base, content string) {
	t.Helper()
	body, _ := json.Marshal(map[string]string{
		"content": content,
		"user":    "e2e",
	})
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(base+"/messages", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /messages: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("POST /messages: unexpected status %d", resp.StatusCode)
	}
}

// TestFanOut_AllViewersReceiveMessage opens many SSE viewers through the ELB,
// posts one message, and asserts EVERY viewer receives it.
//
//   - real-time-ntfn at 1 replica  -> all viewers share one pod's hub  -> PASS
//   - real-time-ntfn at 2+ replicas -> viewers are split across pods; /notify
//     reaches only one pod's hub    -> only that subset receives it      -> FAIL
//
// The failure is the article's fan-out problem, proven end-to-end against the
// real cluster. It will pass again once broadcasts are shared across replicas
// (e.g. Redis pub/sub).
func TestFanOut_AllViewersReceiveMessage(t *testing.T) {
	const numViewers = 20
	const readTimeout = 15 * time.Second

	nonce := fmt.Sprintf("e2e-fanout-%d", time.Now().UnixNano())

	// Open all viewer connections. With multiple replicas, the Service spreads
	// these across pods (20 connections => both pods near-certainly get some).
	viewers := make([]*sseClient, numViewers)
	for i := 0; i < numViewers; i++ {
		viewers[i] = connectSSE(t, baseURL)
	}

	// Let all registrations settle across replicas before publishing.
	time.Sleep(2 * time.Second)

	postMessage(t, baseURL, nonce)

	// Count how many viewers received the message.
	var wg sync.WaitGroup
	received := make([]bool, numViewers)
	for i := range viewers {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			received[i] = viewers[i].waitForMessage(nonce, readTimeout)
		}(i)
	}
	wg.Wait()

	got := 0
	for _, r := range received {
		if r {
			got++
		}
	}

	t.Logf("fan-out result: %d/%d viewers received the message", got, numViewers)
	if got != numViewers {
		t.Fatalf("FAN-OUT BUG: only %d/%d viewers received the message.\n"+
			"  real-time-ntfn is running with >1 replica and no shared pub/sub, so /notify\n"+
			"  reached only one pod's SSE clients. Fix with Redis pub/sub (or run 1 replica).",
			got, numViewers)
	}
}
