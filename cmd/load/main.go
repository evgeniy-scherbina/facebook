// Command load opens a fixed set of SSE viewers across several rooms and holds
// the connections open until interrupted (Ctrl+C). While it runs, inspect how
// the connections are distributed across real-time-ntfn replicas with:
//
//	go run ./cmd/rooms
//
// Usage:
//
//	go run ./cmd/load                          # 2 rooms x 10 viewers
//	go run ./cmd/load -rooms 4 -clients 5      # 4 rooms x 5 viewers
//	go run ./cmd/load -url http://other-elb    # different target
package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	baseURL := flag.String("url", "http://a000aed69195e47dd85cfde7487c9291-139361245.us-east-1.elb.amazonaws.com", "base URL (ingress ELB)")
	numRooms := flag.Int("rooms", 3, "number of rooms")
	perRoom := flag.Int("clients", 10, "SSE viewers per room")
	flag.Parse()

	client := &http.Client{} // no timeout: SSE connections are long-lived

	total := 0
	for i := 0; i < *numRooms; i++ {
		room := fmt.Sprintf("room-%d", i+1)
		for j := 0; j < *perRoom; j++ {
			resp, err := client.Get(*baseURL + "/events?room=" + room)
			if err != nil {
				fmt.Printf("connect (room %s) failed: %v\n", room, err)
				continue
			}
			if resp.StatusCode != http.StatusOK {
				fmt.Printf("connect (room %s) status %d\n", room, resp.StatusCode)
				resp.Body.Close()
				continue
			}
			// Drain the stream in the background to keep the connection alive.
			go func(body io.ReadCloser) {
				defer body.Close()
				_, _ = io.Copy(io.Discard, body)
			}(resp.Body)
			total++
		}
		fmt.Printf("connected %d viewers to %q\n", *perRoom, room)
	}

	fmt.Printf("\n%d viewers connected across %d rooms.\n", total, *numRooms)
	fmt.Println("Inspect distribution:  go run ./cmd/rooms")
	fmt.Println("Press Ctrl+C to disconnect and exit.")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	fmt.Println("\ndisconnecting...")
}
