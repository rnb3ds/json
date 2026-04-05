package internal

import (
	"testing"
)

// ============================================================================
// UNIQUE ARRAY OPTIMIZED TESTS
// ============================================================================

func TestUniqueArrayOptimized(t *testing.T) {
	tests := []struct {
		name             string
		input            []any
		expectedLen      int
		expectedElements map[any]bool
	}{
		{
			name:             "empty_array",
			input:            []any{},
			expectedLen:      0,
			expectedElements: nil,
		},
		{
			name:             "single_element",
			input:            []any{"a"},
			expectedLen:      1,
			expectedElements: map[any]bool{"a": true},
		},
		{
			name:             "no_duplicates_strings",
			input:            []any{"a", "b", "c"},
			expectedLen:      3,
			expectedElements: map[any]bool{"a": true, "b": true, "c": true},
		},
		{
			name:             "all_duplicates",
			input:            []any{"a", "a", "a"},
			expectedLen:      1,
			expectedElements: map[any]bool{"a": true},
		},
		{
			name:             "some_duplicates",
			input:            []any{"a", "b", "a", "c", "b"},
			expectedLen:      3,
			expectedElements: map[any]bool{"a": true, "b": true, "c": true},
		},
		{
			name:             "mixed_types",
			input:            []any{1, "a", 1.0, "a", true},
			expectedLen:      4,
			expectedElements: map[any]bool{1: true, "a": true, 1.0: true, true: true},
		},
		{
			name:             "with_nil",
			input:            []any{nil, "a", nil, "b"},
			expectedLen:      3,
			expectedElements: map[any]bool{nil: true, "a": true, "b": true},
		},
		{
			name:             "integers",
			input:            []any{1, 2, 1, 3, 2, 4},
			expectedLen:      4,
			expectedElements: map[any]bool{1: true, 2: true, 3: true, 4: true},
		},
		{
			name:             "floats",
			input:            []any{1.1, 2.2, 1.1, 3.3},
			expectedLen:      3,
			expectedElements: map[any]bool{1.1: true, 2.2: true, 3.3: true},
		},
		{
			name:             "booleans",
			input:            []any{true, false, true, false},
			expectedLen:      2,
			expectedElements: map[any]bool{true: true, false: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UniqueArrayOptimized(tt.input)

			// Check length
			if len(result) != tt.expectedLen {
				t.Errorf("UniqueArrayOptimized() length = %d, want %d", len(result), tt.expectedLen)
				return
			}

			// For empty arrays, nothing more to check
			if tt.expectedLen == 0 {
				return
			}

			// Check that all expected elements are present
			for _, v := range result {
				if !tt.expectedElements[v] {
					t.Errorf("UniqueArrayOptimized() returned unexpected value %v", v)
				}
			}
		})
	}
}
