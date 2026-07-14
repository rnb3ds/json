package internal

import (
	"strings"
	"testing"
)

// ============================================================================
// Boundary tests for internal/hash.go low-coverage paths.
// ============================================================================

// --- HashBytesFNV1aOffset (hash.go:196, 0% coverage) ---

func TestHashBytesFNV1aOffset(t *testing.T) {
	t.Run("nonempty", func(t *testing.T) {
		got := HashBytesFNV1aOffset([]byte("test"))
		if got == 0 {
			t.Error("expected non-zero hash for non-empty input")
		}
		// Different offset basis must yield a different hash than the primary.
		if got == HashBytesFNV1a([]byte("test")) {
			t.Error("offset-basis hash should differ from primary FNV1a hash")
		}
	})
	t.Run("length_not_multiple_of_8", func(t *testing.T) {
		// 13 bytes exercises both the 8-byte batch loop and the remainder loop.
		got := HashBytesFNV1aOffset([]byte("thirteen_byte"))
		if got == 0 {
			t.Error("expected non-zero hash")
		}
	})
	t.Run("deterministic", func(t *testing.T) {
		a := HashBytesFNV1aOffset([]byte("same"))
		b := HashBytesFNV1aOffset([]byte("same"))
		if a != b {
			t.Errorf("hash not deterministic: %d vs %d", a, b)
		}
	})
}

// --- HashStringFNV1aSampled (hash.go:96, 76% coverage) ---

func TestHashStringFNV1aSampled_LargeString(t *testing.T) {
	t.Run("large_string_sampling_path", func(t *testing.T) {
		// Above LargeStringHashThreshold -> exercises the first/middle/last
		// sampling branches instead of the full-hash fast path.
		s := strings.Repeat("a", 8192)
		got := HashStringFNV1aSampled(s)
		if got == 0 {
			t.Error("expected non-zero sampled hash")
		}
		// Deterministic across calls.
		if got != HashStringFNV1aSampled(s) {
			t.Error("sampled hash not deterministic")
		}
	})
	t.Run("distinguishes_lengths", func(t *testing.T) {
		// Length is mixed into the hash, so different lengths must not collide.
		a := HashStringFNV1aSampled(strings.Repeat("a", 8192))
		b := HashStringFNV1aSampled(strings.Repeat("a", 8193))
		if a == b {
			t.Error("expected different hashes for different lengths")
		}
	})
}
