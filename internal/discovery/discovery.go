// Package discovery keeps a consistent-hash ring in sync with the live set of
// RTN pods. It periodically resolves a headless Service's DNS name (which returns
// one address per ready pod) and pushes the resulting IP:port list into the ring.
package discovery

import (
	"context"
	"log"
	"net"
	"sort"
	"time"
)

// ResolveFunc returns the current addresses (pod IPs) for a host. Injected so
// tests can supply a fake instead of hitting real DNS.
type ResolveFunc func(host string) ([]string, error)

// MemberSetter is the subset of *hashring.Ring that discovery needs.
type MemberSetter interface {
	Set(members []string)
}

// LookupHost is the production resolver (real cluster DNS).
func LookupHost(host string) ([]string, error) {
	return net.LookupHost(host)
}

// resolveMembers resolves host and returns its addresses as sorted "ip:port"
// members. Sorting isn't required for ring correctness (Set is order-independent)
// but keeps logs/debugging stable.
func resolveMembers(resolve ResolveFunc, host, port string) ([]string, error) {
	ips, err := resolve(host)
	if err != nil {
		return nil, err
	}
	members := make([]string, len(ips))
	for i, ip := range ips {
		members[i] = net.JoinHostPort(ip, port)
	}
	sort.Strings(members)
	return members, nil
}

// Run resolves host every interval and updates ring, until ctx is cancelled.
// It resolves once immediately so the ring is populated before the first tick.
func Run(ctx context.Context, ring MemberSetter, resolve ResolveFunc, host, port string, interval time.Duration) {
	update := func() {
		members, err := resolveMembers(resolve, host, port)
		if err != nil {
			log.Printf("discovery: resolving %q failed: %v (keeping previous members)", host, err)
			return
		}
		ring.Set(members)
	}

	update() // populate immediately

	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			update()
		}
	}
}
