package internal

import "testing"

// ============================================================================
// Boundary tests for internal/path.go low-coverage paths.
// ============================================================================

// --- NewWildcardSegment (path.go:458, 0% coverage) ---

func TestNewWildcardSegment(t *testing.T) {
	s := NewWildcardSegment()
	if s.Type != WildcardSegment {
		t.Errorf("Type = %v, want WildcardSegment", s.Type)
	}
	if !s.IsWildcardSegment() {
		t.Error("expected wildcard flag set")
	}
}

// --- validateArrayIndexContent (path.go:1239, 36% coverage) ---

func TestValidateArrayIndexContent(t *testing.T) {
	const maxIdx = 1000000
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{"empty", "", true},
		{"wildcard", "*", false},
		{"append", "+", false},
		{"single_digit", "5", false},
		{"multi_digit", "42", false},
		{"negative", "-1", false},
		{"slice", "1:3", false},
		{"slice_open_end", "1:", false},
		{"slice_open_start", ":3", false},
		{"slice_full", ":", false},
		{"slice_with_step", "0:5:2", false},
		{"non_numeric", "abc", true},
		{"overflow", "99999999999", true},
		{"negative_only", "-", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateArrayIndexContent(tt.content, maxIdx)
			if tt.wantErr && err == nil {
				t.Errorf("validateArrayIndexContent(%q): expected error, got nil", tt.content)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("validateArrayIndexContent(%q): unexpected error: %v", tt.content, err)
			}
		})
	}
}

// --- validateNumericIndex (path.go:1286, 32% coverage) ---

func TestValidateNumericIndex(t *testing.T) {
	const maxIdx = 1000000
	tests := []struct {
		name    string
		s       string
		wantErr bool
	}{
		{"empty", "", true},
		{"negative_only", "-", true},
		{"single_digit", "5", false},
		{"single_non_digit", "a", true},
		{"multi_digit", "42", false},
		{"negative_number", "-1", false},
		{"non_numeric", "abc", true},
		{"mixed", "1a", true},
		{"overflow", "99999999999", true},
		{"out_of_range", "2000000", true},
		{"zero", "0", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateNumericIndex(tt.s, maxIdx)
			if tt.wantErr && err == nil {
				t.Errorf("validateNumericIndex(%q): expected error, got nil", tt.s)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("validateNumericIndex(%q): unexpected error: %v", tt.s, err)
			}
		})
	}
}
