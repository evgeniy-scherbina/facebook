//go:build e2e

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

// viewer is an SSE client for one room with a persistent background reader.
// Unlike the fan-out test's helper, it never closes the connection between
// checks, so a single viewer can be probed across multiple phases.
type viewer struct {
	room string
	resp *http.Response
	msgs chan string // payloads of received `data:` events
}

func connectViewer(t *testing.T, room string) *viewer {
	t.Helper()
	resp, err := sseHTTPClient.Get(baseURL + "/events?room=" + room)
	if err != nil {
		t.Fatalf("connect viewer (room %s): %v", room, err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		t.Fatalf("connect viewer (room %s): status %d", room, resp.StatusCode)
	}
	v := &viewer{room: room, resp: resp, msgs: make(chan string, 128)}
	t.Cleanup(func() { _ = resp.Body.Close() })
	go v.readLoop()
	return v
}

func (v *viewer) readLoop() {
	reader := bufio.NewReader(v.resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return // connection closed (test cleanup) -> stop
		}
		if strings.HasPrefix(line, "data:") {
			payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			select {
			case v.msgs <- payload:
			default: // buffer full, drop (shouldn't happen in this test)
			}
		}
	}
}

// received waits up to timeout for a message containing needle.
func (v *viewer) received(needle string, timeout time.Duration) bool {
	deadline := time.After(timeout)
	for {
		select {
		case m := <-v.msgs:
			if strings.Contains(m, needle) {
				return true
			}
			// not the message we're looking for; keep waiting
		case <-deadline:
			return false
		}
	}
}

// countReceived checks all viewers concurrently and returns how many received
// the needle within timeout. Concurrency matters for the negative checks: they
// always burn the full timeout, so doing them in parallel keeps the test fast.
func countReceived(viewers []*viewer, needle string, timeout time.Duration) int {
	var wg sync.WaitGroup
	var mu sync.Mutex
	count := 0
	for _, v := range viewers {
		wg.Add(1)
		go func(v *viewer) {
			defer wg.Done()
			if v.received(needle, timeout) {
				mu.Lock()
				count++
				mu.Unlock()
			}
		}(v)
	}
	wg.Wait()
	return count
}

func postComment(t *testing.T, room, content string) {
	t.Helper()
	body, _ := json.Marshal(map[string]string{
		"room":    room,
		"content": content,
		"user":    "e2e",
	})
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(baseURL+"/messages", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /messages (room %s): %v", room, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("POST /messages (room %s): status %d", room, resp.StatusCode)
	}
}

// TestRoomIsolation proves two rooms are independent: a message posted to room A
// reaches every A viewer and no B viewer, and vice-versa. Posting to BOTH rooms
// also confirms each room is actually live (so the "received nothing" checks
// aren't passing just because a room is dead).
//
// Run at 2 replicas to also exercise cross-replica fan-out within a room.
// RED until step 2 (per-room publish) ships and is deployed.
func TestRoomIsolation(t *testing.T) {
	const perRoom = 10
	const posTimeout = 10 * time.Second // positive: message MUST arrive
	const negTimeout = 3 * time.Second  // negative: bounded wait to confirm absence

	stamp := time.Now().UnixNano()
	roomA := fmt.Sprintf("A-%d", stamp)
	roomB := fmt.Sprintf("B-%d", stamp)

	aViewers := make([]*viewer, perRoom)
	bViewers := make([]*viewer, perRoom)
	for i := 0; i < perRoom; i++ {
		aViewers[i] = connectViewer(t, roomA)
		bViewers[i] = connectViewer(t, roomB)
	}

	// Let subscriptions settle across replicas before publishing.
	time.Sleep(2 * time.Second)

	// --- Phase 1: post to room A ---
	nonceA := fmt.Sprintf("msg-A-%d", stamp)
	postComment(t, roomA, nonceA)

	gotA := countReceived(aViewers, nonceA, posTimeout)
	leakA := countReceived(bViewers, nonceA, negTimeout)
	t.Logf("phase A: %d/%d room-A viewers got it; %d/%d room-B viewers leaked it", gotA, perRoom, leakA, perRoom)
	if gotA != perRoom {
		t.Errorf("room A delivery: %d/%d viewers received A's message (want all)", gotA, perRoom)
	}
	if leakA != 0 {
		t.Errorf("ISOLATION LEAK: %d room-B viewers received room A's message (want 0)", leakA)
	}

	// --- Phase 2: post to room B (proves B is live + isolation the other way) ---
	nonceB := fmt.Sprintf("msg-B-%d", stamp)
	postComment(t, roomB, nonceB)

	gotB := countReceived(bViewers, nonceB, posTimeout)
	leakB := countReceived(aViewers, nonceB, negTimeout)
	t.Logf("phase B: %d/%d room-B viewers got it; %d/%d room-A viewers leaked it", gotB, perRoom, leakB, perRoom)
	if gotB != perRoom {
		t.Errorf("room B delivery: %d/%d viewers received B's message (want all)", gotB, perRoom)
	}
	if leakB != 0 {
		t.Errorf("ISOLATION LEAK: %d room-A viewers received room B's message (want 0)", leakB)
	}
}
