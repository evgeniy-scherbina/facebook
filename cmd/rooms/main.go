// Command rooms reports how many Redis room-channel subscriptions each
// real-time-ntfn replica holds, by querying every pod's /debug/rooms endpoint
// (the ELB would only hit one random pod, so we exec into each pod instead).
//
// Efficiency signal:
//
//	total subscriptions == distinct active rooms  => CO-LOCATED (each room on 1 replica)
//	total subscriptions >  distinct active rooms  => rooms duplicated across replicas
//
// Usage:
//
//	go run ./cmd/rooms
//
// Only rooms with at least one connected viewer appear, so connect some SSE
// clients (run the e2e test, open browser tabs) before/while querying.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type debugRooms struct {
	Pod   string         `json:"pod"`
	Rooms map[string]int `json:"rooms"`
}

func main() {
	ns := getenv("NAMESPACE", "chat-app")
	label := "app=real-time-ntfn-service"
	port := getenv("PORT", "8081")

	out, err := exec.Command("kubectl", "get", "pods", "-n", ns, "-l", label,
		"-o", "jsonpath={.items[*].metadata.name}").Output()
	if err != nil {
		fatalf("listing pods: %v", err)
	}
	pods := strings.Fields(string(out))
	if len(pods) == 0 {
		fatalf("no real-time-ntfn pods found in namespace %s", ns)
	}

	roomPods := map[string][]string{} // room -> replicas subscribed to it
	totalSubs := 0

	fmt.Println("Per-replica /debug/rooms:")
	for _, pod := range pods {
		raw, err := exec.Command("kubectl", "exec", "-n", ns, pod, "--",
			"wget", "-qO-", "http://localhost:"+port+"/debug/rooms").Output()
		if err != nil {
			fmt.Printf("  %s: query failed: %v\n", pod, err)
			continue
		}
		var d debugRooms
		if err := json.Unmarshal(raw, &d); err != nil {
			fmt.Printf("  %s: bad json: %v\n", pod, err)
			continue
		}
		fmt.Printf("  %s: %v\n", pod, d.Rooms)
		totalSubs += len(d.Rooms)
		for r := range d.Rooms {
			roomPods[r] = append(roomPods[r], pod)
		}
	}

	distinct := len(roomPods)
	fmt.Println()
	fmt.Printf("distinct active rooms : %d\n", distinct)
	fmt.Printf("total subscriptions   : %d  (sum of rooms across all replicas)\n", totalSubs)

	switch {
	case distinct == 0:
		fmt.Println("=> no active rooms (connect some viewers first)")
	case totalSubs == distinct:
		fmt.Println("=> CO-LOCATED: each room subscribed on exactly 1 replica (efficient)")
	default:
		fmt.Printf("=> NOT co-located: %d duplicate subscription(s)\n", totalSubs-distinct)
		for r, ps := range roomPods {
			if len(ps) > 1 {
				fmt.Printf("   room %q is on %d replicas\n", r, len(ps))
			}
		}
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func fatalf(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", a...)
	os.Exit(1)
}
