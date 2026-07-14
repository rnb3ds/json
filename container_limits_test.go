package json

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// makeObject builds a flat JSON object with n keys: {"k0":0,...,"k{n-1}":n-1}.
func makeObject(n int) string {
	var b strings.Builder
	b.WriteByte('{')
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, `"k%d":%d`, i, i)
	}
	b.WriteByte('}')
	return b.String()
}

// makeArray builds a flat JSON array with n elements: [0,1,...,n-1].
func makeArray(n int) string {
	var b strings.Builder
	b.WriteByte('[')
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "%d", i)
	}
	b.WriteByte(']')
	return b.String()
}

// TestValidateContainerCounts_Algorithm exercises the structural scanner
// directly with small limits, covering the tricky cases: per-container (not
// total) counting, nesting, empty containers, and structural characters that
// appear inside string values (which must be ignored).
func TestValidateContainerCounts_Algorithm(t *testing.T) {
	sv := func(maxKeys, maxElems int) *securityValidator {
		return &securityValidator{maxObjectKeys: maxKeys, maxArrayElements: maxElems}
	}

	tests := []struct {
		name      string
		json      string
		maxKeys   int
		maxElems  int
		wantError bool
	}{
		// Object key counting — strict greater-than at the limit.
		{"object at limit", `{"a":1,"b":2}`, 2, 0, false},
		{"object over limit", `{"a":1,"b":2,"c":3}`, 2, 0, true},
		{"object under limit", `{"a":1}`, 2, 0, false},

		// Array element counting.
		{"array at limit", `[1,2]`, 0, 2, false},
		{"array over limit", `[1,2,3]`, 0, 2, true},

		// Empty containers.
		{"empty object", `{}`, 1, 1, false},
		{"empty array", `[]`, 1, 1, false},

		// Structural characters inside string values must NOT be counted.
		{"braces in string value", `{"a":"{}","b":2}`, 2, 0, false},
		{"commas/colons in string value", `{"a":",:","b":2}`, 2, 0, false},
		{"brackets in string value", `["[]","{}"]`, 0, 2, false},
		{"escaped quote in string value", `{"a":"he said \"hi\"","b":2}`, 2, 0, false},

		// Counting is per-container, not total: each nested container is judged
		// independently against the same limit.
		{"nested object over limit", `{"outer":{"x":1,"y":2,"z":3}}`, 2, 0, true},
		{"nested object within limit", `{"outer":{"x":1,"y":2},"z":3}`, 2, 0, false},
		{"nested array over limit", `[[1,2,3]]`, 0, 2, true},
		{"array of small objects", `[{"a":1},{"b":2},{"c":3}]`, 0, 2, true},

		// Mixed structures.
		{"mixed object over array limit", `{"a":[1,2,3]}`, 0, 2, true},
		{"mixed within limits", `{"a":[1,2],"b":{"x":1}}`, 2, 2, false},

		// Whitespace is structural, not a value.
		{"whitespace handled", `{ "a" : 1 , "b" : 2 }`, 2, 0, false},

		// Bare (non-container) values: nothing to count.
		{"bare number", `42`, 1, 1, false},
		{"bare string", `"hello"`, 1, 1, false},
		{"bare null", `null`, 1, 1, false},
		{"bare bool", `true`, 1, 1, false},
		{"negative numbers", `[-1,-2,-3]`, 0, 2, true},

		// Unlimited (<=0) disables enforcement entirely.
		{"unlimited skips check", makeObject(50), 0, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sv(tt.maxKeys, tt.maxElems).validateContainerCounts(tt.json)
			if tt.wantError {
				if !errors.Is(err, ErrSizeLimit) {
					t.Fatalf("expected ErrSizeLimit, got: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
		})
	}
}

// TestValidateContainerCounts_PublicAPI verifies the limits are honored through
// the public Processor API. Config validation clamps MaxObjectKeys /
// MaxArrayElements to a minimum of 100, so the boundary is tested at 100/101.
func TestValidateContainerCounts_PublicAPI(t *testing.T) {
	t.Run("object keys", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.MaxObjectKeys = 100 // survives clamping (min 100)
		p, err := New(cfg)
		if err != nil {
			t.Fatalf("New failed: %v", err)
		}
		defer p.Close()

		// At the limit: allowed.
		if _, err := p.Get(makeObject(100), "k0"); err != nil {
			t.Fatalf("object with 100 keys (at limit) should pass, got: %v", err)
		}
		// Over the limit: rejected before any path processing.
		_, err = p.Get(makeObject(101), "k0")
		if !errors.Is(err, ErrSizeLimit) {
			t.Fatalf("object with 101 keys should be rejected with ErrSizeLimit, got: %v", err)
		}
	})

	t.Run("array elements", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.MaxArrayElements = 100
		p, err := New(cfg)
		if err != nil {
			t.Fatalf("New failed: %v", err)
		}
		defer p.Close()

		// At the limit: allowed.
		var out []any
		if err := p.Parse(makeArray(100), &out); err != nil {
			t.Fatalf("array with 100 elements (at limit) should pass, got: %v", err)
		}
		// Over the limit: rejected.
		err = p.Parse(makeArray(101), &out)
		if !errors.Is(err, ErrSizeLimit) {
			t.Fatalf("array with 101 elements should be rejected with ErrSizeLimit, got: %v", err)
		}
	})

	t.Run("nested object still caught", func(t *testing.T) {
		// A wide nested object at depth 1 must be rejected even though the root
		// has few keys — this is the exact attack the limit exists to stop.
		cfg := DefaultConfig()
		cfg.MaxObjectKeys = 100
		p, err := New(cfg)
		if err != nil {
			t.Fatalf("New failed: %v", err)
		}
		defer p.Close()

		nested := `{"wrapper":` + makeObject(101) + `}`
		_, err = p.Get(nested, "wrapper")
		if !errors.Is(err, ErrSizeLimit) {
			t.Fatalf("nested object with 101 keys should be rejected with ErrSizeLimit, got: %v", err)
		}
	})
}
