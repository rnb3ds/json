package json

import (
	"testing"
)

// Helper function to compare numeric values (handles various number types)
func numericEqual(got any, expected float64) bool {
	switch v := got.(type) {
	case float64:
		return v == expected
	case float32:
		return float64(v) == expected
	case int:
		return float64(v) == expected
	case int64:
		return float64(v) == expected
	case int32:
		return float64(v) == expected
	case Number:
		f, err := v.Float64()
		if err != nil {
			return false
		}
		return f == expected
	}
	return false
}

// parseResult parses JSON string to map using the library's decoder
func parseResult(t *testing.T, result string) map[string]any {
	t.Helper()
	decoder := newNumberPreservingDecoder(true)
	data, err := decoder.DecodeToAny(result)
	if err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}
	resultMap, ok := data.(map[string]any)
	if !ok {
		t.Fatalf("result is not an object, got %T", data)
	}
	return resultMap
}

// ============================================================================
// MergeJSON TESTS
// ============================================================================

func TestMergeJSON_Union(t *testing.T) {
	base := `{"a": 1, "b": 2}`
	override := `{"b": 3, "c": 4}`

	result, err := MergeJSON(base, override)
	if err != nil {
		t.Fatalf("MergeJSON() error: %v", err)
	}

	resultMap := parseResult(t, result)

	// Union should have all keys
	if !numericEqual(resultMap["a"], 1) {
		t.Errorf("expected a=1, got %v (type %T)", resultMap["a"], resultMap["a"])
	}
	if !numericEqual(resultMap["b"], 3) {
		t.Errorf("expected b=3 (overridden), got %v (type %T)", resultMap["b"], resultMap["b"])
	}
	if !numericEqual(resultMap["c"], 4) {
		t.Errorf("expected c=4, got %v (type %T)", resultMap["c"], resultMap["c"])
	}
}

func TestMergeJSON_NestedObjects(t *testing.T) {
	base := `{"db": {"host": "localhost", "port": 3306}}`
	override := `{"db": {"port": 5432, "ssl": true}}`

	result, err := MergeJSON(base, override)
	if err != nil {
		t.Fatalf("MergeJSON() error: %v", err)
	}

	resultMap := parseResult(t, result)

	db := resultMap["db"].(map[string]any)
	if db["host"].(string) != "localhost" {
		t.Errorf("expected host=localhost, got %v", db["host"])
	}
	if !numericEqual(db["port"], 5432) {
		t.Errorf("expected port=5432 (overridden), got %v", db["port"])
	}
	if db["ssl"].(bool) != true {
		t.Errorf("expected ssl=true, got %v", db["ssl"])
	}
}

