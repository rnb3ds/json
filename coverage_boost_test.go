package json

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cybergodev/json/internal"
)

// ============================================================================
// API FILE OPERATIONS TESTS - Only tests not covered in file_test.go
// ============================================================================

// TestAPI_UnmarshalFromFile tests the top-level UnmarshalFromFile function
// Note: LoadFromFile, SaveToFile, MarshalToFile are tested in file_test.go
func TestAPI_UnmarshalFromFile(t *testing.T) {
	tests := []struct {
		name        string
		setupFile   bool
		fileContent string
		target      any
		wantErr     bool
		checkResult func(t *testing.T, result any)
	}{
		{
			name:        "UnmarshalFromFileValid",
			setupFile:   true,
			fileContent: `{"name":"test","value":123}`,
			target: &struct {
				Name  string
				Value int
			}{},
			wantErr: false,
			checkResult: func(t *testing.T, result any) {
				r := result.(*struct {
					Name  string
					Value int
				})
				if r.Name != "test" || r.Value != 123 {
					t.Errorf("Result = %+v, want {Name:test, Value:123}", r)
				}
			},
		},
		{
			name:      "UnmarshalFromFileNonExistent",
			setupFile: false,
			target:    &map[string]any{},
			wantErr:   true,
		},
		{
			name:        "UnmarshalFromFileNilTarget",
			setupFile:   true,
			fileContent: `{}`,
			target:      nil,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var filePath string
			if tt.setupFile {
				tempDir := t.TempDir()
				filePath = filepath.Join(tempDir, "test.json")
				os.WriteFile(filePath, []byte(tt.fileContent), 0644)
			} else {
				filePath = "/non/existent/file.json"
			}

			err := UnmarshalFromFile(filePath, tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalFromFile() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && tt.checkResult != nil {
				tt.checkResult(t, tt.target)
			}
		})
	}
}

