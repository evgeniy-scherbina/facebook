// Package hashring is a small consistent-hash ring (ketama-style) mapping keys
// (room IDs) to members (RTN pods). Adding/removing a member remaps only a
// fraction of keys, and a key only ever moves TO a newly-added member — never
// between two existing members.
package hashring

import (
	"crypto/sha1"
	"encoding/binary"
	"sort"
	"strconv"
	"sync"
)

// defaultVNodes is how many points each member occupies on the ring. More
// virtual nodes => smoother key distribution across members.
const defaultVNodes = 150

type Ring struct {
	mu      sync.RWMutex
	vnodes  int
	points  []uint32          // sorted ring positions
	owner   map[uint32]string // ring position -> member
	members []string          // current member set (for introspection)
}

// New returns an empty ring. vnodes <= 0 uses the default.
func New(vnodes int) *Ring {
	if vnodes <= 0 {
		vnodes = defaultVNodes
	}
	return &Ring{
		vnodes: vnodes,
		owner:  make(map[uint32]string),
	}
}

// Set replaces the ring's members. Safe to call whenever the pod set changes.
func (r *Ring) Set(members []string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.points = r.points[:0]
	r.owner = make(map[uint32]string, len(members)*r.vnodes)
	r.members = append([]string(nil), members...)

	for _, m := range members {
		for i := 0; i < r.vnodes; i++ {
			p := hash(m + "#" + strconv.Itoa(i))
			r.points = append(r.points, p)
			r.owner[p] = m // last-writer-wins on the rare collision
		}
	}
	sort.Slice(r.points, func(i, j int) bool { return r.points[i] < r.points[j] })
}

// Get returns the member that owns key. ok is false only when the ring is empty.
func (r *Ring) Get(key string) (member string, ok bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.points) == 0 {
		return "", false
	}
	h := hash(key)
	// First ring point >= h; wrap around to the start if none.
	i := sort.Search(len(r.points), func(i int) bool { return r.points[i] >= h })
	if i == len(r.points) {
		i = 0
	}
	return r.owner[r.points[i]], true
}

// Members returns the current member set.
func (r *Ring) Members() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]string(nil), r.members...)
}

// hash uses SHA-1's strong avalanche so that near-identical vnode keys
// ("pod-a#0", "pod-a#1", …) scatter evenly around the ring. FNV clumped them.
func hash(s string) uint32 {
	sum := sha1.Sum([]byte(s))
	return binary.BigEndian.Uint32(sum[:4])
}
