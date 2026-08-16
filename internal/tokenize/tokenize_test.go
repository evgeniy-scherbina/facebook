package tokenize

import (
	"reflect"
	"testing"
)

func TestTerms(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{"empty", "", []string{}},
		{"simple", "hello world", []string{"hello", "world"}},
		{"lowercased", "Hello WORLD", []string{"hello", "world"}},
		{"punctuation split out", "hello, world! it's here.", []string{"hello", "world", "it", "s", "here"}},
		{"deduped, first-seen order", "go go gophers go", []string{"go", "gophers"}},
		{"digits kept", "go1.26 released", []string{"go1", "26", "released"}},
		{"collapses whitespace", "  spaced   out  ", []string{"spaced", "out"}},
		{"punctuation only", "!!! ??? ...", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Terms(tt.content)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("Terms(%q) = %v, want %v", tt.content, got, tt.want)
			}
		})
	}
}
