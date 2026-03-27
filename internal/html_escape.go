package internal

import (
	"bytes"
)

// HTMLEscape performs HTML escaping on JSON string.
// Compatible with encoding/json: escapes <, >, &, U+2028, U+2029.
//
// This is the centralized implementation used by:
//   - json.HTMLEscape() for encoding/json compatibility
//   - json.HTMLEscapeBuffer() for buffer operations
//   - Any other internal components needing HTML escaping
//
// PERFORMANCE: Uses bytes.Buffer with Grow() to minimize allocations.
func HTMLEscape(s string) string {
	// Fast path: check if escaping is needed
	needsEscape := false
	for _, r := range s {
		switch r {
		case '<', '>', '&', '\u2028', '\u2029':
			needsEscape = true
		}
		if needsEscape {
			break
		}
	}
	if !needsEscape {
		return s // No allocation for strings without special characters
	}

	// Slow path: perform escaping
	var buf bytes.Buffer
	buf.Grow(len(s) + len(s)/10) // Pre-allocate with 10% extra for escapes
	for _, r := range s {
		switch r {
		case '<':
			buf.WriteString("\\u003c")
		case '>':
			buf.WriteString("\\u003e")
		case '&':
			buf.WriteString("\\u0026")
		case '\u2028': // Line Separator - required for JSON-in-JS compatibility
			buf.WriteString("\\u2028")
		case '\u2029': // Paragraph Separator - required for JSON-in-JS compatibility
			buf.WriteString("\\u2029")
		default:
			buf.WriteRune(r)
		}
	}
	return buf.String()
}

// HTMLEscapeTo writes HTML-escaped JSON to the destination buffer.
// This is more efficient than HTMLEscape when writing to an existing buffer.
func HTMLEscapeTo(dst *bytes.Buffer, s string) {
	for _, r := range s {
		switch r {
		case '<':
			dst.WriteString("\\u003c")
		case '>':
			dst.WriteString("\\u003e")
		case '&':
			dst.WriteString("\\u0026")
		case '\u2028':
			dst.WriteString("\\u2028")
		case '\u2029':
			dst.WriteString("\\u2029")
		default:
			dst.WriteRune(r)
		}
	}
}

// NeedsHTMLEscape checks if a string needs HTML escaping.
// PERFORMANCE: Fast check to avoid allocation when no escaping is needed.
func NeedsHTMLEscape(s string) bool {
	for _, r := range s {
		switch r {
		case '<', '>', '&', '\u2028', '\u2029':
			return true
		}
	}
	return false
}
