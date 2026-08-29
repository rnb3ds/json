package json

import (
	"encoding/json"
	"math"
	"testing"
	"time"
)

// TestA2EncoderEquivalence proves Marshal/MarshalIndent and EncodeWithConfig
// produce identical bytes for the configurations the old MarshalToFile
// pipeline used. MarshalToFile and SaveToFile now share one pipeline
// (writeFileJSON → EncodeWithConfig), so this equivalence is what guarantees
// the unification did not change output for previously-supported inputs —
// and it guards against the two encoders drifting apart again.
func TestA2EncoderEquivalence(t *testing.T) {
	ls := "x" + string(rune(0x2028)) + "y"
	inv := string([]byte{'a', 0xff, 'b'})
	type inner struct {
		B string `json:"b"`
	}
	type outer struct {
		Name string          `json:"name"`
		N    int             `json:"n"`
		F    float64         `json:"f"`
		Arr  []int           `json:"arr"`
		Obj  inner           `json:"obj"`
		Ptr  *int            `json:"ptr"`
		Raw  json.RawMessage `json:"raw"`
		Num  json.Number     `json:"num"`
		T    time.Time       `json:"t"`
		Skip string          `json:"-"`
		Opt  string          `json:"opt,omitempty"`
	}
	seven := 7
	values := []any{
		nil, true, 0, -1, 42, 1e21, math.Copysign(0, -1), 0.1, 1e-7, math.MaxInt64,
		"", "plain", "<script>&</script>", ls, inv, "emoji:" + string(rune(0x1F600)),
		[]any{}, map[string]any{}, []any(nil), map[string]any(nil),
		[]int{1, 2, 3}, map[string]any{"a": 1, "b": []any{"x", "y"}},
		map[string]any{"deep": map[string]any{"deeper": []any{map[string]any{"k": "<v>"}}}},
		[]byte("binary<h>"),
		json.RawMessage(`{"raw":[1,2]}`),
		json.Number("1.2300"),
		time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC),
		outer{Name: "n", N: 9, F: 3.5, Arr: []int{1}, Obj: inner{B: "<b>"}, Ptr: &seven,
			Raw: json.RawMessage(`[3]`), Num: json.Number("1e2"), Skip: "x"},
		&outer{Name: "ptr"},
	}

	p, _ := New()
	defer p.Close()

	compactCfg := DefaultConfig() // Pretty=false — 旧管线等价 cfg
	prettyCfg := PrettyConfig()   // Pretty=true, Indent="  " — 旧 MarshalIndent 参数

	for i, v := range values {
		got, err1 := p.EncodeWithConfig(v, compactCfg)
		want, err2 := p.Marshal(v)
		if (err1 == nil) != (err2 == nil) {
			t.Errorf("compact[%d] err mismatch: %v vs %v", i, err1, err2)
			continue
		}
		if err1 == nil && string(got) != string(want) {
			t.Errorf("compact[%d]: EncodeWithConfig=%s Marshal=%s", i, got, want)
		}

		gotP, err3 := p.EncodeWithConfig(v, prettyCfg)
		wantP, err4 := p.MarshalIndent(v, "", "  ")
		if (err3 == nil) != (err4 == nil) {
			t.Errorf("pretty[%d] err mismatch: %v vs %v", i, err3, err4)
			continue
		}
		if err3 == nil && string(gotP) != string(wantP) {
			t.Errorf("pretty[%d]: EncodeWithConfig=%s | MarshalIndent=%s", i, gotP, wantP)
		}
	}
}
