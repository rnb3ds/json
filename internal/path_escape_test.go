package internal

import (
	"testing"
)

// TestHasEscapeSequence tests the hasEscapeSequence function
func TestHasEscapeSequence(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{"simple", "user.name", false},
		{"escaped dot", "user\\.name", true},
		{"escaped backslash", "user\\\\name", true},
		{"non-escape char after backslash", "user\\name", false},
		{"mixed", "user\\.name.first", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasEscape := HasEscapeSequence(tt.path)
			if hasEscape != tt.want {
				t.Errorf("HasEscapeSequence(%q) = %v, want %t", tt.path, hasEscape, tt.want)
			}
		})
	}
}

// TestUnescapePathSegment tests the UnescapePathSegment function
func TestUnescapePathSegment(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"simple", "user.name", "user.name"},
		{"escaped dot", "user\\.name", "user.name"},
		{"escaped backslash", "user\\name", "user\\name"},
		{"mixed", "user\\.name.first", "user.name.first"},
		{"multiple escapes", "a\\.b\\.c", "a.b.c"},
		{"trailing backslash", "test\\", "test\\"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unescaped := UnescapePathSegment(tt.path)
			if unescaped != tt.want {
				t.Errorf("UnescapePathSegment(%q) = %q, want %q", tt.path, unescaped, tt.want)
			}
		})
	}
}