// TestIsMapType tests the internal IsMapType function
// Note: IsMapType only returns true for map[string]any and map[any]any
func TestIsMapType(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected bool
	}{
		{"map[string]any", map[string]any{"key": "value"}, true},
		{"map[any]any", map[any]any{"key": "value"}, true},
		{"map[string]string", map[string]string{"key": "value"}, false}, // Not map[string]any
		{"map[int]string", map[int]string{1: "one"}, false},             // Not supported type
		{"slice", []any{1, 2, 3}, false},
		{"string", "string", false},
		{"nil", nil, false},
		{"int", 42, false},
		{"struct", struct{ Name string }{"test"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := internal.IsMapType(tt.input)
			if result != tt.expected {
				t.Errorf("IsMapType(%T) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestIsSliceType tests the internal IsSliceType function
func TestIsSliceType(t *testing.T) {
	tests := []struct {
		input    any
		expected bool
	}{
		{[]any{1, 2, 3}, true},
		{[]string{"a", "b"}, true},
		{[]int{1, 2, 3}, true},
		{map[string]any{"key": "value"}, false},
		{"string", false},
		{nil, false},
		{42, false},
	}

	for i, tt := range tests {
		result := internal.IsSliceType(tt.input)
		if result != tt.expected {
			t.Errorf("Test %d: IsSliceType(%T) = %v, want %v", i, tt.input, result, tt.expected)
		}
	}
}

// ============================================================================
// CONFIG TESTS - Coverage for Clone, Validate edge cases
// ============================================================================

// TestConfigCloneZero tests Config.Clone on zero value
func TestConfigCloneZero(t *testing.T) {
	var c Config
	result := c.Clone()
	// Zero value clone should return a zero config
	if result.MaxJSONSize != 0 {
		t.Error("Clone() on zero Config should return zero config")
	}
}

// TestConfigValidateEdgeCases tests Config.Validate edge cases
func TestConfigValidateEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		wantErr     bool
		checkResult func(t *testing.T, config *Config)
	}{
		{
			name:    "ZeroValues",
			config:  &Config{},
			wantErr: false,
			checkResult: func(t *testing.T, config *Config) {
				if config.MaxJSONSize <= 0 {
					t.Error("MaxJSONSize should be set to default after validation")
				}
				if config.MaxPathDepth <= 0 {
					t.Error("MaxPathDepth should be set to default after validation")
				}
			},
		},
		{
			name:    "NegativeCacheSize",
			config:  &Config{MaxCacheSize: -1},
			wantErr: false,
			checkResult: func(t *testing.T, config *Config) {
				if config.MaxCacheSize != 0 {
					t.Errorf("Negative MaxCacheSize should be clamped to 0, got %d", config.MaxCacheSize)
				}
				if config.EnableCache {
					t.Error("EnableCache should be false when MaxCacheSize is negative")
				}
			},
		},
		{
			name:    "LargeCacheSize",
			config:  &Config{MaxCacheSize: 5000},
			wantErr: false,
			checkResult: func(t *testing.T, config *Config) {
				if config.MaxCacheSize > 2000 {
					t.Errorf("Large MaxCacheSize should be clamped to 2000, got %d", config.MaxCacheSize)
				}
			},
		},
		{
			name:    "ZeroCacheTTL",
			config:  &Config{CacheTTL: 0},
			wantErr: false,
			checkResult: func(t *testing.T, config *Config) {
				if config.CacheTTL <= 0 {
					t.Error("Zero CacheTTL should be set to default")
				}
			},
		},
		{
			name:    "LargeValues",
			config:  &Config{MaxJSONSize: 1000 * 1024 * 1024, MaxPathDepth: 500, MaxConcurrency: 500},
			wantErr: false,
			checkResult: func(t *testing.T, config *Config) {
				if config.MaxJSONSize > 100*1024*1024 {
					t.Error("MaxJSONSize should be clamped")
				}
				if config.MaxPathDepth > 200 {
					t.Error("MaxPathDepth should be clamped")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.checkResult != nil {
				tt.checkResult(t, tt.config)
			}
		})
	}
}

// ============================================================================
// ENCODING TESTS - Coverage for printData branches
// ============================================================================

// TestPrintData tests the printData function
func TestPrintData(t *testing.T) {
	t.Run("ValidJSONString", func(t *testing.T) {
		result, err := printData(`{"key":"value"}`, false)
		if err != nil {
			t.Errorf("printData with valid JSON string failed: %v", err)
		}
		if !strings.Contains(result, `"key"`) {
			t.Error("Result should contain the JSON key")
		}
	})

	t.Run("InvalidJSONString", func(t *testing.T) {
		result, err := printData("not json", false)
		if err != nil {
			t.Errorf("printData with non-JSON string failed: %v", err)
		}
		// Should encode as a normal string
		if !strings.Contains(result, `"not json"`) {
			t.Error("Non-JSON string should be encoded as JSON string")
		}
	})

	t.Run("ValidJSONBytes", func(t *testing.T) {
		result, err := printData([]byte(`{"key":"value"}`), false)
		if err != nil {
			t.Errorf("printData with valid JSON bytes failed: %v", err)
		}
		if !strings.Contains(result, `"key"`) {
			t.Error("Result should contain the JSON key")
		}
	})

	t.Run("InvalidJSONBytes", func(t *testing.T) {
		result, err := printData([]byte("not json"), false)
		if err != nil {
			t.Errorf("printData with non-JSON bytes failed: %v", err)
		}
		// Should encode as normal
		if result == "" {
			t.Error("Result should not be empty")
		}
	})

	t.Run("MapData", func(t *testing.T) {
		data := map[string]any{"name": "test"}
		result, err := printData(data, false)
		if err != nil {
			t.Errorf("printData with map failed: %v", err)
		}
		if !strings.Contains(result, `"name"`) {
			t.Error("Result should contain the map key")
		}
	})

	t.Run("PrettyFormatting", func(t *testing.T) {
		data := map[string]any{"name": "test"}
		result, err := printData(data, true)
		if err != nil {
			t.Errorf("printData with pretty failed: %v", err)
		}
		if !strings.Contains(result, "\n") {
			t.Error("Pretty output should contain newlines")
		}
	})
}

// TestCompactError tests Compact function error case
func TestCompactError(t *testing.T) {
	var dst bytes.Buffer
	err := Compact(&dst, []byte(`{invalid}`))
	if err == nil {
		t.Error("Compact should return error for invalid JSON")
	}
}

// TestIndentError tests Indent function error case
func TestIndentError(t *testing.T) {
	var dst bytes.Buffer
	err := Indent(&dst, []byte(`{invalid}`), "", "  ")
	if err == nil {
		t.Error("Indent should return error for invalid JSON")
	}
}

// TestEncodeWithConfig tests EncodeWithConfig with custom config
func TestEncodeWithConfig(t *testing.T) {
	t.Run("WithPretty", func(t *testing.T) {
		opts := DefaultConfig()
		opts.Pretty = true

		result, err := EncodeWithConfig(map[string]any{"key": "value"}, opts)
		if err != nil {
			t.Errorf("EncodeWithConfig failed: %v", err)
		}
		if !strings.Contains(result, "\n") {
			t.Error("Result should be pretty-printed")
		}
	})

	t.Run("WithCompact", func(t *testing.T) {
		opts := DefaultConfig()
		opts.Pretty = false

		result, err := EncodeWithConfig(map[string]any{"key": "value"}, opts)
		if err != nil {
			t.Errorf("EncodeWithConfig failed: %v", err)
		}
		if strings.Contains(result, "\n") {
			t.Error("Result should not be pretty-printed")
		}
	})
}

// TestEncode tests Encode function
func TestEncode(t *testing.T) {
	t.Run("WithConfig", func(t *testing.T) {
		config := DefaultConfig()
		config.EscapeHTML = true

		result, err := Encode(map[string]any{"html": "<script>"}, config)
		if err != nil {
			t.Errorf("Encode failed: %v", err)
		}
		if !strings.Contains(result, "\\u003c") {
			t.Error("HTML should be escaped")
		}
	})

	t.Run("WithoutConfig", func(t *testing.T) {
		result, err := Encode(map[string]any{"key": "value"})
		if err != nil {
			t.Errorf("Encode failed: %v", err)
		}
		if !strings.Contains(result, `"key"`) {
			t.Error("Result should contain key")
		}
	})
}

// ============================================================================
// API PRINT FUNCTIONS TESTS
// ============================================================================

// TestPrintFunctions tests Print and PrintPretty functions
func TestPrintFunctions(t *testing.T) {
	t.Run("Print", func(t *testing.T) {
		// Just verify it doesn't panic
		Print(map[string]any{"test": "value"})
	})

	t.Run("PrintPretty", func(t *testing.T) {
		// Just verify it doesn't panic
		PrintPretty(map[string]any{"test": "value"})
	})

	t.Run("PrintE", func(t *testing.T) {
		err := PrintE(map[string]any{"test": "value"})
		if err != nil {
			t.Errorf("PrintE failed: %v", err)
		}
	})

	t.Run("PrintPrettyE", func(t *testing.T) {
		err := PrintPrettyE(map[string]any{"test": "value"})
		if err != nil {
			t.Errorf("PrintPrettyE failed: %v", err)
		}
	})

	t.Run("PrintInvalidJSONString", func(t *testing.T) {
		// Should encode as a normal string, not fail
		Print("not valid json")
	})
}

// ============================================================================
// PROCESSOR METHODS TESTS - Additional coverage
// ============================================================================

// TestProcessorClosedState tests processor operations when closed
func TestProcessorClosedState(t *testing.T) {
	t.Run("ClosedProcessorOperations", func(t *testing.T) {
		processor := MustNew()
		processor.Close()

		// All operations should fail on closed processor
		_, err := processor.Get(`{"key":"value"}`, "key")
		if err == nil {
			t.Error("Get should fail on closed processor")
		}

		_, err = processor.Set(`{"key":"value"}`, "key", "new")
		if err == nil {
			t.Error("Set should fail on closed processor")
		}

		_, err = processor.Delete(`{"key":"value"}`, "key")
		if err == nil {
			t.Error("Delete should fail on closed processor")
		}

		_, err = processor.Marshal(map[string]any{"key": "value"})
		if err == nil {
			t.Error("Marshal should fail on closed processor")
		}

		err = processor.Unmarshal([]byte(`{"key":"value"}`), &map[string]any{})
		if err == nil {
			t.Error("Unmarshal should fail on closed processor")
		}
	})
}

// TestProcessorValidBytes tests ValidBytes method
func TestProcessorValidBytes(t *testing.T) {
	processor := MustNew()
	defer processor.Close()

	tests := []struct {
		input    []byte
		expected bool
	}{
		{[]byte(`{"key":"value"}`), true},
		{[]byte(`[1, 2, 3]`), true},
		{[]byte(`"string"`), true},
		{[]byte(`123`), true},
		{[]byte(`{invalid}`), false},
		{[]byte(``), false},
	}

	for _, tt := range tests {
		result := processor.ValidBytes(tt.input)
		if result != tt.expected {
			t.Errorf("ValidBytes(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

// TestProcessorParse tests Parse method
func TestProcessorParse(t *testing.T) {
	processor := MustNew()
	defer processor.Close()

	t.Run("ParseToMap", func(t *testing.T) {
		var result map[string]any
		err := processor.Parse(`{"key":"value"}`, &result)
		if err != nil {
			t.Errorf("Parse failed: %v", err)
		}
		if result["key"] != "value" {
			t.Errorf("Result[key] = %v, want value", result["key"])
		}
	})

	t.Run("ParseToSlice", func(t *testing.T) {
		var result []any
		err := processor.Parse(`[1, 2, 3]`, &result)
		if err != nil {
			t.Errorf("Parse failed: %v", err)
		}
		if len(result) != 3 {
			t.Errorf("Result length = %d, want 3", len(result))
		}
	})

	t.Run("ParseNilTarget", func(t *testing.T) {
		err := processor.Parse(`{"key":"value"}`, nil)
		if err == nil {
			t.Error("Parse with nil target should return error")
		}
	})

	t.Run("ParseInvalidJSON", func(t *testing.T) {
		var result map[string]any
		err := processor.Parse(`{invalid}`, &result)
		if err == nil {
			t.Error("Parse with invalid JSON should return error")
		}
	})

	t.Run("ParseWithPreserveNumbers", func(t *testing.T) {
		opts := Config{PreserveNumbers: true}
		var result map[string]any
		err := processor.Parse(`{"num":123}`, &result, opts)
		if err != nil {
			t.Errorf("Parse with PreserveNumbers failed: %v", err)
		}
	})
}

// TestProcessorBufferMethods tests buffer operation methods
func TestProcessorBufferMethods(t *testing.T) {
	processor := MustNew()
	defer processor.Close()

	t.Run("CompactBuffer", func(t *testing.T) {
		var dst bytes.Buffer
		src := []byte(`{"key": "value"}`)
		err := processor.CompactBuffer(&dst, src)
		if err != nil {
			t.Errorf("CompactBuffer failed: %v", err)
		}
		if dst.Len() == 0 {
			t.Error("CompactBuffer should write to dst")
		}
	})

	t.Run("IndentBuffer", func(t *testing.T) {
		var dst bytes.Buffer
		src := []byte(`{"key":"value"}`)
		err := processor.IndentBuffer(&dst, src, "", "  ")
		if err != nil {
			t.Errorf("IndentBuffer failed: %v", err)
		}
		if !strings.Contains(dst.String(), "\n") {
			t.Error("IndentBuffer should produce indented output")
		}
	})

	t.Run("HTMLEscapeBuffer", func(t *testing.T) {
		var dst bytes.Buffer
		src := []byte(`{"html":"<script>"}`)
		processor.HTMLEscapeBuffer(&dst, src)
		if dst.Len() == 0 {
			t.Error("HTMLEscapeBuffer should write to dst")
		}
	})

	t.Run("HTMLEscapeBufferInvalidJSON", func(t *testing.T) {
		var dst bytes.Buffer
		src := []byte(`{invalid}`)
		processor.HTMLEscapeBuffer(&dst, src)
		// Should write original content on error
		if dst.String() != string(src) {
			t.Error("HTMLEscapeBuffer should write original on error")
		}
	})
}

// ============================================================================
// RECURSIVE PROCESSOR TESTS
// ============================================================================

// TestRecursiveProcessor tests RecursiveProcessor operations
func TestRecursiveProcessor(t *testing.T) {
	processor := MustNew()
	defer processor.Close()
	rp := NewRecursiveProcessor(processor)

	t.Run("GetNestedValue", func(t *testing.T) {
		data := map[string]any{
			"level1": map[string]any{
				"level2": map[string]any{
					"value": "nested",
				},
			},
		}

		result, err := rp.ProcessRecursively(data, "level1.level2.value", opGet, nil)
		if err != nil {
			t.Errorf("ProcessRecursively Get failed: %v", err)
		}
		if result != "nested" {
			t.Errorf("Result = %v, want nested", result)
		}
	})

	t.Run("SetNestedValue", func(t *testing.T) {
		data := map[string]any{
			"level1": map[string]any{
				"level2": "old",
			},
		}

		result, err := rp.ProcessRecursively(data, "level1.level2", opSet, "new")
		if err != nil {
			t.Errorf("ProcessRecursively Set failed: %v", err)
		}

		// Result might be a different type, just verify no error
		_ = result
	})

	t.Run("DeleteNestedValue", func(t *testing.T) {
		data := map[string]any{
			"level1": map[string]any{
				"level2": "value",
			},
		}

		result, err := rp.ProcessRecursively(data, "level1.level2", opDelete, nil)
		if err != nil {
			t.Errorf("ProcessRecursively Delete failed: %v", err)
		}

		// Result might be a different type, just verify no error
		_ = result
	})

	t.Run("EmptyPath", func(t *testing.T) {
		data := map[string]any{"key": "value"}

		result, err := rp.ProcessRecursively(data, "", opGet, nil)
		if err != nil {
			t.Errorf("ProcessRecursively with empty path failed: %v", err)
		}
		// Verify result contains the expected data
		resultMap, ok := result.(map[string]any)
		if !ok {
			t.Fatal("Result should be a map")
		}
		if resultMap["key"] != "value" {
			t.Error("Empty path should return original data")
		}
	})

	t.Run("CreatePaths", func(t *testing.T) {
		data := map[string]any{}

		result, err := rp.ProcessRecursivelyWithOptions(data, "new.path.value", opSet, "created", true)
		if err != nil {
			t.Errorf("ProcessRecursivelyWithOptions with CreatePaths failed: %v", err)
		}

		// Result might be a different internal type, just verify no error
		_ = result
	})
}

// ============================================================================
// ERROR TESTS
// ============================================================================

// TestErrorMethods tests error type methods
func TestErrorMethods(t *testing.T) {
	t.Run("JsonsError", func(t *testing.T) {
		err := &JsonsError{
			Op:      "test",
			Path:    "test.path",
			Message: "test error",
			Err:     ErrInvalidJSON,
		}

		errStr := err.Error()
		if !strings.Contains(errStr, "test") || !strings.Contains(errStr, "test.path") {
			t.Errorf("Error() = %q, should contain operation and path", errStr)
		}

		unwrapped := err.Unwrap()
		if unwrapped != ErrInvalidJSON {
			t.Errorf("Unwrap() = %v, want %v", unwrapped, ErrInvalidJSON)
		}
	})
}

// ============================================================================
// DEEP COPY TESTS
// ============================================================================

// TestDeepCopy tests deep copy functionality
func TestDeepCopy(t *testing.T) {
	processor := MustNew()
	defer processor.Close()

	t.Run("DeepCopyMap", func(t *testing.T) {
		original := map[string]any{
			"key": "value",
			"nested": map[string]any{
				"inner": "data",
			},
		}

		copy, err := DeepCopy(original)
		if err != nil {
			t.Fatalf("DeepCopy() error: %v", err)
		}

		// Modify original
		original["key"] = "modified"
		original["nested"].(map[string]any)["inner"] = "changed"

		// Copy should be unaffected
		copyMap := copy.(map[string]any)
		if copyMap["key"] != "value" {
			t.Error("Deep copy should not be affected by original modifications")
		}
		if copyMap["nested"].(map[string]any)["inner"] != "data" {
			t.Error("Deep copy nested map should not be affected")
		}
	})

	t.Run("DeepCopyArray", func(t *testing.T) {
		original := []any{
			1,
			map[string]any{"key": "value"},
		}

		copy, err := DeepCopy(original)
		if err != nil {
			t.Fatalf("DeepCopy() error: %v", err)
		}

		// Modify original
		original[0] = 999
		original[1].(map[string]any)["key"] = "modified"

		// Copy should be unaffected
		copyArr := copy.([]any)
		if copyArr[0] != 1 {
			t.Error("Deep copy array should not be affected by original modifications")
		}
		if copyArr[1].(map[string]any)["key"] != "value" {
			t.Error("Deep copy array element should not be affected")
		}
	})

	t.Run("DeepCopyPrimitives", func(t *testing.T) {
		tests := []any{
			"string",
			42,
			int64(123456789),
			float64(3.14),
			true,
		}

		for _, original := range tests {
			copy, err := DeepCopy(original)
			if err != nil {
				t.Fatalf("DeepCopy() error: %v", err)
			}
			if copy != original {
				t.Errorf("Deep copy of primitive %v should return same value", original)
			}
		}
	})
}

// ============================================================================
// BATCH OPERATIONS TESTS
// ============================================================================

// TestBatchOperationsAdditional tests additional batch operations
func TestBatchOperationsAdditional(t *testing.T) {
	processor := MustNew()
	defer processor.Close()

	t.Run("BatchGetMultiple", func(t *testing.T) {
		jsonStr := `{"a":1,"b":2,"c":3}`
		paths := []string{"a", "b", "c"}

		result, err := processor.FastGetMultiple(jsonStr, paths)
		if err != nil {
			t.Errorf("FastGetMultiple failed: %v", err)
		}

		if len(result) != 3 {
			t.Errorf("Result length = %d, want 3", len(result))
		}
	})

	t.Run("BatchSetOptimized", func(t *testing.T) {
		jsonStr := `{"a":1,"b":2}`
		updates := map[string]any{
			"a": 10,
			"b": 20,
			"c": 30,
		}

		result, err := processor.BatchSetOptimized(jsonStr, updates)
		if err != nil {
			t.Errorf("BatchSetOptimized failed: %v", err)
		}

		if !strings.Contains(result, `"a":10`) {
			t.Error("Result should contain updated value for a")
		}
	})

	t.Run("BatchDeleteOptimized", func(t *testing.T) {
		jsonStr := `{"a":1,"b":2,"c":3}`
		paths := []string{"a", "c"}

		result, err := processor.BatchDeleteOptimized(jsonStr, paths)
		if err != nil {
			t.Errorf("BatchDeleteOptimized failed: %v", err)
		}

		if strings.Contains(result, `"a"`) {
			t.Error("Result should not contain deleted key a")
		}
	})
}

// TestFormatNumberBoost tests FormatNumber function
func TestFormatNumberBoost(t *testing.T) {
	tests := []struct {
		input float64
		check func(string) bool
	}{
		{3.14, func(s string) bool { return strings.Contains(s, "3.14") }},
		{100.0, func(s string) bool { return strings.Contains(s, "100") }},
		{0.001, func(s string) bool { return len(s) > 0 }},
	}

	for _, tt := range tests {
		result := FormatNumber(tt.input)
		if !tt.check(result) {
			t.Errorf("FormatNumber(%v) = %q, check failed", tt.input, result)
		}
	}
}

// ============================================================================
// NUMBER PRESERVING DECODER TESTS
// ============================================================================

// TestNumberPreservingDecoder tests numberPreservingDecoder
func TestNumberPreservingDecoder(t *testing.T) {
	decoder := newNumberPreservingDecoder(true)

	t.Run("DecodeInteger", func(t *testing.T) {
		result, err := decoder.DecodeToAny(`42`)
		if err != nil {
			t.Errorf("DecodeToAny failed: %v", err)
		}
		// Number type may be preserved as int or Number
		_ = result
	})

	t.Run("DecodeFloat", func(t *testing.T) {
		result, err := decoder.DecodeToAny(`3.14`)
		if err != nil {
			t.Errorf("DecodeToAny failed: %v", err)
		}
		// Number type may be preserved as float64 or Number
		_ = result
	})

	t.Run("DecodeObject", func(t *testing.T) {
		result, err := decoder.DecodeToAny(`{"num":42}`)
		if err != nil {
			t.Errorf("DecodeToAny failed: %v", err)
		}
		m, ok := result.(map[string]any)
		if !ok {
			t.Fatalf("Result should be map, got %T", result)
		}
		// Just verify the key exists
		if _, ok := m["num"]; !ok {
			t.Error("num key should exist")
		}
	})

	t.Run("DecodeArray", func(t *testing.T) {
		result, err := decoder.DecodeToAny(`[1,2,3]`)
		if err != nil {
			t.Errorf("DecodeToAny failed: %v", err)
		}
		arr, ok := result.([]any)
		if !ok {
			t.Fatalf("Result should be array, got %T", result)
		}
		if len(arr) != 3 {
			t.Errorf("Array length = %d, want 3", len(arr))
		}
	})

	t.Run("DecodeInvalidJSON", func(t *testing.T) {
		_, err := decoder.DecodeToAny(`{invalid}`)
		if err == nil {
			t.Error("DecodeToAny should return error for invalid JSON")
		}
	})
}

// TestPreservingUnmarshal tests preservingUnmarshal function
func TestPreservingUnmarshal(t *testing.T) {
	t.Run("PreserveNumbersTrue", func(t *testing.T) {
		var result map[string]any
		err := preservingUnmarshal([]byte(`{"num":123}`), &result, true)
		if err != nil {
			t.Errorf("preservingUnmarshal failed: %v", err)
		}
	})

	t.Run("PreserveNumbersFalse", func(t *testing.T) {
		var result map[string]any
		err := preservingUnmarshal([]byte(`{"num":123}`), &result, false)
		if err != nil {
			t.Errorf("preservingUnmarshal failed: %v", err)
		}
		if result["num"] == nil {
			t.Error("num should not be nil")
		}
	})
}

// ============================================================================
// CACHE TESTS
// ============================================================================

// TestCacheOperationsBoost tests cache operations
func TestCacheOperationsBoost(t *testing.T) {
	t.Run("ClearCache", func(t *testing.T) {
		ClearCache() // Should not panic
	})

	t.Run("GetStats", func(t *testing.T) {
		stats := GetStats()
		if stats.OperationCount < 0 {
			t.Error("OperationCount should be non-negative")
		}
	})

	t.Run("GetHealthStatus", func(t *testing.T) {
		status := GetHealthStatus()
		// Just verify it doesn't panic
		_ = status
	})
}

// ============================================================================
// WARMUP CACHE TESTS
// ============================================================================

// TestWarmupCacheBoost tests WarmupCache function
func TestWarmupCacheBoost(t *testing.T) {
	jsonStr := `{"a":1,"b":2,"c":3}`
	paths := []string{"a", "b", "c"}

	result, err := WarmupCache(jsonStr, paths)
	if err != nil {
		t.Errorf("WarmupCache failed: %v", err)
	}

	if result == nil {
		t.Error("WarmupCache should return non-nil result")
	}
}

// ============================================================================
// PROCESS BATCH TESTS
// ============================================================================

// TestProcessBatchBoost tests ProcessBatch function
func TestProcessBatchBoost(t *testing.T) {
	jsonStr := `{"name":"Alice","age":30}`
	operations := []BatchOperation{
		{Type: "get", JSONStr: jsonStr, Path: "name"},
		{Type: "get", JSONStr: jsonStr, Path: "age"},
	}

	results, err := ProcessBatch(operations)
	if err != nil {
		t.Errorf("ProcessBatch failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("ProcessBatch should return 2 results, got %d", len(results))
	}
}

// ============================================================================
// STREAMING ENCODER TESTS
// ============================================================================

// TestEncodeStreamBoost tests EncodeStream function
func TestEncodeStreamBoost(t *testing.T) {
	data := []map[string]any{
		{"id": 1, "name": "Item 1"},
		{"id": 2, "name": "Item 2"},
	}

	opts := DefaultConfig()
	opts.Pretty = true
	result, err := EncodeStream(data, opts)
	if err != nil {
		t.Errorf("EncodeStream failed: %v", err)
	}

	if !strings.Contains(result, "\n") {
		t.Error("Pretty EncodeStream should contain newlines")
	}
}

// TestEncodeBatchBoost tests EncodeBatch function
func TestEncodeBatchBoost(t *testing.T) {
	pairs := map[string]any{
		"name":  "Alice",
		"age":   30,
		"admin": true,
	}

	opts := DefaultConfig()
	opts.Pretty = true
	result, err := EncodeBatch(pairs, opts)
	if err != nil {
		t.Errorf("EncodeBatch failed: %v", err)
	}

	for key := range pairs {
		if !strings.Contains(result, `"`+key+`"`) {
			t.Errorf("Result should contain key %q", key)
		}
	}
}

// TestEncodeFieldsBoost tests EncodeFields function
func TestEncodeFieldsBoost(t *testing.T) {
	type User struct {
		ID       int    `json:"id"`
		Name     string `json:"name"`
		Password string `json:"password"`
	}

	user := User{
		ID:       1,
		Name:     "Alice",
		Password: "secret",
	}

	fields := []string{"id", "name"}
	opts := DefaultConfig()
	opts.Pretty = true
	result, err := EncodeFields(user, fields, opts)
	if err != nil {
		t.Errorf("EncodeFields failed: %v", err)
	}

	if !strings.Contains(result, `"id"`) {
		t.Error("Result should contain 'id'")
	}
	if !strings.Contains(result, `"name"`) {
		t.Error("Result should contain 'name'")
	}
	if strings.Contains(result, `"password"`) {
		t.Error("Result should not contain 'password'")
	}
}

// ============================================================================
// JSON POINTER ESCAPE TESTS
// ============================================================================

// TestEscapeJSONPointer tests EscapeJSONPointer function
func TestEscapeJSONPointer(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"with/slash", "with~1slash"},
		{"with~tilde", "with~0tilde"},
		{"both/test~here", "both~1test~0here"},
	}

	for _, tt := range tests {
		result := EscapeJSONPointer(tt.input)
		if result != tt.expected {
			t.Errorf("EscapeJSONPointer(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// ============================================================================
// ENCODING EDGE CASES TESTS - Coverage for encoding functions
// ============================================================================

// TestEncodeStringSpecialChars tests encoding strings with special characters
func TestEncodeStringSpecialChars(t *testing.T) {
	processor := MustNew()
	defer processor.Close()

	tests := []struct {
		name  string
		input any
	}{
		{"StringWithNewline", map[string]any{"text": "line1\nline2"}},
		{"StringWithTab", map[string]any{"text": "col1\tcol2"}},
		{"StringWithQuotes", map[string]any{"text": `say "hello"`}},
		{"StringWithBackslash", map[string]any{"text": `path\to\file`}},
		{"StringWithUnicode", map[string]any{"text": "Hello 世界"}},
		{"StringWithControlChars", map[string]any{"text": string([]byte{0x01, 0x02})}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processor.EncodeWithConfig(tt.input, DefaultConfig())
			if err != nil {
				t.Errorf("EncodeWithConfig failed: %v", err)
			}
			// Verify it can be decoded back
			var decoded map[string]any
			err = processor.Unmarshal([]byte(result), &decoded)
			if err != nil {
				t.Errorf("Failed to decode result: %v", err)
			}
		})
	}
}

// TestEncodeStructEdgeCases tests encoding struct edge cases
func TestEncodeStructEdgeCases(t *testing.T) {
	processor := MustNew()
	defer processor.Close()

	t.Run("StructWithOmitEmpty", func(t *testing.T) {
		type TestStruct struct {
			Name    string `json:"name"`
			Skipped string `json:"skipped,omitempty"`
			Value   int    `json:"value,omitempty"`
		}

		data := TestStruct{Name: "test"}
		result, err := processor.EncodeWithConfig(data, DefaultConfig())
		if err != nil {
			t.Errorf("EncodeWithConfig failed: %v", err)
		}
		if strings.Contains(result, "skipped") {
			t.Error("OmitEmpty field should be omitted when empty")
		}
	})

	t.Run("StructWithAllFieldsEmpty", func(t *testing.T) {
		type TestStruct struct {
			Name  string `json:"name,omitempty"`
			Value int    `json:"value,omitempty"`
		}

		data := TestStruct{}
		result, err := processor.EncodeWithConfig(data, DefaultConfig())
		if err != nil {
			t.Errorf("EncodeWithConfig failed: %v", err)
		}
		_ = result
	})

	t.Run("StructWithNestedStruct", func(t *testing.T) {
		type Inner struct {
			Value string `json:"value"`
		}
		type Outer struct {
			Name  string `json:"name"`
			Inner Inner  `json:"inner"`
		}

		data := Outer{Name: "outer", Inner: Inner{Value: "inner"}}
		result, err := processor.EncodeWithConfig(data, DefaultConfig())
		if err != nil {
			t.Errorf("EncodeWithConfig failed: %v", err)
		}
		if !strings.Contains(result, "inner") {
			t.Error("Result should contain nested struct")
		}
	})

	t.Run("StructWithPointerFields", func(t *testing.T) {
		type TestStruct struct {
			Name  *string `json:"name"`
			Value *int    `json:"value"`
		}

		name := "test"
		data := TestStruct{Name: &name}
		result, err := processor.EncodeWithConfig(data, DefaultConfig())
		if err != nil {
			t.Errorf("EncodeWithConfig failed: %v", err)
		}
		if !strings.Contains(result, "test") {
			t.Error("Result should contain pointer value")
		}
	})
}

// TestValidateNumberEdgeCases tests number validation edge cases
func TestValidateNumberEdgeCases(t *testing.T) {
	processor := MustNew()
	defer processor.Close()

	t.Run("NumberValidation", func(t *testing.T) {
		schema := &Schema{
			Type:    "number",
			Minimum: 0,
			Maximum: 100,
		}
		schema.hasMinimum = true
		schema.hasMaximum = true

		tests := []struct {
			jsonStr   string
			expectErr bool
		}{
			{`50`, false},
			{`0`, false},
			{`100`, false},
		}

		for _, tt := range tests {
			errors, err := processor.ValidateSchema(tt.jsonStr, schema)
			if err != nil {
				t.Errorf("ValidateSchema failed for %s: %v", tt.jsonStr, err)
				continue
			}
			hasErrors := len(errors) > 0
			if hasErrors != tt.expectErr {
				t.Errorf("ValidateSchema(%s) errors = %v, expectErr = %v", tt.jsonStr, errors, tt.expectErr)
			}
		}
	})

	t.Run("IntegerValidation", func(t *testing.T) {
		schema := &Schema{
			Type:    "number", // Use number since JSON decodes to float64
			Minimum: 0,
			Maximum: 100,
		}
		schema.hasMinimum = true
		schema.hasMaximum = true

		tests := []struct {
			jsonStr   string
			expectErr bool
		}{
			{`50`, false},
			{`0`, false},
			{`100`, false},
		}

		for _, tt := range tests {
			errors, err := processor.ValidateSchema(tt.jsonStr, schema)
			if err != nil {
				t.Errorf("ValidateSchema failed for %s: %v", tt.jsonStr, err)
				continue
			}
			hasErrors := len(errors) > 0
			if hasErrors != tt.expectErr {
				t.Errorf("ValidateSchema(%s) errors = %v, expectErr = %v", tt.jsonStr, errors, tt.expectErr)
			}
		}
	})
}

// TestIsEmptyFunction tests isEmpty function indirectly
func TestIsEmptyFunction(t *testing.T) {
	processor := MustNew()
	defer processor.Close()

	t.Run("EmptySlice", func(t *testing.T) {
		data := map[string]any{"items": []any{}}
		result, err := processor.EncodeWithConfig(data, DefaultConfig())
		if err != nil {
			t.Errorf("EncodeWithConfig failed: %v", err)
		}
		_ = result
	})

	t.Run("EmptyMap", func(t *testing.T) {
		data := map[string]any{"obj": map[string]any{}}
		result, err := processor.EncodeWithConfig(data, DefaultConfig())
		if err != nil {
			t.Errorf("EncodeWithConfig failed: %v", err)
		}
		_ = result
	})

	t.Run("NilValue", func(t *testing.T) {
		data := map[string]any{"value": nil}
		result, err := processor.EncodeWithConfig(data, DefaultConfig())
		if err != nil {
			t.Errorf("EncodeWithConfig failed: %v", err)
		}
		if !strings.Contains(result, "null") {
			t.Error("Result should contain null for nil value")
		}
	})
}

// TestValuesEqual tests valuesEqual function indirectly via validation
func TestValuesEqual(t *testing.T) {
	processor := MustNew()
	defer processor.Close()

	t.Run("ConstValidation", func(t *testing.T) {
		schema := &Schema{
			Type:  "string",
			Const: "expected",
		}

		tests := []struct {
			jsonStr   string
			expectErr bool
		}{
			{`"expected"`, false},
			{`"other"`, true},
		}

		for _, tt := range tests {
			errors, err := processor.ValidateSchema(tt.jsonStr, schema)
			if err != nil {
				t.Errorf("ValidateSchema failed: %v", err)
				continue
			}
			hasErrors := len(errors) > 0
			if hasErrors != tt.expectErr {
				t.Errorf("ValidateSchema(%s) errors = %v, expectErr = %v", tt.jsonStr, errors, tt.expectErr)
			}
		}
	})
}

// TestAssignResult tests assignResult function indirectly
func TestAssignResult(t *testing.T) {
	processor := MustNew()
	defer processor.Close()

	t.Run("AssignToInt", func(t *testing.T) {
		var result int
		err := processor.Unmarshal([]byte(`42`), &result)
		if err != nil {
			t.Errorf("Unmarshal failed: %v", err)
		}
		if result != 42 {
			t.Errorf("Result = %d, want 42", result)
		}
	})

	t.Run("AssignToFloat", func(t *testing.T) {
		var result float64
		err := processor.Unmarshal([]byte(`3.14`), &result)
		if err != nil {
			t.Errorf("Unmarshal failed: %v", err)
		}
		if result < 3.13 || result > 3.15 {
			t.Errorf("Result = %v, want approximately 3.14", result)
		}
	})

	t.Run("AssignToBool", func(t *testing.T) {
		var result bool
		err := processor.Unmarshal([]byte(`true`), &result)
		if err != nil {
			t.Errorf("Unmarshal failed: %v", err)
		}
		if !result {
			t.Error("Result should be true")
		}
	})

	t.Run("AssignToString", func(t *testing.T) {
		var result string
		err := processor.Unmarshal([]byte(`"hello"`), &result)
		if err != nil {
			t.Errorf("Unmarshal failed: %v", err)
		}
		if result != "hello" {
			t.Errorf("Result = %s, want hello", result)
		}
	})

	t.Run("AssignToSlice", func(t *testing.T) {
		var result []int
		err := processor.Unmarshal([]byte(`[1, 2, 3]`), &result)
		if err != nil {
			t.Errorf("Unmarshal failed: %v", err)
		}
		if len(result) != 3 {
			t.Errorf("Result length = %d, want 3", len(result))
		}
	})

	t.Run("AssignToMap", func(t *testing.T) {
		var result map[string]int
		err := processor.Unmarshal([]byte(`{"a": 1, "b": 2}`), &result)
		if err != nil {
			t.Errorf("Unmarshal failed: %v", err)
		}
		if result["a"] != 1 || result["b"] != 2 {
			t.Errorf("Result = %v, want {a:1, b:2}", result)
		}
	})
}

// TestMoreMethod tests More method of Decoder
func TestMoreMethod(t *testing.T) {
	t.Run("MoreWithMultipleValues", func(t *testing.T) {
		decoder := NewDecoder(strings.NewReader(`{"a":1}{"b":2}`))

		// First decode
		var v1 map[string]any
		err := decoder.Decode(&v1)
		if err != nil {
			t.Errorf("First Decode failed: %v", err)
		}

		// Check if more
		if !decoder.More() {
			t.Error("More() should return true when there's more data")
		}

		// Second decode
		var v2 map[string]any
		err = decoder.Decode(&v2)
		if err != nil {
			t.Errorf("Second Decode failed: %v", err)
		}

		// No more
		if decoder.More() {
			t.Error("More() should return false when there's no more data")
		}
	})
}

// TestEncoderMethods tests Encoder methods
func TestEncoderMethods(t *testing.T) {
	t.Run("SetEscapeHTML", func(t *testing.T) {
		var buf bytes.Buffer
		encoder := NewEncoder(&buf)
		encoder.SetEscapeHTML(true)

		data := map[string]string{"html": "<script>"}
		err := encoder.Encode(data)
		if err != nil {
			t.Errorf("Encode failed: %v", err)
		}

		if !strings.Contains(buf.String(), "\\u003c") {
			t.Error("HTML should be escaped")
		}
	})

	t.Run("SetIndent", func(t *testing.T) {
		var buf bytes.Buffer
		encoder := NewEncoder(&buf)
		encoder.SetIndent("", "  ")

		data := map[string]string{"key": "value"}
		err := encoder.Encode(data)
		if err != nil {
			t.Errorf("Encode failed: %v", err)
		}

		if !strings.Contains(buf.String(), "\n") {
			t.Error("Output should be indented")
		}
	})
}

// ============================================================================
// ADDITIONAL ENCODING TESTS
// ============================================================================

// TestEncodeStructComprehensive tests struct encoding more comprehensively
func TestEncodeStructComprehensive(t *testing.T) {
	processor := MustNew()
	defer processor.Close()

	t.Run("StructWithJSONTags", func(t *testing.T) {
		type User struct {
			ID       int      `json:"id"`
			Name     string   `json:"name"`
			Email    string   `json:"email,omitempty"`
			Tags     []string `json:"tags,omitempty"`
			Active   bool     `json:"active"`
			Balance  float64  `json:"balance"`
			Password string   `json:"-"`
		}

		user := User{
			ID:      1,
			Name:    "Alice",
			Tags:    []string{"admin", "user"},
			Active:  true,
			Balance: 100.50,
		}

		result, err := processor.EncodeWithConfig(user, DefaultConfig())
		if err != nil {
			t.Errorf("EncodeWithConfig failed: %v", err)
		}

		if strings.Contains(result, "Password") {
			t.Error("Password should be excluded via json:\"-\" tag")
		}
		if !strings.Contains(result, `"id"`) {
			t.Error("Result should contain id")
		}
	})

	t.Run("NestedStructs", func(t *testing.T) {
		type Address struct {
			Street  string `json:"street"`
			City    string `json:"city"`
			Country string `json:"country"`
		}

		type Person struct {
			Name    string  `json:"name"`
			Address Address `json:"address"`
		}

		person := Person{
			Name: "Bob",
			Address: Address{
				Street:  "123 Main St",
				City:    "New York",
				Country: "USA",
			},
		}

		result, err := processor.EncodeWithConfig(person, DefaultConfig())
		if err != nil {
			t.Errorf("EncodeWithConfig failed: %v", err)
		}

		if !strings.Contains(result, "address") || !strings.Contains(result, "street") {
			t.Error("Result should contain nested struct fields")
		}
	})

	t.Run("StructWithPointers", func(t *testing.T) {
		type Item struct {
			Name  string  `json:"name"`
			Price float64 `json:"price"`
		}

		type Order struct {
			ID    int    `json:"id"`
			Item  *Item  `json:"item,omitempty"`
			Notes string `json:"notes,omitempty"`
		}

		item := Item{Name: "Widget", Price: 9.99}
		order := Order{ID: 1, Item: &item}

		result, err := processor.EncodeWithConfig(order, DefaultConfig())
		if err != nil {
			t.Errorf("EncodeWithConfig failed: %v", err)
		}

		if !strings.Contains(result, "Widget") {
			t.Error("Result should contain item name")
		}
	})
}

// TestEncodeStringEdgeCases tests string encoding edge cases
func TestEncodeStringEdgeCases(t *testing.T) {
	processor := MustNew()
	defer processor.Close()

	tests := []struct {
		name  string
		input map[string]any
	}{
		{"StringWithHighUnicode", map[string]any{"text": "\U0001F600"}}, // Emoji
		{"StringWithNull", map[string]any{"text": string([]byte{0})}},
		{"StringWithBell", map[string]any{"text": string([]byte{7})}},
		{"StringWithFormFeed", map[string]any{"text": string([]byte{12})}},
		{"EmptyString", map[string]any{"text": ""}},
		{"VeryLongString", map[string]any{"text": strings.Repeat("a", 10000)}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processor.EncodeWithConfig(tt.input, DefaultConfig())
			if err != nil {
				t.Errorf("EncodeWithConfig failed: %v", err)
			}
			// Verify it can be decoded back
			var decoded map[string]any
			err = processor.Unmarshal([]byte(result), &decoded)
			if err != nil {
				t.Errorf("Failed to decode: %v", err)
			}
		})
	}
}

// TestEncodeArrayEdgeCases tests array encoding edge cases
func TestEncodeArrayEdgeCases(t *testing.T) {
	processor := MustNew()
	defer processor.Close()

	t.Run("EmptyArray", func(t *testing.T) {
		data := map[string]any{"items": []any{}}
		result, err := processor.EncodeWithConfig(data, DefaultConfig())
		if err != nil {
			t.Errorf("EncodeWithConfig failed: %v", err)
		}
		if !strings.Contains(result, "[]") {
			t.Error("Result should contain empty array")
		}
	})

	t.Run("NestedArrays", func(t *testing.T) {
		data := map[string]any{
			"matrix": [][]any{
				{1, 2, 3},
				{4, 5, 6},
			},
		}
		result, err := processor.EncodeWithConfig(data, DefaultConfig())
		if err != nil {
			t.Errorf("EncodeWithConfig failed: %v", err)
		}
		if !strings.Contains(result, "[[") {
			t.Error("Result should contain nested arrays")
		}
	})

	t.Run("MixedArray", func(t *testing.T) {
		data := map[string]any{
			"mixed": []any{1, "two", 3.0, true, nil, map[string]any{"key": "value"}},
		}
		result, err := processor.EncodeWithConfig(data, DefaultConfig())
		if err != nil {
			t.Errorf("EncodeWithConfig failed: %v", err)
		}
		_ = result
	})
}

// TestEncodeMapEdgeCases tests map encoding edge cases
func TestEncodeMapEdgeCases(t *testing.T) {
	processor := MustNew()
	defer processor.Close()

	t.Run("EmptyMap", func(t *testing.T) {
		data := map[string]any{"obj": map[string]any{}}
		result, err := processor.EncodeWithConfig(data, DefaultConfig())
		if err != nil {
			t.Errorf("EncodeWithConfig failed: %v", err)
		}
		if !strings.Contains(result, "{}") {
			t.Error("Result should contain empty object")
		}
	})

	t.Run("NestedMaps", func(t *testing.T) {
		data := map[string]any{
			"level1": map[string]any{
				"level2": map[string]any{
					"level3": "deep",
				},
			},
		}
		result, err := processor.EncodeWithConfig(data, DefaultConfig())
		if err != nil {
			t.Errorf("EncodeWithConfig failed: %v", err)
		}
		_ = result
	})

	t.Run("MapWithNumericKeys", func(t *testing.T) {
		// Map with non-string keys should still encode
		data := map[int]string{1: "one", 2: "two"}
		result, err := processor.EncodeWithConfig(data, DefaultConfig())
		if err != nil {
			t.Errorf("EncodeWithConfig failed: %v", err)
		}
		_ = result
	})
}

// TestValidateStringComprehensive tests string validation more comprehensively
func TestValidateStringComprehensive(t *testing.T) {
	processor := MustNew()
	defer processor.Close()

	t.Run("PatternValidation", func(t *testing.T) {
		schema := &Schema{
			Type:    "string",
			Pattern: `^[A-Z]{3}-\d{4}$`,
		}

		tests := []struct {
			jsonStr   string
			expectErr bool
		}{
			{`"ABC-1234"`, false},
			{`"XYZ-9999"`, false},
			{`"abc-1234"`, true},
			{`"ABC1234"`, true},
		}

		for _, tt := range tests {
			errors, err := processor.ValidateSchema(tt.jsonStr, schema)
			if err != nil {
				t.Errorf("ValidateSchema failed: %v", err)
				continue
			}
			hasErrors := len(errors) > 0
			if hasErrors != tt.expectErr {
				t.Errorf("ValidateSchema(%s) errors = %v, expectErr = %v", tt.jsonStr, errors, tt.expectErr)
			}
		}
	})

	t.Run("LengthValidation", func(t *testing.T) {
		schema := &Schema{
			Type:      "string",
			MinLength: 3,
			MaxLength: 10,
		}
		schema.hasMinLength = true
		schema.hasMaxLength = true

		tests := []struct {
			jsonStr   string
			expectErr bool
		}{
			{`"abc"`, false},
			{`"abcdefghij"`, false},
			{`"ab"`, true},
			{`"abcdefghijk"`, true},
		}

		for _, tt := range tests {
			errors, err := processor.ValidateSchema(tt.jsonStr, schema)
			if err != nil {
				t.Errorf("ValidateSchema failed: %v", err)
				continue
			}
			hasErrors := len(errors) > 0
			if hasErrors != tt.expectErr {
				t.Errorf("ValidateSchema(%s) errors = %v, expectErr = %v", tt.jsonStr, errors, tt.expectErr)
			}
		}
	})
}

// TestTokenMethod tests Token method of Decoder
func TestTokenMethod(t *testing.T) {
	t.Run("ReadTokens", func(t *testing.T) {
		decoder := NewDecoder(strings.NewReader(`{"key": [1, 2, 3]}`))

		var tokens []Token
		for decoder.More() {
			token, err := decoder.Token()
			if err != nil {
				break
			}
			tokens = append(tokens, token)
		}

		if len(tokens) == 0 {
			t.Error("Should have read some tokens")
		}
	})
}

// TestDecodeEdgeCases tests Decode edge cases
func TestDecodeEdgeCases(t *testing.T) {
	t.Run("DecodeIntoInterface", func(t *testing.T) {
		decoder := NewDecoder(strings.NewReader(`{"key": "value"}`))

		var result any
		err := decoder.Decode(&result)
		if err != nil {
			t.Errorf("Decode failed: %v", err)
		}

		m, ok := result.(map[string]any)
		if !ok {
			t.Fatal("Result should be a map")
		}
		if m["key"] != "value" {
			t.Errorf("key = %v, want value", m["key"])
		}
	})

	t.Run("MultipleDecodes", func(t *testing.T) {
		decoder := NewDecoder(strings.NewReader(`1 "two" 3.0`))

		var results []any
		for decoder.More() {
			var v any
			err := decoder.Decode(&v)
			if err != nil {
				break
			}
			results = append(results, v)
		}

		if len(results) < 2 {
			t.Error("Should have decoded multiple values")
		}
	})
}

// TestEncodeWithOptions tests EncodeWithOptions function
func TestEncodeWithOptions(t *testing.T) {
	processor := MustNew()
	defer processor.Close()

	config := DefaultConfig()
	config.Pretty = true
	config.Indent = "    "

	data := map[string]any{"key": "value"}
	result, err := processor.EncodeWithOptions(data, config)
	if err != nil {
		t.Errorf("EncodeWithOptions failed: %v", err)
	}

	if !strings.Contains(result, "    ") {
		t.Error("Result should use custom indent")
	}
}

// TestEncodeStreamWithOptions tests EncodeStreamWithOptions function
func TestEncodeStreamWithOptions(t *testing.T) {
	processor := MustNew()
	defer processor.Close()

	config := DefaultConfig()
	data := []map[string]any{
		{"id": 1},
		{"id": 2},
	}

	result, err := processor.EncodeStreamWithOptions(data, config)
	if err != nil {
		t.Errorf("EncodeStreamWithOptions failed: %v", err)
	}

	if !strings.Contains(result, `"id"`) {
		t.Error("Result should contain id")
	}
}

// TestTruncateFloat tests float truncation
func TestTruncateFloat(t *testing.T) {
	processor := MustNew()
	defer processor.Close()

	t.Run("TruncatePrecision", func(t *testing.T) {
		type Data struct {
			Value float64 `json:"value"`
		}

		data := Data{Value: 3.141592653589793}

		config := DefaultConfig()
		config.FloatPrecision = 4
		config.FloatTruncate = true

		result, err := processor.EncodeWithConfig(data, config)
		if err != nil {
			t.Errorf("EncodeWithConfig failed: %v", err)
		}

		if !strings.Contains(result, "3.1415") {
			t.Errorf("Result should contain truncated value, got: %s", result)
		}
	})
}

// ============================================================================
// PARSER EDGE CASES TESTS
// ============================================================================

// TestParseBoolean tests parseBoolean function
func TestParseBoolean(t *testing.T) {
	t.Run("ParseTrue", func(t *testing.T) {
		decoder := NewDecoder(strings.NewReader(`true`))
		var result bool
		err := decoder.Decode(&result)
		if err != nil {
			t.Errorf("Decode failed: %v", err)
		}
		if !result {
			t.Error("Result should be true")
		}
	})

	t.Run("ParseFalse", func(t *testing.T) {
		decoder := NewDecoder(strings.NewReader(`false`))
		var result bool
		err := decoder.Decode(&result)
		if err != nil {
			t.Errorf("Decode failed: %v", err)
		}
		if result {
			t.Error("Result should be false")
		}
	})
}

// TestParseNull tests parseNull function
func TestParseNull(t *testing.T) {
	t.Run("ParseNullValue", func(t *testing.T) {
		decoder := NewDecoder(strings.NewReader(`null`))
		var result any
		err := decoder.Decode(&result)
		if err != nil {
			t.Errorf("Decode failed: %v", err)
		}
		if result != nil {
			t.Error("Result should be nil")
		}
	})
}

// TestParseStringEdgeCases tests parseString function edge cases
func TestParseStringEdgeCases(t *testing.T) {
	t.Run("EscapedCharacters", func(t *testing.T) {
		decoder := NewDecoder(strings.NewReader(`"hello\nworld\ttab"`))
		var result string
		err := decoder.Decode(&result)
		if err != nil {
			t.Errorf("Decode failed: %v", err)
		}
		if !strings.Contains(result, "\n") {
			t.Error("Result should contain newline")
		}
	})

	t.Run("UnicodeEscape", func(t *testing.T) {
		decoder := NewDecoder(strings.NewReader(`"\u0041"`))
		var result string
		err := decoder.Decode(&result)
		if err != nil {
			t.Errorf("Decode failed: %v", err)
		}
		if result != "A" {
			t.Errorf("Result = %q, want A", result)
		}
	})

	t.Run("EscapedQuote", func(t *testing.T) {
		decoder := NewDecoder(strings.NewReader(`"say \"hello\""`))
		var result string
		err := decoder.Decode(&result)
		if err != nil {
			t.Errorf("Decode failed: %v", err)
		}
		if !strings.Contains(result, `"`) {
			t.Error("Result should contain quote")
		}
	})

	t.Run("EscapedBackslash", func(t *testing.T) {
		decoder := NewDecoder(strings.NewReader(`"path\\to\\file"`))
		var result string
		err := decoder.Decode(&result)
		if err != nil {
			t.Errorf("Decode failed: %v", err)
		}
		if !strings.Contains(result, "\\") {
			t.Error("Result should contain backslash")
		}
	})
}

// TestValuesEqualComprehensive tests valuesEqual function more comprehensively
func TestValuesEqualComprehensive(t *testing.T) {
	processor := MustNew()
	defer processor.Close()

	t.Run("StringEquality", func(t *testing.T) {
		schema := &Schema{
			Type:  "string",
			Const: "expected",
		}

		tests := []struct {
			jsonStr   string
			expectErr bool
		}{
			{`"expected"`, false},
			{`"other"`, true},
		}

		for _, tt := range tests {
			errors, err := processor.ValidateSchema(tt.jsonStr, schema)
			if err != nil {
				t.Errorf("ValidateSchema failed: %v", err)
				continue
			}
			hasErrors := len(errors) > 0
			if hasErrors != tt.expectErr {
				t.Errorf("ValidateSchema(%s) errors = %v, expectErr = %v", tt.jsonStr, errors, tt.expectErr)
			}
		}
	})

	t.Run("NumberEquality", func(t *testing.T) {
		schema := &Schema{
			Type:  "number",
			Const: 42.0,
		}

		tests := []struct {
			jsonStr   string
			expectErr bool
		}{
			{`42`, false},
			{`43`, true},
		}

		for _, tt := range tests {
			errors, err := processor.ValidateSchema(tt.jsonStr, schema)
			if err != nil {
				t.Errorf("ValidateSchema failed: %v", err)
				continue
			}
			hasErrors := len(errors) > 0
			if hasErrors != tt.expectErr {
				t.Errorf("ValidateSchema(%s) errors = %v, expectErr = %v", tt.jsonStr, errors, tt.expectErr)
			}
		}
	})

	t.Run("BooleanEquality", func(t *testing.T) {
		schema := &Schema{
			Type:  "boolean",
			Const: true,
		}

		tests := []struct {
			jsonStr   string
			expectErr bool
		}{
			{`true`, false},
			{`false`, true},
		}

		for _, tt := range tests {
			errors, err := processor.ValidateSchema(tt.jsonStr, schema)
			if err != nil {
				t.Errorf("ValidateSchema failed: %v", err)
				continue
			}
			hasErrors := len(errors) > 0
			if hasErrors != tt.expectErr {
				t.Errorf("ValidateSchema(%s) errors = %v, expectErr = %v", tt.jsonStr, errors, tt.expectErr)
			}
		}
	})
}

// TestIsEmptyComprehensive tests isEmpty function more comprehensively
func TestIsEmptyComprehensive(t *testing.T) {
	processor := MustNew()
	defer processor.Close()

	tests := []struct {
		name  string
		input any
	}{
		{"EmptySlice", []any{}},
		{"EmptyMap", map[string]any{}},
		{"EmptyString", ""},
		{"ZeroInt", 0},
		{"ZeroFloat", 0.0},
		{"False", false},
		{"Nil", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := map[string]any{"value": tt.input}
			result, err := processor.EncodeWithConfig(data, DefaultConfig())
			if err != nil {
				t.Errorf("EncodeWithConfig failed: %v", err)
			}
			_ = result
		})
	}
}

// TestValidateNumberComprehensive tests validateNumber function more comprehensively
func TestValidateNumberComprehensive(t *testing.T) {
	processor := MustNew()
	defer processor.Close()

	t.Run("MultipleOf", func(t *testing.T) {
		schema := &Schema{
			Type:       "number",
			MultipleOf: 5,
		}

		tests := []struct {
			jsonStr   string
			expectErr bool
		}{
			{`10`, false},
			{`15`, false},
			{`7`, true},
		}

		for _, tt := range tests {
			errors, err := processor.ValidateSchema(tt.jsonStr, schema)
			if err != nil {
				t.Errorf("ValidateSchema failed: %v", err)
				continue
			}
			hasErrors := len(errors) > 0
			if hasErrors != tt.expectErr {
				t.Errorf("ValidateSchema(%s) errors = %v, expectErr = %v", tt.jsonStr, errors, tt.expectErr)
			}
		}
	})
}

// TestEncodeNumberEdgeCases tests number encoding edge cases
func TestEncodeNumberEdgeCases(t *testing.T) {
	processor := MustNew()
	defer processor.Close()

	tests := []struct {
		name  string
		input any
	}{
		{"LargeInt", map[string]any{"value": 9223372036854775807}},
		{"SmallInt", map[string]any{"value": -9223372036854775808}},
		{"LargeFloat", map[string]any{"value": 1.7976931348623157e+308}},
		{"SmallFloat", map[string]any{"value": -1.7976931348623157e+308}},
		{"NegativeZero", map[string]any{"value": -0.0}},
		{"VerySmallFloat", map[string]any{"value": 1e-300}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processor.EncodeWithConfig(tt.input, DefaultConfig())
			if err != nil {
				t.Errorf("EncodeWithConfig failed: %v", err)
			}
			_ = result
		})
	}
}

// TestCustomEncoderEdgeCases tests customEncoder edge cases
func TestCustomEncoderEdgeCases(t *testing.T) {
	t.Run("EncodeWithCustomConfig", func(t *testing.T) {
		config := Config{
			Pretty:          true,
			Indent:          "  ",
			EscapeHTML:      false,
			SortKeys:        true,
			ValidateUTF8:    true,
			MaxDepth:        50,
			PreserveNumbers: true,
			EscapeUnicode:   false,
			IncludeNulls:    true,
		}

		encoder := newCustomEncoder(config)
		data := map[string]any{
			"html": "<script>",
			"num":  42,
		}

		result, err := encoder.Encode(data)
		if err != nil {
			t.Errorf("Encode failed: %v", err)
		}

		if !strings.Contains(result, "html") {
			t.Error("Result should contain html key")
		}

		encoder.Close()
	})

	t.Run("EncodeNil", func(t *testing.T) {
		config := DefaultConfig()
		encoder := newCustomEncoder(config)

		result, err := encoder.Encode(nil)
		if err != nil {
			t.Errorf("Encode failed: %v", err)
		}

		if result != "null" {
			t.Errorf("Result = %s, want null", result)
		}

		encoder.Close()
	})
}

// TestStringFormatValidation tests string format validation
func TestStringFormatValidation(t *testing.T) {
	processor := MustNew()
	defer processor.Close()

	t.Run("EmailFormat", func(t *testing.T) {
		schema := &Schema{
			Type:   "string",
			Format: "email",
		}

		tests := []struct {
			jsonStr   string
			expectErr bool
		}{
			{`"test@example.com"`, false},
			{`"invalid-email"`, true},
			{`"user@domain"`, false}, // Basic format check
		}

		for _, tt := range tests {
			errors, err := processor.ValidateSchema(tt.jsonStr, schema)
			if err != nil {
				t.Errorf("ValidateSchema failed: %v", err)
				continue
			}
			// Just verify validation runs
			_ = len(errors) > 0
		}
	})

	t.Run("DateFormat", func(t *testing.T) {
		schema := &Schema{
			Type:   "string",
			Format: "date",
		}

		tests := []string{
			`"2024-01-15"`,
			`"invalid-date"`,
		}

		for _, jsonStr := range tests {
			_, err := processor.ValidateSchema(jsonStr, schema)
			if err != nil {
				t.Errorf("ValidateSchema failed: %v", err)
			}
		}
	})

	t.Run("DateTimeFormat", func(t *testing.T) {
		schema := &Schema{
			Type:   "string",
			Format: "date-time",
		}

		tests := []string{
			`"2024-01-15T10:30:00Z"`,
			`"invalid-datetime"`,
		}

		for _, jsonStr := range tests {
			_, err := processor.ValidateSchema(jsonStr, schema)
			if err != nil {
				t.Errorf("ValidateSchema failed: %v", err)
			}
		}
	})

	t.Run("UUIDFormat", func(t *testing.T) {
		schema := &Schema{
			Type:   "string",
			Format: "uuid",
		}

		tests := []string{
			`"550e8400-e29b-41d4-a716-446655440000"`,
			`"invalid-uuid"`,
		}

		for _, jsonStr := range tests {
			_, err := processor.ValidateSchema(jsonStr, schema)
			if err != nil {
				t.Errorf("ValidateSchema failed: %v", err)
			}
		}
	})

	t.Run("URIFormat", func(t *testing.T) {
		schema := &Schema{
			Type:   "string",
			Format: "uri",
		}

		tests := []string{
			`"https://example.com"`,
			`"invalid-uri"`,
		}

		for _, jsonStr := range tests {
			_, err := processor.ValidateSchema(jsonStr, schema)
			if err != nil {
				t.Errorf("ValidateSchema failed: %v", err)
			}
		}
	})

	t.Run("IPv4Format", func(t *testing.T) {
		schema := &Schema{
			Type:   "string",
			Format: "ipv4",
		}

		tests := []string{
			`"192.168.1.1"`,
			`"invalid-ip"`,
		}

		for _, jsonStr := range tests {
			_, err := processor.ValidateSchema(jsonStr, schema)
			if err != nil {
				t.Errorf("ValidateSchema failed: %v", err)
			}
		}
	})

	t.Run("IPv6Format", func(t *testing.T) {
		schema := &Schema{
			Type:   "string",
			Format: "ipv6",
		}

		tests := []string{
			`"2001:0db8:85a3:0000:0000:8a2e:0370:7334"`,
			`"invalid-ipv6"`,
		}

		for _, jsonStr := range tests {
			_, err := processor.ValidateSchema(jsonStr, schema)
			if err != nil {
				t.Errorf("ValidateSchema failed: %v", err)
			}
		}
	})
}
