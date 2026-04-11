package tui

import "github.com/sahilm/fuzzy"

// modalError is a shared error type for all modal validation errors.
type modalError struct {
	msg string
}

func (e *modalError) Error() string {
	return e.msg
}

// fuzzyFilterItems filters a slice of MoveItem by fuzzy-matching query against their labels.
// If query is empty, all items are returned unchanged.
func fuzzyFilterItems(query string, items []MoveItem) []MoveItem {
	if query == "" {
		return items
	}

	// Build list of labels for fuzzy matching
	labels := make([]string, len(items))
	for i, item := range items {
		labels[i] = item.Label
	}

	// Fuzzy match
	matches := fuzzy.Find(query, labels)

	// Return matched items in order of match quality
	result := make([]MoveItem, len(matches))
	for i, match := range matches {
		result[i] = items[match.Index]
	}

	return result
}
