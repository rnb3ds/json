package json

import (
	"reflect"
	"testing"
)

// TestBareIndexMatchesBracket is the core invariant of the bare-index feature:
// for every supported array path, the bare form ("0", "-1", "*", "*.prop")
// returns the same result as the bracket form ("[0]", "[-1]", "[*]", "[*].prop").
func TestBareIndexMatchesBracket(t *testing.T) {
	intArr := `[10,20,30]`
	objArr := `[{"name_cn":"万国数据","symbol":"GDS.US"},{"name_cn":"极氪","symbol":"ZK.US"}]`

	cases := []struct {
		name string
		json string
		bare string
		brkt string
	}{
		{"int 0", intArr, "0", "[0]"},
		{"int -1", intArr, "-1", "[-1]"},
		{"int *", intArr, "*", "[*]"},
		{"object 0", objArr, "0", "[0]"},
		{"object 0.symbol", objArr, "0.symbol", "[0].symbol"},
		{"object -1.symbol", objArr, "-1.symbol", "[-1].symbol"},
		{"object *.symbol", objArr, "*.symbol", "[*].symbol"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			bare, errB := Get(c.json, c.bare)
			brkt, errK := Get(c.json, c.brkt)
			if (errB != nil) != (errK != nil) {
				t.Errorf("errors differ: bare %q err=%v, bracket %q err=%v", c.bare, errB, c.brkt, errK)
			}
			if !reflect.DeepEqual(bare, brkt) {
				t.Errorf("bare %q = %#v, bracket %q = %#v", c.bare, bare, c.brkt, brkt)
			}
		})
	}
}

// TestBareIndexObjectKeyCompat verifies backward compatibility for objects
// that use numeric-string keys (user chose "array-first, object numeric-key
// compat"): the value is reachable via the bare numeric path AND via "[N]".
func TestBareIndexObjectKeyCompat(t *testing.T) {
	cases := []struct {
		name string
		json string
		path string
		want any
	}{
		{`bare "0" on {"0":"x"}`, `{"0":"x"}`, "0", "x"},
		{`bracket "[0]" on {"0":"x"}`, `{"0":"x"}`, "[0]", "x"},
		{`recurse "0.a"`, `{"0":{"a":1}}`, "0.a", float64(1)},
		{`bare "-1" on {"-1":"neg"}`, `{"-1":"neg"}`, "-1", "neg"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := Get(c.json, c.path)
			if err != nil {
				t.Fatalf("Get(%q) error: %v", c.path, err)
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("Get(%q) = %#v, want %#v", c.path, got, c.want)
			}
		})
	}
}

// TestBareIndexSetGuard verifies Set with a bare numeric path is unchanged.
func TestBareIndexSetGuard(t *testing.T) {
	out, err := Set(`[10,20,30]`, "0", 99)
	if err != nil {
		t.Fatalf("Set error: %v", err)
	}
	if out != `[99,20,30]` {
		t.Errorf(`Set([10,20,30], "0", 99) = %s, want [99,20,30]`, out)
	}
}

// TestBareIndexUserScenario is the exact case from the bug report: getting an
// array element via a bare numeric index.
func TestBareIndexUserScenario(t *testing.T) {
	jsonStr := `[` +
		`{"name_cn":"万国数据","name_en":"GDS Holdings Limited","name_hk":"万国数据","symbol":"GDS.US"},` +
		`{"name_cn":"极氪","name_en":"ZEEKR Intelligent Technology Holding Limited","name_hk":"極氪","symbol":"ZK.US"}` +
		`]`

	if got := GetString(jsonStr, "0.symbol"); got != "GDS.US" {
		t.Errorf(`GetString(_, "0.symbol") = %q, want "GDS.US"`, got)
	}
	if got := GetString(jsonStr, "1.symbol"); got != "ZK.US" {
		t.Errorf(`GetString(_, "1.symbol") = %q, want "ZK.US"`, got)
	}
	if got := GetArray(jsonStr, "*.symbol"); !reflect.DeepEqual(got, []any{"GDS.US", "ZK.US"}) {
		t.Errorf(`GetArray(_, "*.symbol") = %#v, want ["GDS.US","ZK.US"]`, got)
	}
}
