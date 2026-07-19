package discovery

import (
	"reflect"
	"testing"

	discoveryv1 "k8s.io/api/discovery/v1"
)

func boolp(b bool) *bool    { return &b }
func int32p(i int32) *int32 { return &i }

func TestMembersFromSlices(t *testing.T) {
	tests := []struct {
		name   string
		slices []discoveryv1.EndpointSlice
		want   []string
	}{
		{
			name:   "empty",
			slices: nil,
			want:   []string{},
		},
		{
			name: "ready only — excludes not-ready and unknown(nil)",
			slices: []discoveryv1.EndpointSlice{{
				Ports: []discoveryv1.EndpointPort{{Port: int32p(8081)}},
				Endpoints: []discoveryv1.Endpoint{
					{Addresses: []string{"10.0.0.1"}, Conditions: discoveryv1.EndpointConditions{Ready: boolp(true)}},
					{Addresses: []string{"10.0.0.2"}, Conditions: discoveryv1.EndpointConditions{Ready: boolp(false)}},
					{Addresses: []string{"10.0.0.3"}, Conditions: discoveryv1.EndpointConditions{}}, // nil = unknown
				},
			}},
			want: []string{"10.0.0.1:8081"},
		},
		{
			name: "aggregates multiple slices, sorted + deduped",
			slices: []discoveryv1.EndpointSlice{
				{
					Ports: []discoveryv1.EndpointPort{{Port: int32p(8081)}},
					Endpoints: []discoveryv1.Endpoint{
						{Addresses: []string{"10.0.0.9"}, Conditions: discoveryv1.EndpointConditions{Ready: boolp(true)}},
					},
				},
				{
					Ports: []discoveryv1.EndpointPort{{Port: int32p(8081)}},
					Endpoints: []discoveryv1.Endpoint{
						{Addresses: []string{"10.0.0.2"}, Conditions: discoveryv1.EndpointConditions{Ready: boolp(true)}},
						{Addresses: []string{"10.0.0.9"}, Conditions: discoveryv1.EndpointConditions{Ready: boolp(true)}}, // dup across slices
					},
				},
			},
			want: []string{"10.0.0.2:8081", "10.0.0.9:8081"},
		},
		{
			name: "slice with no port is skipped",
			slices: []discoveryv1.EndpointSlice{{
				Endpoints: []discoveryv1.Endpoint{
					{Addresses: []string{"10.0.0.1"}, Conditions: discoveryv1.EndpointConditions{Ready: boolp(true)}},
				},
			}},
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := membersFromSlices(tt.slices)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}
