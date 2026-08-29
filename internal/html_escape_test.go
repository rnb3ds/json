package internal

import (
	"bytes"
	"testing"
)

// TestHTMLEscape tests the HTMLEscape function
// htmlEscapeViaTo exercises HTMLEscapeTo (the standalone string variant was
// removed as production-dead; the buffer variant implements the same rules).
func htmlEscapeViaTo(s string) string {
	var buf bytes.Buffer
	HTMLEscapeTo(&buf, s)
	return buf.String()
}

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
			result := htmlEscapeViaTo(tt.input)
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
	// Ported to the bytes variant (the string twin was production-dead).
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
			result := NeedsHTMLEscapeBytes([]byte(tt.input))
			if result != tt.expected {
				t.Errorf("NeedsHTMLEscapeBytes(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestHTMLEscape_EdgeCases tests edge cases
func TestHTMLEscape_EdgeCases(t *testing.T) {
	t.Run("multiple consecutive special chars", func(t *testing.T) {
		input := "<<<>>>&&&"
		result := htmlEscapeViaTo(input)
		expected := "\\u003c\\u003c\\u003c\\u003e\\u003e\\u003e\\u0026\\u0026\\u0026"
		if result != expected {
			t.Errorf("got %q, want %q", result, expected)
		}
	})

	t.Run("mixed normal and special", func(t *testing.T) {
		input := "a < b > c & d"
		result := htmlEscapeViaTo(input)
		expected := "a \\u003c b \\u003e c \\u0026 d"
		if result != expected {
			t.Errorf("got %q, want %q", result, expected)
		}
	})

	t.Run("only special chars", func(t *testing.T) {
		input := "<>&"
		result := htmlEscapeViaTo(input)
		expected := "\\u003c\\u003e\\u0026"
		if result != expected {
			t.Errorf("got %q, want %q", result, expected)
		}
	})
}

// Benchmark tests
func BenchmarkHTMLEscapeTo(b *testing.B) {
	s := "<script>alert('XSS')</script>"
	var buf bytes.Buffer
	for i := 0; i < b.N; i++ {
		buf.Reset()
		HTMLEscapeTo(&buf, s)
	}
}

// ============================================================================
// Bytes-based HTML escape tests
// ============================================================================

func TestHTMLEscapeBytes(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{"empty", []byte{}, ""},
		{"no escape", []byte("hello"), "hello"},
		{"less than", []byte("<"), `\u003c`},
		{"greater than", []byte(">"), `\u003e`},
		{"ampersand", []byte("&"), `\u0026`},
		{"mixed", []byte("a<b>c&d"), `a\u003cb\u003ec\u0026d`},
		{"U+2028", []byte{0xe2, 0x80, 0xa8}, `\u2028`},
		{"U+2029", []byte{0xe2, 0x80, 0xa9}, `\u2029`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HTMLEscapeBytes(tt.input)
			if string(got) != tt.want {
				t.Errorf("HTMLEscapeBytes(%q) = %q, want %q", tt.input, string(got), tt.want)
			}
			// Clean up pooled buffer
			if len(tt.input) > 0 {
				PutHTMLEscapeBytes(got)
			}
		})
	}
}

func TestHTMLEscapeBytes_NoEscapeReturnsOriginal(t *testing.T) {
	input := []byte("no special chars")
	result := HTMLEscapeBytes(input)
	if &result[0] != &input[0] {
		t.Error("HTMLEscapeBytes should return original slice when no escaping needed")
	}
}

func TestPutHTMLEscapeBytes(t *testing.T) {
	t.Run("normal size buffer is pooled", func(t *testing.T) {
		input := []byte("<script>alert('xss')</script>")
		result := HTMLEscapeBytes(input)
		// Should not panic
		PutHTMLEscapeBytes(result)
	})

	t.Run("large buffer not pooled", func(t *testing.T) {
		// Create input > 8KB to exceed pool threshold
		input := make([]byte, 9000)
		for i := range input {
			input[i] = 'a'
		}
		result := HTMLEscapeBytes(input)
		// Should not panic even for large buffers
		PutHTMLEscapeBytes(result)
	})

	t.Run("empty buffer", func(t *testing.T) {
		// Should not panic
		PutHTMLEscapeBytes(nil)
		PutHTMLEscapeBytes([]byte{})
	})
}
