package internal

import (
	"bytes"
	"sync"
)

// htmlEscapeBufferPool pools buffers for HTML escaping operations
// PERFORMANCE: Reduces allocations in hot paths
var htmlEscapeBufferPool = sync.Pool{
	New: func() any {
		buf := &bytes.Buffer{}
		buf.Grow(256)
		return buf
	},
}

// HTMLEscape performs HTML escaping on JSON string.
// Compatible with encoding/json: escapes <, >, &, U+2028, U+2029.
//
// This is the centralized implementation used by:
//   - json.HTMLEscape() for encoding/json compatibility
//   - json.HTMLEscapeBuffer() for buffer operations
//   - Any other internal components needing HTML escaping
//
// PERFORMANCE v2: Uses pooled buffer and byte-level scanning for speed.
func HTMLEscape(s string) string {
	// Fast path: byte-level check for escape characters
	// ASCII-only check is safe because all HTML-escape characters (<, >, &) are single-byte
	needsEscape := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '<' || c == '>' || c == '&' {
			needsEscape = true
			break
		}
		// Check for U+2028 (E2 80 A8) or U+2029 (E2 80 A9) UTF-8 sequences
		if c == 0xE2 && i+2 < len(s) && s[i+1] == 0x80 && (s[i+2] == 0xA8 || s[i+2] == 0xA9) {
			needsEscape = true
			break
		}
	}
	if !needsEscape {
		return s // No allocation for strings without special characters
	}

	// Slow path: perform escaping with pooled buffer
	buf := htmlEscapeBufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	// Grow buffer with estimate: original + 10% for escapes
	if buf.Cap() < len(s)+len(s)/10 {
		buf.Grow(len(s) + len(s)/10 - buf.Cap())
	}

	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '<':
			buf.WriteString("\\u003c")
		case '>':
			buf.WriteString("\\u003e")
		case '&':
			buf.WriteString("\\u0026")
		case 0xE2:
			// Check for U+2028 or U+2029
			if i+2 < len(s) && s[i+1] == 0x80 {
				if s[i+2] == 0xA8 {
					buf.WriteString("\\u2028")
					i += 2
					continue
				} else if s[i+2] == 0xA9 {
					buf.WriteString("\\u2029")
					i += 2
					continue
				}
			}
			buf.WriteByte(c)
		default:
			buf.WriteByte(c)
		}
	}

	result := buf.String()

	// Return buffer to pool if not too large
	if buf.Cap() <= 8*1024 {
		htmlEscapeBufferPool.Put(buf)
	}

	return result
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
// PERFORMANCE: Fast byte-level check to avoid allocation when no escaping is needed.
// ASCII-only check is safe because all HTML-escape characters (<, >, &) are single-byte ASCII.
func NeedsHTMLEscape(s string) bool {
	// Fast path: byte-level check for ASCII characters (<, >, &)
	// This is much faster than range-over-rune for ASCII-heavy strings
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '<' || c == '>' || c == '&' {
			return true
		}
		// Check for U+2028 (E2 80 A8) or U+2029 (E2 80 A9) UTF-8 sequences
		if c == 0xE2 && i+2 < len(s) && s[i+1] == 0x80 && (s[i+2] == 0xA8 || s[i+2] == 0xA9) {
			return true
		}
	}
	return false
}
