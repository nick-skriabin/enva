// Package search provides fuzzy search functionality for environment variables.
package search

import (
	"sort"
	"strings"

	"github.com/sahilm/fuzzy"

	"enva/internal/env"
)

// SearchResult represents a search result with match information.
type SearchResult struct {
	Var          *env.ResolvedVar
	Score        int
	KeyMatches   []int // indices in key that matched
	ValueMatches []int // indices in value that matched
}

// searchItem implements fuzzy.Source for fuzzy matching.
type searchItem struct {
	idx    int
	text   string
	isKey  bool
	varPtr *env.ResolvedVar
}

type searchSource []searchItem

func (s searchSource) String(i int) string { return s[i].text }
func (s searchSource) Len() int            { return len(s) }

// Search performs fuzzy search over vars, matching against both key and value.
// Returns results sorted by score desc, then key asc.
func Search(vars []*env.ResolvedVar, query string) []*SearchResult {
	if query == "" {
		// No query: return all vars sorted by key
		results := make([]*SearchResult, len(vars))
		for i, v := range vars {
			results[i] = &SearchResult{Var: v, Score: 0}
		}
		sort.Slice(results, func(i, j int) bool {
			return results[i].Var.Key < results[j].Var.Key
		})
		return results
	}

	// Build search source with both keys and values
	source := make(searchSource, 0, len(vars)*2)
	for i, v := range vars {
		source = append(source, searchItem{idx: i, text: v.Key, isKey: true, varPtr: v})
		source = append(source, searchItem{idx: i, text: v.Value, isKey: false, varPtr: v})
	}

	// Perform fuzzy match
	matches := fuzzy.FindFrom(query, source)

	// Aggregate results by var index
	resultMap := make(map[int]*SearchResult)
	for _, m := range matches {
		item := source[m.Index]
		varIdx := item.idx

		if existing, ok := resultMap[varIdx]; ok {
			// Take max score
			if m.Score > existing.Score {
				existing.Score = m.Score
			}
			// Add match indices
			if item.isKey {
				existing.KeyMatches = mergeIndices(existing.KeyMatches, m.MatchedIndexes)
			} else {
				existing.ValueMatches = mergeIndices(existing.ValueMatches, m.MatchedIndexes)
			}
		} else {
			result := &SearchResult{
				Var:   item.varPtr,
				Score: m.Score,
			}
			if item.isKey {
				result.KeyMatches = m.MatchedIndexes
			} else {
				result.ValueMatches = m.MatchedIndexes
			}
			resultMap[varIdx] = result
		}
	}

	// Convert to slice
	results := make([]*SearchResult, 0, len(resultMap))
	for _, r := range resultMap {
		results = append(results, r)
	}

	// Sort by score desc, then key asc
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		return results[i].Var.Key < results[j].Var.Key
	})

	return results
}

// mergeIndices merges two sorted index slices, removing duplicates.
func mergeIndices(a, b []int) []int {
	seen := make(map[int]bool)
	for _, i := range a {
		seen[i] = true
	}
	for _, i := range b {
		seen[i] = true
	}
	result := make([]int, 0, len(seen))
	for i := range seen {
		result = append(result, i)
	}
	sort.Ints(result)
	return result
}

// HighlightMatches returns a string with matched indices highlighted using ANSI.
func HighlightMatches(text string, indices []int, highlightStyle, normalStyle string) string {
	if len(indices) == 0 {
		return normalStyle + text
	}

	indexSet := make(map[int]bool)
	for _, i := range indices {
		indexSet[i] = true
	}

	var sb strings.Builder
	inHighlight := false
	for i, r := range text {
		shouldHighlight := indexSet[i]
		if shouldHighlight && !inHighlight {
			sb.WriteString(highlightStyle)
			inHighlight = true
		} else if !shouldHighlight && inHighlight {
			sb.WriteString(normalStyle)
			inHighlight = false
		}
		sb.WriteRune(r)
	}
	if inHighlight {
		sb.WriteString(normalStyle)
	}

	return sb.String()
}

// FilterByKey returns vars that contain the substring in their key (case-insensitive).
func FilterByKey(vars []*env.ResolvedVar, substr string) []*env.ResolvedVar {
	if substr == "" {
		return vars
	}
	substr = strings.ToLower(substr)
	var result []*env.ResolvedVar
	for _, v := range vars {
		if strings.Contains(strings.ToLower(v.Key), substr) {
			result = append(result, v)
		}
	}
	return result
}
