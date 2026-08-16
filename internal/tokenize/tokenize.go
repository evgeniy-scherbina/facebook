// Package tokenize turns post content into the set of index terms it should be
// findable by. Both the ingestion path (indexing a post) and the query path
// (parsing a search string) must use the SAME tokenizer, or a post indexed under
// "hello" would never match a search for "Hello!".
package tokenize

import "strings"

// Terms lowercases content, splits it on any non-alphanumeric character, and
// returns the unique terms in first-seen order.
//
// Deliberately simple for now: no stemming ("running" != "run"), no stopword
// removal ("the" is indexed), no unicode normalization.
func Terms(content string) []string {
	fields := strings.FieldsFunc(strings.ToLower(content), func(r rune) bool {
		return !isAlphanumeric(r)
	})

	seen := make(map[string]struct{}, len(fields))
	terms := make([]string, 0, len(fields))
	for _, f := range fields {
		if _, dup := seen[f]; dup {
			continue
		}
		seen[f] = struct{}{}
		terms = append(terms, f)
	}
	return terms
}

func isAlphanumeric(r rune) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9')
}
