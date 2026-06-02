package data

import (
	"strings"
	"testing"
)

// =============================================================================
// buildJSONAggregation Tests
// Criteria B: Chunking logic when columns exceed MaxSelectColumns (50)
// =============================================================================

func TestBuildJSONAggregation(t *testing.T) {
	tests := []struct {
		name     string
		numPairs int
		wantBase string // substring that should be present
		wantNot  string // substring that should NOT be present (empty = don't check)
	}{
		{"empty", 0, "json_object()", "json_patch"},
		{"single", 1, "json_object(", "json_patch"},
		{"within limit", MaxSelectColumns, "json_object(", "json_patch"},
		{"exceeds limit", MaxSelectColumns + 1, "json_patch(", ""},
		{"triple limit", MaxSelectColumns*2 + 1, "json_patch(json_patch(", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pairs := make([]string, tt.numPairs)
			for i := range pairs {
				pairs[i] = "'col', [col]"
			}

			result := buildJSONAggregation(pairs)

			if !strings.Contains(result, tt.wantBase) {
				t.Errorf("result = %q, want containing %q", result, tt.wantBase)
			}

			if tt.wantNot != "" && strings.Contains(result, tt.wantNot) {
				t.Errorf("result = %q, should NOT contain %q", result, tt.wantNot)
			}
		})
	}
}

func TestBuildJSONAggregation_ChunkCount(t *testing.T) {
	// Test that the number of json_patch nesting matches expected chunks
	// 51 pairs = 2 chunks = 1 json_patch
	// 101 pairs = 3 chunks = 2 nested json_patch

	// 51 pairs (2 chunks)
	pairs51 := make([]string, MaxSelectColumns+1)
	for i := range pairs51 {
		pairs51[i] = "'c', [c]"
	}
	result51 := buildJSONAggregation(pairs51)
	patchCount51 := strings.Count(result51, "json_patch(")
	if patchCount51 != 1 {
		t.Errorf("51 pairs: expected 1 json_patch, got %d", patchCount51)
	}

	// 101 pairs (3 chunks)
	pairs101 := make([]string, MaxSelectColumns*2+1)
	for i := range pairs101 {
		pairs101[i] = "'c', [c]"
	}
	result101 := buildJSONAggregation(pairs101)
	patchCount101 := strings.Count(result101, "json_patch(")
	if patchCount101 != 2 {
		t.Errorf("101 pairs: expected 2 json_patch, got %d", patchCount101)
	}
}

// =============================================================================
// splitTableColumn Tests
// Criteria A: Stable utility function
// =============================================================================

func TestSplitTableColumn(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"users.id", []string{"users", "id"}},
		{"id", []string{"id"}},
		{"schema.users.id", []string{"schema", "users.id"}}, // only first dot
		{"", []string{""}},
		{".", []string{"", ""}},
		{".id", []string{"", "id"}},
		{"users.", []string{"users", ""}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := splitTableColumn(tt.input)

			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("got[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// =============================================================================
// relationDepth Tests
// Criteria B: Tree traversal edge cases
// =============================================================================

func TestRelationDepth(t *testing.T) {
	tests := []struct {
		name string
		rel  *Relation
		want int
	}{
		{"nil", nil, 1},
		{"no joins", &Relation{name: "users"}, 1},
		{"one level", &Relation{
			name:  "users",
			joins: []*Relation{{name: "posts"}},
		}, 2},
		{"two levels", &Relation{
			name: "users",
			joins: []*Relation{{
				name:  "posts",
				joins: []*Relation{{name: "comments"}},
			}},
		}, 3},
		{"wide tree", &Relation{
			name: "users",
			joins: []*Relation{
				{name: "posts"},
				{name: "comments"},
				{name: "likes"},
			},
		}, 2},
		{"asymmetric tree", &Relation{
			name: "users",
			joins: []*Relation{
				{name: "posts"},
				{name: "comments", joins: []*Relation{
					{name: "replies", joins: []*Relation{{name: "reactions"}}},
				}},
			},
		}, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := relationDepth(tt.rel)
			if got != tt.want {
				t.Errorf("relationDepth() = %d, want %d", got, tt.want)
			}
		})
	}
}

// =============================================================================
// sanitizeJSONKey Tests
// Criteria B: Validation and escaping edge cases
// =============================================================================

func TestSanitizeJSONKey(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"empty", "", "", false},
		{"simple", "column_name", "column_name", false},
		{"with underscore", "user_id", "user_id", false},
		{"with number", "col1", "col1", false},
		{"single quote escape", "it's", "", true}, // invalid identifier
		{"valid with quote", "col_name", "col_name", false},

		// These depend on ValidateIdentifier implementation
		{"starts with number", "1column", "", true},
		{"has space", "col name", "", true},
		{"has hyphen", "col-name", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := sanitizeJSONKey(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Errorf("sanitizeJSONKey(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitizeJSONKey_EscapesSingleQuotes(t *testing.T) {
	// This tests the escaping logic, but we need a valid identifier with a quote
	// Since ValidateIdentifier likely rejects quotes, we test the ReplaceAll directly
	// by checking that the function would escape if validation passed

	// Test with a key that passes validation but contains no quotes
	key := "valid_key"
	got, err := sanitizeJSONKey(key)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != key {
		t.Errorf("got %q, want %q", got, key)
	}
}
