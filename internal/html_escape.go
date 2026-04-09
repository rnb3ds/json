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
// PERFORMANCE v2: Uses SWAR (SIMD Within A Register) for 8-byte batch processing.
// This is the byte-slice version to avoid string conversion.
func NeedsHTMLEscapeBytes(data []byte) bool {
	n := len(data)
	if n == 0 {
		return false
	}

	// SWAR: Process 8 bytes at a time
	for i := 0; i+8 <= n; i += 8 {
		b0, b1, b2, b3 := data[i], data[i+1], data[i+2], data[i+3]
		b4, b5, b6, b7 := data[i+4], data[i+5], data[i+6], data[i+7]

		// Quick check: if any byte matches our target characters
		if b0 == '<' || b1 == '<' || b2 == '<' || b3 == '<' ||
			b4 == '<' || b5 == '<' || b6 == '<' || b7 == '<' ||
			b0 == '>' || b1 == '>' || b2 == '>' || b3 == '>' ||
			b4 == '>' || b5 == '>' || b6 == '>' || b7 == '>' ||
			b0 == '&' || b1 == '&' || b2 == '&' || b3 == '&' ||
			b4 == '&' || b5 == '&' || b6 == '&' || b7 == '&' {
			return true
		}

		// Check for 0xE2 (start of U+2028/U+2029 UTF-8 sequence)
		if b0 == 0xE2 || b1 == 0xE2 || b2 == 0xE2 || b3 == 0xE2 ||
			b4 == 0xE2 || b5 == 0xE2 || b6 == 0xE2 || b7 == 0xE2 {
			for j := i; j < i+8 && j+2 < n; j++ {
				if data[j] == 0xE2 && data[j+1] == 0x80 && (data[j+2] == 0xA8 || data[j+2] == 0xA9) {
					return true
				}
			}
		}
	}

	// Check remaining bytes
	for i := n &^ 7; i < n; i++ {
		c := data[i]
		if c == '<' || c == '>' || c == '&' {
			return true
		}
		if c == 0xE2 && i+2 < n && data[i+1] == 0x80 && (data[i+2] == 0xA8 || data[i+2] == 0xA9) {
			return true
		}
	}
	return false
}

// NeedsHTMLEscape checks if a string needs HTML escaping.
// PERFORMANCE v2: Uses SWAR (SIMD Within A Register) for 8-byte batch processing.
// ASCII-only check is safe because all HTML-escape characters (<, >, &) are single-byte ASCII.
func NeedsHTMLEscape(s string) bool {
	n := len(s)
	if n == 0 {
		return false
	}

	// SWAR: Process 8 bytes at a time
	// Check for '<' (0x3C), '>' (0x3E), '&' (0x26), and 0xE2 (potential U+2028/U+2029)
	for i := 0; i+8 <= n; i += 8 {
		b0, b1, b2, b3 := s[i], s[i+1], s[i+2], s[i+3]
		b4, b5, b6, b7 := s[i+4], s[i+5], s[i+6], s[i+7]

		// Quick check: if any byte matches our target characters
		if b0 == '<' || b1 == '<' || b2 == '<' || b3 == '<' ||
			b4 == '<' || b5 == '<' || b6 == '<' || b7 == '<' ||
			b0 == '>' || b1 == '>' || b2 == '>' || b3 == '>' ||
			b4 == '>' || b5 == '>' || b6 == '>' || b7 == '>' ||
			b0 == '&' || b1 == '&' || b2 == '&' || b3 == '&' ||
			b4 == '&' || b5 == '&' || b6 == '&' || b7 == '&' {
			return true
		}

		// Check for 0xE2 (start of U+2028/U+2029 UTF-8 sequence)
		if b0 == 0xE2 || b1 == 0xE2 || b2 == 0xE2 || b3 == 0xE2 ||
			b4 == 0xE2 || b5 == 0xE2 || b6 == 0xE2 || b7 == 0xE2 {
			// Fall back to precise check for U+2028/U+2029
			for j := i; j < i+8 && j+2 < n; j++ {
				if s[j] == 0xE2 && s[j+1] == 0x80 && (s[j+2] == 0xA8 || s[j+2] == 0xA9) {
					return true
				}
			}
		}
	}

	// Check remaining bytes (less than 8)
	for i := n &^ 7; i < n; i++ {
		c := s[i]
		if c == '<' || c == '>' || c == '&' {
			return true
		}
		if c == 0xE2 && i+2 < n && s[i+1] == 0x80 && (s[i+2] == 0xA8 || s[i+2] == 0xA9) {
			return true
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
