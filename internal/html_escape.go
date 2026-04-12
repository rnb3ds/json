package internal

import (
	"bytes"
	"sync"
)

// needsHTMLEscapeTable is a pre-computed lookup table for bytes that need HTML escaping.
// Index is the byte value. Set for '<' (0x3C), '>' (0x3E), '&' (0x26), and 0xE2 (U+2028/U+2029 prefix).
// PERFORMANCE: Replaces 24 per-byte comparisons with a single table lookup per byte.
var needsHTMLEscapeTable [256]bool

func init() {
	needsHTMLEscapeTable['<'] = true
	needsHTMLEscapeTable['>'] = true
	needsHTMLEscapeTable['&'] = true
	needsHTMLEscapeTable[0xE2] = true // start of U+2028/U+2029 UTF-8 sequence
}

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
//   - json.HTMLEscape() for encoding/json compatibility and buffer operations
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

// NeedsHTMLEscapeBytes checks if a byte slice needs HTML escaping.
// PERFORMANCE v3: Uses lookup table for single table-access per byte instead of 24 comparisons.
// The table checks for '<', '>', '&', and 0xE2 (start of U+2028/U+2029).
func NeedsHTMLEscapeBytes(data []byte) bool {
	n := len(data)
	if n == 0 {
		return false
	}

	// Main loop: table lookup is faster than chained OR comparisons
	// The CPU can speculate and branch-predict table lookups efficiently
	for i := 0; i < n; i++ {
		if needsHTMLEscapeTable[data[i]] {
			// For <, >, & — confirmed escape needed
			c := data[i]
			if c == '<' || c == '>' || c == '&' {
				return true
			}
			// For 0xE2 — check for U+2028/U+2029
			if c == 0xE2 && i+2 < n && data[i+1] == 0x80 && (data[i+2] == 0xA8 || data[i+2] == 0xA9) {
				return true
			}
		}
	}
	return false
}

// NeedsHTMLEscape checks if a string needs HTML escaping.
// PERFORMANCE v3: Uses lookup table — same optimization as NeedsHTMLEscapeBytes.
func NeedsHTMLEscape(s string) bool {
	n := len(s)
	if n == 0 {
		return false
	}

	for i := 0; i < n; i++ {
		if needsHTMLEscapeTable[s[i]] {
			c := s[i]
			if c == '<' || c == '>' || c == '&' {
				return true
			}
			if c == 0xE2 && i+2 < n && s[i+1] == 0x80 && (s[i+2] == 0xA8 || s[i+2] == 0xA9) {
				return true
			}
		}
	}
	return false
}

// htmlEscapeBytesPool pools byte slices for HTML escaping operations
var htmlEscapeBytesPool = sync.Pool{
	New: func() any {
		b := make([]byte, 0, 256)
		return &b
	},
}

// HTMLEscapeBytes performs HTML escaping on a byte slice and returns the escaped result.
// PERFORMANCE v2: Works directly on byte slices to avoid string conversion.
// This is the most efficient path for encoder output.
func HTMLEscapeBytes(data []byte) []byte {
	n := len(data)
	if n == 0 {
		return data
	}

	// Fast path: check if escaping is needed using SWAR
	if !NeedsHTMLEscapeBytes(data) {
		return data
	}

	// Slow path: perform escaping
	// Estimate: worst case each char becomes 6 chars (\u003c)
	result := htmlEscapeBytesPool.Get().(*[]byte)
	*result = (*result)[:0]
	if cap(*result) < n*2 {
		*result = make([]byte, 0, n*2)
	}

	for i := 0; i < n; i++ {
		c := data[i]
		switch c {
		case '<':
			*result = append(*result, '\\', 'u', '0', '0', '3', 'c')
		case '>':
			*result = append(*result, '\\', 'u', '0', '0', '3', 'e')
		case '&':
			*result = append(*result, '\\', 'u', '0', '0', '2', '6')
		case 0xE2:
			// Check for U+2028 or U+2029
			if i+2 < n && data[i+1] == 0x80 {
				if data[i+2] == 0xA8 {
					*result = append(*result, '\\', 'u', '2', '0', '2', '8')
					i += 2
					continue
				} else if data[i+2] == 0xA9 {
					*result = append(*result, '\\', 'u', '2', '0', '2', '9')
					i += 2
					continue
				}
			}
			*result = append(*result, c)
		default:
			*result = append(*result, c)
		}
	}

	escaped := *result

	// Don't pool large buffers
	if cap(*result) > 8*1024 {
		// Return a copy to avoid pool retention
		copied := make([]byte, len(escaped))
		copy(copied, escaped)
		return copied
	}

	return escaped
}

// PutHTMLEscapeBytes returns a byte slice obtained from HTMLEscapeBytes to the pool.
// Call this after using the result if you don't need it anymore.
func PutHTMLEscapeBytes(b []byte) {
	if cap(b) <= 8*1024 && cap(b) > 0 {
		// Only pool reasonably sized buffers
		htmlEscapeBytesPool.Put(&b)
	}
}
