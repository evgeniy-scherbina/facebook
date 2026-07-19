// Package discovery keeps a consistent-hash ring in sync with the live set of
// RTN pods by watching their EndpointSlices.
package discovery

import (
	"context"
	"fmt"
	"log"
	"net"
	"sort"
	"strconv"
	"time"

	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

// RingSetter is the subset of *hashring.Ring that discovery needs.
type RingSetter interface {
	Set(members []string)
}

// Watch watches the EndpointSlices for serviceName in namespace and updates ring
// whenever the ready pod set changes. It uses in-cluster config (the pod's
// ServiceAccount) and blocks until ctx is cancelled.
func Watch(ctx context.Context, ring RingSetter, namespace, serviceName string) error {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("in-cluster config: %w", err)
	}
	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("kubernetes client: %w", err)
	}
	return watchWithClient(ctx, ring, client, namespace, serviceName)
}

// watchWithClient is the testable core: same as Watch but with an injected
// client (e.g. a fake clientset in tests).
func watchWithClient(ctx context.Context, ring RingSetter, client kubernetes.Interface, namespace, serviceName string) error {
	// EndpointSlices for a Service carry the label kubernetes.io/service-name=<svc>.
	selector := discoveryv1.LabelServiceName + "=" + serviceName

	factory := informers.NewSharedInformerFactoryWithOptions(
		client,
		30*time.Second, // resync: safety net so a missed event still self-heals
		informers.WithNamespace(namespace),
		informers.WithTweakListOptions(func(opts *metav1.ListOptions) {
			opts.LabelSelector = selector
		}),
	)
	sliceInformer := factory.Discovery().V1().EndpointSlices()
	lister := sliceInformer.Lister()

	// Recompute the full member set from all slices on every change (simple and
	// correct: the ring is a pure function of the current slices).
	update := func() {
		ptrs, err := lister.EndpointSlices(namespace).List(labels.Everything())
		if err != nil {
			log.Printf("discovery: listing endpointslices: %v", err)
			return
		}
		slices := make([]discoveryv1.EndpointSlice, 0, len(ptrs))
		for _, s := range ptrs {
			slices = append(slices, *s)
		}
		members := membersFromSlices(slices)
		ring.Set(members)
		log.Printf("discovery: %d ready member(s): %v", len(members), members)
	}

	if _, err := sliceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(any) { update() },
		UpdateFunc: func(_, _ any) { update() },
		DeleteFunc: func(any) { update() },
	}); err != nil {
		return fmt.Errorf("add event handler: %w", err)
	}

	factory.Start(ctx.Done())
	if !cache.WaitForCacheSync(ctx.Done(), sliceInformer.Informer().HasSynced) {
		return fmt.Errorf("discovery: endpointslice cache failed to sync")
	}
	update() // ensure the ring is populated right after the initial sync
	<-ctx.Done()
	return nil
}

// membersFromSlices returns the ready pod addresses ("ip:port") found across the
// given EndpointSlices. Only endpoints whose Ready condition is true are
// included; the port comes from each slice's port list (the RTN container port).
// The result is deduplicated and sorted — Set is order-independent, but a stable
// order keeps logs readable.
func membersFromSlices(slices []discoveryv1.EndpointSlice) []string {
	seen := map[string]struct{}{}
	for _, s := range slices {
		port := slicePort(s)
		if port == 0 {
			continue // no usable port -> can't route to these endpoints
		}
		for _, ep := range s.Endpoints {
			if !endpointReady(ep) {
				continue
			}
			for _, addr := range ep.Addresses {
				seen[net.JoinHostPort(addr, strconv.Itoa(int(port)))] = struct{}{}
			}
		}
	}

	members := make([]string, 0, len(seen))
	for m := range seen {
		members = append(members, m)
	}
	sort.Strings(members)
	return members
}

// endpointReady reports whether an endpoint can receive traffic. A nil Ready
// pointer means "unknown", which we conservatively treat as not ready.
func endpointReady(ep discoveryv1.Endpoint) bool {
	return ep.Conditions.Ready != nil && *ep.Conditions.Ready
}

// slicePort returns the first defined port of a slice, or 0 if none.
func slicePort(s discoveryv1.EndpointSlice) int32 {
	for _, p := range s.Ports {
		if p.Port != nil {
			return *p.Port
		}
	}
	return 0
}
