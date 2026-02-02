package search

import (
	"testing"

	"github.com/nick-skriabin/enva/internal/env"
)

func makeVars(pairs ...string) []*env.ResolvedVar {
	var vars []*env.ResolvedVar
	for i := 0; i < len(pairs); i += 2 {
		vars = append(vars, &env.ResolvedVar{
			Key:           pairs[i],
			Value:         pairs[i+1],
			DefinedAtPath: "/test",
		})
	}
	return vars
}

func TestSearchEmptyQuery(t *testing.T) {
	vars := makeVars(
		"ZEBRA", "last",
		"ALPHA", "first",
		"MIDDLE", "middle",
	)

	results := Search(vars, "")

	if len(results) != 3 {
		t.Errorf("Search('') returned %d results, want 3", len(results))
	}

	// Should be sorted alphabetically by key
	expected := []string{"ALPHA", "MIDDLE", "ZEBRA"}
	for i, want := range expected {
		if results[i].Var.Key != want {
			t.Errorf("Search('')[%d].Key = %q, want %q", i, results[i].Var.Key, want)
		}
	}
}

func TestSearchMatchesKey(t *testing.T) {
	vars := makeVars(
		"API_KEY", "secret",
		"DATABASE_URL", "postgres://",
		"DEBUG", "true",
	)

	results := Search(vars, "api")

	if len(results) != 1 {
		t.Errorf("Search('api') returned %d results, want 1", len(results))
		return
	}

	if results[0].Var.Key != "API_KEY" {
		t.Errorf("Search('api')[0].Key = %q, want 'API_KEY'", results[0].Var.Key)
	}

	if len(results[0].KeyMatches) == 0 {
		t.Error("Search('api') should have KeyMatches")
	}
}

func TestSearchMatchesValue(t *testing.T) {
	vars := makeVars(
		"API_KEY", "secret",
		"DATABASE_URL", "postgres://localhost",
		"REDIS_URL", "redis://localhost",
	)

	results := Search(vars, "postgres")

	if len(results) != 1 {
		t.Errorf("Search('postgres') returned %d results, want 1", len(results))
		return
	}

	if results[0].Var.Key != "DATABASE_URL" {
		t.Errorf("Search('postgres')[0].Key = %q, want 'DATABASE_URL'", results[0].Var.Key)
	}

	if len(results[0].ValueMatches) == 0 {
		t.Error("Search('postgres') should have ValueMatches")
	}
}

func TestSearchFuzzyMatching(t *testing.T) {
	vars := makeVars(
		"DATABASE_URL", "postgres://localhost",
		"DEBUG", "true",
		"API_KEY", "secret",
	)

	// "dbu" should fuzzy match "DATABASE_URL"
	results := Search(vars, "dbu")

	found := false
	for _, r := range results {
		if r.Var.Key == "DATABASE_URL" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Search('dbu') should match 'DATABASE_URL' via fuzzy matching")
	}
}

func TestSearchSortsByScore(t *testing.T) {
	vars := makeVars(
		"API", "value",
		"API_KEY", "value",
		"MY_API_KEY", "value",
	)

	results := Search(vars, "API")

	if len(results) < 2 {
		t.Fatalf("Search('API') returned %d results, want at least 2", len(results))
	}

	// "API" should have higher score than "API_KEY" (exact vs partial match)
	if results[0].Var.Key != "API" {
		t.Errorf("Search('API')[0].Key = %q, want 'API' (exact match first)", results[0].Var.Key)
	}
}

func TestSearchNoResults(t *testing.T) {
	vars := makeVars(
		"API_KEY", "secret",
		"DATABASE_URL", "postgres://",
	)

	results := Search(vars, "zzzznotfound")

	if len(results) != 0 {
		t.Errorf("Search('zzzznotfound') returned %d results, want 0", len(results))
	}
}

func TestFilterByKey(t *testing.T) {
	vars := makeVars(
		"API_KEY", "secret",
		"API_SECRET", "more_secret",
		"DATABASE_URL", "postgres://",
	)

	t.Run("matches substring", func(t *testing.T) {
		filtered := FilterByKey(vars, "API")
		if len(filtered) != 2 {
			t.Errorf("FilterByKey('API') returned %d results, want 2", len(filtered))
		}
	})

	t.Run("case insensitive", func(t *testing.T) {
		filtered := FilterByKey(vars, "api")
		if len(filtered) != 2 {
			t.Errorf("FilterByKey('api') returned %d results, want 2", len(filtered))
		}
	})

	t.Run("empty query returns all", func(t *testing.T) {
		filtered := FilterByKey(vars, "")
		if len(filtered) != 3 {
			t.Errorf("FilterByKey('') returned %d results, want 3", len(filtered))
		}
	})

	t.Run("no matches", func(t *testing.T) {
		filtered := FilterByKey(vars, "NOTFOUND")
		if len(filtered) != 0 {
			t.Errorf("FilterByKey('NOTFOUND') returned %d results, want 0", len(filtered))
		}
	})
}

func TestHighlightMatches(t *testing.T) {
	tests := []struct {
		text      string
		indices   []int
		highlight string
		normal    string
		expected  string
	}{
		{
			text:      "API_KEY",
			indices:   []int{0, 1, 2},
			highlight: "[",
			normal:    "]",
			expected:  "[API]_KEY",
		},
		{
			text:      "DATABASE",
			indices:   []int{0, 4},
			highlight: "[",
			normal:    "]",
			expected:  "[D]ATA[B]ASE",
		},
		{
			text:      "TEST",
			indices:   []int{},
			highlight: "[",
			normal:    "]",
			expected:  "]TEST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			got := HighlightMatches(tt.text, tt.indices, tt.highlight, tt.normal)
			if got != tt.expected {
				t.Errorf("HighlightMatches(%q, %v) = %q, want %q", tt.text, tt.indices, got, tt.expected)
			}
		})
	}
}

func TestMergeIndices(t *testing.T) {
	tests := []struct {
		a        []int
		b        []int
		expected []int
	}{
		{[]int{1, 3, 5}, []int{2, 4, 6}, []int{1, 2, 3, 4, 5, 6}},
		{[]int{1, 2, 3}, []int{2, 3, 4}, []int{1, 2, 3, 4}},
		{[]int{}, []int{1, 2}, []int{1, 2}},
		{[]int{1, 2}, []int{}, []int{1, 2}},
		{[]int{}, []int{}, []int{}},
	}

	for _, tt := range tests {
		got := mergeIndices(tt.a, tt.b)
		if len(got) != len(tt.expected) {
			t.Errorf("mergeIndices(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.expected)
			continue
		}
		for i := range got {
			if got[i] != tt.expected[i] {
				t.Errorf("mergeIndices(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.expected)
				break
			}
		}
	}
}
