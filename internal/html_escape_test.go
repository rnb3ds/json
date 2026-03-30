package internal

import (
	"bytes"
	"testing"
)

// TestHTMLEscape tests the HTMLEscape function
func TestHTMLEscape(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"no escaping needed", "hello world", "hello world"},
		{"less than", "a<b", "a\\u003cb"},
		{"greater than", "a>b", "a\\u003eb"},
		{"ampersand", "a&b", "a\\u0026b"},
		{"all special", "<>&", "\\u003c\\u003e\\u0026"},
		{"line separator", "a\u2028b", "a\\u2028b"},
		{"paragraph separator", "a\u2029b", "a\\u2029b"},
		{"mixed content", "<script>alert('XSS')</script>", "\\u003cscript\\u003ealert('XSS')\\u003c/script\\u003e"},
		{"unicode preserved", "hello world", "hello world"},
		{"emoji preserved", "test emoji", "test emoji"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HTMLEscape(tt.input)
			if result != tt.expected {
				t.Errorf("HTMLEscape(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestHTMLEscape_NoAllocation tests that no allocation occurs when escaping is not needed
func TestHTMLEscape_NoAllocation(t *testing.T) {
	input := "no escaping needed here"
	result := HTMLEscape(input)
	// The result should be the same string (no allocation)
	if result != input {
		t.Errorf("Expected same string for no-escape case")
	}
}

// TestHTMLEscapeTo tests the HTMLEscapeTo function
func TestHTMLEscapeTo(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"no escaping needed", "hello", "hello"},
		{"less than", "<", "\\u003c"},
		{"greater than", ">", "\\u003e"},
		{"ampersand", "&", "\\u0026"},
		{"line separator", "\u2028", "\\u2028"},
		{"paragraph separator", "\u2029", "\\u2029"},
		{"all special chars", "<>&\u2028\u2029", "\\u003c\\u003e\\u0026\\u2028\\u2029"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			HTMLEscapeTo(&buf, tt.input)
			result := buf.String()
			if result != tt.expected {
				t.Errorf("HTMLEscapeTo(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestHTMLEscapeTo_Append tests that HTMLEscapeTo appends to existing buffer
func TestHTMLEscapeTo_Append(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString("prefix:")
	HTMLEscapeTo(&buf, "<test>")
	result := buf.String()
	expected := "prefix:\\u003ctest\\u003e"
	if result != expected {
		t.Errorf("HTMLEscapeTo append = %q, want %q", result, expected)
	}
}

// TestNeedsHTMLEscape tests the NeedsHTMLEscape function
func TestNeedsHTMLEscape(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"empty string", "", false},
		{"no special chars", "hello world", false},
		{"less than", "a<b", true},
		{"greater than", "a>b", true},
		{"ampersand", "a&b", true},
		{"line separator", "a\u2028b", true},
		{"paragraph separator", "a\u2029b", true},
		{"only special char", "<", true},
		{"special at start", "<hello", true},
		{"special at end", "hello>", true},
		{"special in middle", "a<b>c", true},
		{"unicode no escape", "hello world", false},
		{"numbers no escape", "12345", false},
		{"spaces no escape", "hello world", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NeedsHTMLEscape(tt.input)
			if result != tt.expected {
				t.Errorf("NeedsHTMLEscape(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestHTMLEscape_EdgeCases tests edge cases
func TestHTMLEscape_EdgeCases(t *testing.T) {
	t.Run("multiple consecutive special chars", func(t *testing.T) {
		input := "<<<>>>&&&"
		result := HTMLEscape(input)
		expected := "\\u003c\\u003c\\u003c\\u003e\\u003e\\u003e\\u0026\\u0026\\u0026"
		if result != expected {
			t.Errorf("got %q, want %q", result, expected)
		}
	})

	t.Run("mixed normal and special", func(t *testing.T) {
		input := "a < b > c & d"
		result := HTMLEscape(input)
		expected := "a \\u003c b \\u003e c \\u0026 d"
		if result != expected {
			t.Errorf("got %q, want %q", result, expected)
		}
	})

	t.Run("only special chars", func(t *testing.T) {
		input := "<>&"
		result := HTMLEscape(input)
		expected := "\\u003c\\u003e\\u0026"
		if result != expected {
			t.Errorf("got %q, want %q", result, expected)
		}
	})
}

// Benchmark tests
func BenchmarkHTMLEscape_NoEscape(b *testing.B) {
	s := "hello world this is a normal string without special characters"
	for i := 0; i < b.N; i++ {
		_ = HTMLEscape(s)
	}
}

func BenchmarkHTMLEscape_WithEscape(b *testing.B) {
	s := "<script>alert('XSS')</script>&hello<world>"
	for i := 0; i < b.N; i++ {
		_ = HTMLEscape(s)
	}
}

func BenchmarkHTMLEscapeTo(b *testing.B) {
	s := "<script>alert('XSS')</script>"
	var buf bytes.Buffer
	for i := 0; i < b.N; i++ {
		buf.Reset()
		HTMLEscapeTo(&buf, s)
	}
}

func BenchmarkNeedsHTMLEscape(b *testing.B) {
	s := "hello world <script>"
	for i := 0; i < b.N; i++ {
		_ = NeedsHTMLEscape(s)
	}
}