func TestMergeJSON_InvalidJSON(t *testing.T) {
	tests := []struct {
		name     string
		base     string
		override string
		wantErr  bool
	}{
		{"invalid base", `{invalid}`, `{"a":1}`, true},
		{"invalid override", `{"a":1}`, `{invalid}`, true},
		{"both invalid", `{invalid}`, `{invalid}`, true},
		{"both valid", `{"a":1}`, `{"b":2}`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := MergeJSON(tt.base, tt.override)
			if (err != nil) != tt.wantErr {
				t.Errorf("MergeJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMergeJSON_NonObject(t *testing.T) {
	tests := []struct {
		name     string
		base     string
		override string
		wantErr  bool
	}{
		{"base is array", `[1,2]`, `{"a":1}`, true},
		{"override is array", `{"a":1}`, `[1,2]`, true},
		{"both arrays", `[1,2]`, `[3,4]`, true},
		{"base is primitive", `"hello"`, `{"a":1}`, true},
		{"override is primitive", `{"a":1}`, `"hello"`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := MergeJSON(tt.base, tt.override)
			if (err != nil) != tt.wantErr {
				t.Errorf("MergeJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// ============================================================================
// MergeJSON with Mode TESTS
// ============================================================================

func TestMergeJSON_UnionMode(t *testing.T) {
	base := `{"a": 1, "b": 2}`
	override := `{"b": 3, "c": 4}`

	result, err := MergeJSON(base, override, MergeUnion)
	if err != nil {
		t.Fatalf("MergeJSON() error: %v", err)
	}

	resultMap := parseResult(t, result)

	// Union: all keys, overridden values
	if len(resultMap) != 3 {
		t.Errorf("expected 3 keys, got %d", len(resultMap))
	}
	if !numericEqual(resultMap["b"], 3) {
		t.Errorf("expected b=3, got %v", resultMap["b"])
	}
}

func TestMergeJSON_IntersectionMode(t *testing.T) {
	tests := []struct {
		name         string
		base         string
		override     string
		expectedKeys []string
		checkValues  map[string]float64
	}{
		{
			name:         "no common keys",
			base:         `{"a": 1}`,
			override:     `{"b": 2}`,
			expectedKeys: []string{},
			checkValues:  map[string]float64{},
		},
		{
			name:         "some common keys",
			base:         `{"a": 1, "b": 2, "c": 3}`,
			override:     `{"b": 20, "c": 30, "d": 40}`,
			expectedKeys: []string{"b", "c"},
			checkValues:  map[string]float64{"b": 20, "c": 30},
		},
		{
			name:         "all common keys",
			base:         `{"a": 1, "b": 2}`,
			override:     `{"a": 10, "b": 20}`,
			expectedKeys: []string{"a", "b"},
			checkValues:  map[string]float64{"a": 10, "b": 20},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := MergeJSON(tt.base, tt.override, MergeIntersection)
			if err != nil {
				t.Fatalf("MergeJSON() error: %v", err)
			}

			resultMap := parseResult(t, result)

			// Compare lengths first
			if len(resultMap) != len(tt.expectedKeys) {
				t.Errorf("expected %d keys, got %d", len(tt.expectedKeys), len(resultMap))
				return
			}

			// Compare values
			for k, v := range tt.checkValues {
				if !numericEqual(resultMap[k], v) {
					t.Errorf("key %s: expected %v, got %v", k, v, resultMap[k])
				}
			}
		})
	}
}

func TestMergeJSON_DifferenceMode(t *testing.T) {
	tests := []struct {
		name         string
		base         string
		override     string
		expectedKeys []string
		checkValues  map[string]float64
	}{
		{
			name:         "no overlap",
			base:         `{"a": 1, "b": 2}`,
			override:     `{"c": 3, "d": 4}`,
			expectedKeys: []string{"a", "b"},
			checkValues:  map[string]float64{"a": 1, "b": 2},
		},
		{
			name:         "some overlap",
			base:         `{"a": 1, "b": 2, "c": 3}`,
			override:     `{"b": 2, "d": 4}`,
			expectedKeys: []string{"a", "c"},
			checkValues:  map[string]float64{"a": 1, "c": 3},
		},
		{
			name:         "full overlap",
			base:         `{"a": 1, "b": 2}`,
			override:     `{"a": 1, "b": 2}`,
			expectedKeys: []string{},
			checkValues:  map[string]float64{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := MergeJSON(tt.base, tt.override, MergeDifference)
			if err != nil {
				t.Fatalf("MergeJSON() error: %v", err)
			}

			resultMap := parseResult(t, result)

			// Compare lengths
			if len(resultMap) != len(tt.expectedKeys) {
				t.Errorf("expected %d keys, got %d: %v", len(tt.expectedKeys), len(resultMap), resultMap)
				return
			}

			// Compare values
			for k, v := range tt.checkValues {
				if !numericEqual(resultMap[k], v) {
					t.Errorf("key %s: expected %v, got %v", k, v, resultMap[k])
				}
			}
		})
	}
}

func TestMergeJSON_NestedIntersectionMode(t *testing.T) {
	base := `{
		"common": {"a": 1, "b": 2},
		"onlyA": 1
	}`
	override := `{
		"common": {"a": 10, "c": 3},
		"onlyB": 2
	}`

	result, err := MergeJSON(base, override, MergeIntersection)
	if err != nil {
		t.Fatalf("MergeJSON() error: %v", err)
	}

	resultMap := parseResult(t, result)

	// Only "common" should exist (it's the only common key at top level)
	if len(resultMap) != 1 {
		t.Errorf("expected 1 key, got %d: %v", len(resultMap), resultMap)
	}

	common := resultMap["common"].(map[string]any)
	// In nested object, only "a" is common
	if len(common) != 1 {
		t.Errorf("expected 1 key in common, got %d: %v", len(common), common)
	}
	if !numericEqual(common["a"], 10) {
		t.Errorf("expected common.a=10, got %v", common["a"])
	}
}

// ============================================================================
// MergeJSONMany TESTS
// ============================================================================

func TestMergeJSONMany_Union(t *testing.T) {
	json1 := `{"a": 1}`
	json2 := `{"b": 2}`
	json3 := `{"c": 3}`

	result, err := MergeJSONMany(MergeUnion, json1, json2, json3)
	if err != nil {
		t.Fatalf("MergeJSONMany() error: %v", err)
	}

	resultMap := parseResult(t, result)

	if len(resultMap) != 3 {
		t.Errorf("expected 3 keys, got %d", len(resultMap))
	}
}

func TestMergeJSONMany_WithOverride(t *testing.T) {
	json1 := `{"a": 1, "b": 1}`
	json2 := `{"b": 2}`
	json3 := `{"b": 3, "c": 3}`

	result, err := MergeJSONMany(MergeUnion, json1, json2, json3)
	if err != nil {
		t.Fatalf("MergeJSONMany() error: %v", err)
	}

	resultMap := parseResult(t, result)

	// b should be 3 (last override)
	if !numericEqual(resultMap["b"], 3) {
		t.Errorf("expected b=3, got %v", resultMap["b"])
	}
	// a should be preserved
	if !numericEqual(resultMap["a"], 1) {
		t.Errorf("expected a=1, got %v", resultMap["a"])
	}
	// c should be added
	if !numericEqual(resultMap["c"], 3) {
		t.Errorf("expected c=3, got %v", resultMap["c"])
	}
}

func TestMergeJSONMany_Intersection(t *testing.T) {
	json1 := `{"a": 1, "b": 2, "c": 3}`
	json2 := `{"b": 20, "c": 30, "d": 40}`
	json3 := `{"b": 200, "c": 300, "e": 500}`

	result, err := MergeJSONMany(MergeIntersection, json1, json2, json3)
	if err != nil {
		t.Fatalf("MergeJSONMany() error: %v", err)
	}

	resultMap := parseResult(t, result)

	// Only b and c are in all three
	if len(resultMap) != 2 {
		t.Errorf("expected 2 keys (b, c), got %d: %v", len(resultMap), resultMap)
	}
	if !numericEqual(resultMap["b"], 200) {
		t.Errorf("expected b=200, got %v", resultMap["b"])
	}
}

func TestMergeJSONMany_Difference(t *testing.T) {
	json1 := `{"a": 1, "b": 2, "c": 3}`
	json2 := `{"b": 2, "d": 4}`

	result, err := MergeJSONMany(MergeDifference, json1, json2)
	if err != nil {
		t.Fatalf("MergeJSONMany() error: %v", err)
	}

	resultMap := parseResult(t, result)

	// Only a and c are in json1 but not in json2
	if len(resultMap) != 2 {
		t.Errorf("expected 2 keys (a, c), got %d: %v", len(resultMap), resultMap)
	}
	if !numericEqual(resultMap["a"], 1) {
		t.Errorf("expected a=1, got %v", resultMap["a"])
	}
	if !numericEqual(resultMap["c"], 3) {
		t.Errorf("expected c=3, got %v", resultMap["c"])
	}
}

func TestMergeJSONMany_InsufficientArgs(t *testing.T) {
	_, err := MergeJSONMany(MergeUnion)
	if err == nil {
		t.Error("expected error for 0 arguments")
	}

	_, err = MergeJSONMany(MergeUnion, `{"a":1}`)
	if err == nil {
		t.Error("expected error for 1 argument")
	}
}

func TestMergeJSONMany_InvalidJSON(t *testing.T) {
	_, err := MergeJSONMany(MergeUnion, `{"a":1}`, `{invalid}`, `{"c":3}`)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

// ============================================================================
// MergeMode TESTS
// ============================================================================

func TestMergeMode_UnionDefault(t *testing.T) {
	// Verify MergeJSON uses union mode by default
	base := `{"a": 1, "b": 2}`
	override := `{"b": 3, "c": 4}`

	result, err := MergeJSON(base, override)
	if err != nil {
		t.Fatalf("MergeJSON() error: %v", err)
	}

	resultMap := parseResult(t, result)

	// Union should have all keys
	if len(resultMap) != 3 {
		t.Errorf("expected 3 keys for union, got %d", len(resultMap))
	}
}

// ============================================================================
// MergeJSONWithArrays TESTS
// ============================================================================

func TestMergeJSONWithArrays(t *testing.T) {
	tests := []struct {
		name     string
		base     string
		override string
		mode     MergeMode
		checkLen int
	}{
		{"union arrays", `{"arr": [1, 2]}`, `{"arr": [2, 3]}`, MergeUnion, 3},
		{"intersection arrays", `{"arr": [1, 2, 3]}`, `{"arr": [2, 3, 4]}`, MergeIntersection, 2},
		{"difference arrays", `{"arr": [1, 2, 3]}`, `{"arr": [2, 3, 4]}`, MergeDifference, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := MergeJSON(tt.base, tt.override, tt.mode)
			if err != nil {
				t.Fatalf("MergeJSON() error: %v", err)
			}

			resultMap := parseResult(t, result)

			arr := resultMap["arr"].([]any)
			if len(arr) != tt.checkLen {
				t.Errorf("expected array length %d, got %d: %v", tt.checkLen, len(arr), arr)
			}
		})
	}
}
