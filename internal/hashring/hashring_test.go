package hashring

import (
	"fmt"
	"testing"
)

func keys(n int) []string {
	ks := make([]string, n)
	for i := range ks {
		ks[i] = fmt.Sprintf("room-%d", i)
	}
	return ks
}

// Empty ring: Get returns ok=false.
func TestGet_EmptyRing(t *testing.T) {
	r := New(0)
	if _, ok := r.Get("room-1"); ok {
		t.Fatal("expected ok=false on an empty ring")
	}
}

// Deterministic: same members => same key->member mapping, every time.
func TestGet_Deterministic(t *testing.T) {
	members := []string{"pod-a", "pod-b", "pod-c"}

	r1 := New(0)
	r1.Set(members)
	r2 := New(0)
	r2.Set(members) // fresh ring, same members

	for _, k := range keys(1000) {
		m1, _ := r1.Get(k)
		m2, _ := r2.Get(k)
		if m1 != m2 {
			t.Fatalf("non-deterministic: %q -> %q vs %q", k, m1, m2)
		}
		if m1 == "" {
			t.Fatalf("key %q mapped to no member", k)
		}
	}
}

// The core consistent-hashing property: adding a member remaps only a small
// fraction of keys, and every moved key goes TO the new member (never between
// two existing members).
func TestSet_MinimalRemapOnAdd(t *testing.T) {
	const nKeys = 10000
	ks := keys(nKeys)

	r := New(0)
	r.Set([]string{"pod-a", "pod-b", "pod-c", "pod-d"}) // 4 members

	before := make(map[string]string, nKeys)
	for _, k := range ks {
		before[k], _ = r.Get(k)
	}

	r.Set([]string{"pod-a", "pod-b", "pod-c", "pod-d", "pod-e"}) // + pod-e

	moved := 0
	for _, k := range ks {
		now, _ := r.Get(k)
		if now != before[k] {
			moved++
			if now != "pod-e" {
				t.Fatalf("key %q moved %q -> %q, but only moves to the new member (pod-e) are allowed",
					k, before[k], now)
			}
		}
	}

	frac := float64(moved) / float64(nKeys)
	t.Logf("remapped %d/%d keys (%.1f%%) on 4->5 members", moved, nKeys, frac*100)

	// Ideal is ~1/5 = 20%. Assert it's small (a modulo hash would move ~80%),
	// and non-zero.
	if frac == 0 {
		t.Fatal("no keys moved after adding a member")
	}
	if frac > 0.35 {
		t.Fatalf("too many keys remapped: %.1f%% (expected ~20%%)", frac*100)
	}
}

// Distribution: with many virtual nodes, keys spread roughly evenly across members.
func TestSet_DistributionRoughlyEven(t *testing.T) {
	const nKeys = 30000
	members := []string{"pod-a", "pod-b", "pod-c"}

	r := New(0)
	r.Set(members)

	counts := map[string]int{}
	for _, k := range keys(nKeys) {
		m, _ := r.Get(k)
		counts[m]++
	}

	mean := float64(nKeys) / float64(len(members))
	for m, c := range counts {
		ratio := float64(c) / mean
		t.Logf("%s: %d keys (%.2fx mean)", m, c, ratio)
		if ratio < 0.5 || ratio > 1.5 {
			t.Fatalf("%s got %d keys, too far from mean %.0f (%.2fx)", m, c, mean, ratio)
		}
	}
}
