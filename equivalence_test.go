package json

import (
	"testing"
)

// TestP001_EncodePathsEquivalence guards the P-001 single fast-path claim:
// with default config, EncodeWithConfig (bytes fast path) and Marshal must
// produce byte-identical, HTML-escaped output for the values FastEncoder
// handles — including single-key maps, the forEachSortedEntry fast case.
func TestP001_EncodePathsEquivalence(t *testing.T) {
	p, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	values := []any{
		map[string]any{"name": "updated"},
		map[string]any{"name": "test", "age": 30, "active": true},
		map[string]any{"html": "<script>&amp;</script>"},
		[]any{1, 2, 3, "x", true},
		"a \"quoted\" <string>",
		42,
		3.14,
		true,
		nil,
		map[string]int{"b": 2, "a": 1},
		map[string]string{"z": "<z>", "a": "a"},
		[]string{"<1>", "2"},
	}

	for _, v := range values {
		enc, err1 := p.EncodeWithConfig(v)
		mar, err2 := p.Marshal(v)
		if (err1 != nil) != (err2 != nil) {
			t.Errorf("value %#v: error mismatch: %v vs %v", v, err1, err2)
			continue
		}
		if err1 != nil {
			continue
		}
		if enc != string(mar) {
			t.Errorf("value %#v:\n  EncodeWithConfig: %s\n  Marshal:          %s", v, enc, string(mar))
		}
	}
}
