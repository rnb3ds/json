package json

import (
	"testing"
)

// ============================================================================
// CONSOLIDATED GETOR TESTS - Merged from multiple TestGetOr* functions
// ============================================================================

// TestGetTypedOrConsolidated tests all GetTypedOr variants in a single table-driven test
func TestGetTypedOrConsolidated(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		tests := []struct {
			name         string
			jsonStr      string
			path         string
			defaultValue string
			expected     string
		}{
			{"Found", `{"name":"Alice"}`, "name", "default", "Alice"},
			{"NotFound", `{"name":"Alice"}`, "missing", "default", "default"},
			{"Nested", `{"user":{"name":"Bob"}}`, "user.name", "default", "Bob"},
			{"EmptyString", `{"name":""}`, "name", "default", ""},
			{"InvalidJSON", `{invalid}`, "name", "default", "default"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := GetTypedOr[string](tt.jsonStr, tt.path, tt.defaultValue)
				if result != tt.expected {
					t.Errorf("GetTypedOr[string]() = %q, want %q", result, tt.expected)
				}
			})
		}
	})

	t.Run("Int", func(t *testing.T) {
		tests := []struct {
			name         string
			jsonStr      string
			path         string
			defaultValue int
			expected     int
		}{
			{"Found", `{"count":42}`, "count", -1, 42},
			{"NotFound", `{"count":42}`, "missing", -1, -1},
			{"Zero", `{"count":0}`, "count", -1, 0},
			{"Negative", `{"count":-10}`, "count", -1, -10},
			{"Nested", `{"data":{"value":100}}`, "data.value", 0, 100},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := GetTypedOr[int](tt.jsonStr, tt.path, tt.defaultValue)
				if result != tt.expected {
					t.Errorf("GetTypedOr[int]() = %d, want %d", result, tt.expected)
				}
			})
		}
	})

	t.Run("Float64", func(t *testing.T) {
		tests := []struct {
			name         string
			jsonStr      string
			path         string
			defaultValue float64
			expected     float64
		}{
			{"Found", `{"value":3.14}`, "value", -1.0, 3.14},
			{"NotFound", `{"value":3.14}`, "missing", -1.0, -1.0},
			{"IntToFloat", `{"count":42}`, "count", -1.0, 42.0},
			{"Zero", `{"value":0.0}`, "value", -1.0, 0.0},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := GetTypedOr[float64](tt.jsonStr, tt.path, tt.defaultValue)
				if result != tt.expected {
					t.Errorf("GetTypedOr[float64]() = %f, want %f", result, tt.expected)
				}
			})
		}
	})

	t.Run("Bool", func(t *testing.T) {
		tests := []struct {
			name         string
			jsonStr      string
			path         string
			defaultValue bool
			expected     bool
		}{
			{"FoundTrue", `{"active":true}`, "active", false, true},
			{"FoundFalse", `{"active":false}`, "active", true, false},
			{"NotFound", `{"active":true}`, "missing", false, false},
			{"NestedTrue", `{"config":{"enabled":true}}`, "config.enabled", false, true},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := GetTypedOr[bool](tt.jsonStr, tt.path, tt.defaultValue)
				if result != tt.expected {
					t.Errorf("GetTypedOr[bool]() = %v, want %v", result, tt.expected)
				}
			})
		}
	})

	t.Run("Slice", func(t *testing.T) {
		defaultArr := []any{"default"}
		tests := []struct {
			name     string
			jsonStr  string
			path     string
			expected []any
		}{
			{"Found", `{"items":[1,2,3]}`, "items", []any{1.0, 2.0, 3.0}},
			{"NotFound", `{"items":[1,2,3]}`, "missing", defaultArr},
			{"Empty", `{"items":[]}`, "items", []any{}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := GetTypedOr[[]any](tt.jsonStr, tt.path, defaultArr)
				if len(result) != len(tt.expected) {
					t.Errorf("GetTypedOr[[]any]() length = %d, want %d", len(result), len(tt.expected))
				}
			})
		}
	})

	t.Run("Map", func(t *testing.T) {
		defaultObj := map[string]any{"default": true}
		tests := []struct {
			name     string
			jsonStr  string
			path     string
			expected map[string]any
		}{
			{"Found", `{"config":{"theme":"dark"}}`, "config", map[string]any{"theme": "dark"}},
			{"NotFound", `{"config":{"theme":"dark"}}`, "missing", defaultObj},
			{"Empty", `{"config":{}}`, "config", map[string]any{}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := GetTypedOr[map[string]any](tt.jsonStr, tt.path, defaultObj)
				if len(result) != len(tt.expected) {
					t.Errorf("GetTypedOr[map]() length = %d, want %d", len(result), len(tt.expected))
				}
			})
		}
	})
}

// ============================================================================
// CONFIG EXTENSION METHODS TESTS - AddHook, AddValidator, AddDangerousPattern
// ============================================================================

// testHook is a simple Hook implementation for testing
type testHook struct {
	beforeCalled bool
	afterCalled  bool
}

func (h *testHook) Before(ctx HookContext) error {
	h.beforeCalled = true
	return nil
}

func (h *testHook) After(ctx HookContext, result any, err error) (any, error) {
	h.afterCalled = true
	return result, err
}

// testValidator is a simple Validator implementation for testing
type testValidator struct {
	validateCalled bool
}

func (v *testValidator) Validate(jsonStr string) error {
	v.validateCalled = true
	return nil
}

func TestConfigExtensionMethods(t *testing.T) {
	t.Run("AddHook", func(t *testing.T) {
		cfg := DefaultConfig()
		hook := &testHook{}

		cfg.AddHook(hook)

		if len(cfg.Hooks) != 1 {
			t.Errorf("AddHook() hooks count = %d, want 1", len(cfg.Hooks))
		}
	})

	t.Run("AddValidator", func(t *testing.T) {
		cfg := DefaultConfig()
		validator := &testValidator{}

		cfg.AddValidator(validator)

		if len(cfg.CustomValidators) != 1 {
			t.Errorf("AddValidator() validators count = %d, want 1", len(cfg.CustomValidators))
		}
	})

	t.Run("AddDangerousPattern", func(t *testing.T) {
		cfg := DefaultConfig()
		pattern := DangerousPattern{
			Pattern: "test_pattern",
			Name:    "Test Pattern",
			Level:   PatternLevelWarning,
		}

		cfg.AddDangerousPattern(pattern)

		if len(cfg.AdditionalDangerousPatterns) != 1 {
			t.Errorf("AddDangerousPattern() patterns count = %d, want 1", len(cfg.AdditionalDangerousPatterns))
		}
	})

	t.Run("MultipleHooks", func(t *testing.T) {
		cfg := DefaultConfig()
		hook1 := &testHook{}
		hook2 := &testHook{}

		cfg.AddHook(hook1)
		cfg.AddHook(hook2)

		if len(cfg.Hooks) != 2 {
			t.Errorf("AddHook() hooks count = %d, want 2", len(cfg.Hooks))
		}
	})
}

// ============================================================================
// PARSEDJSON DATA METHOD TEST
// ============================================================================

func TestParsedJSONDataMethod(t *testing.T) {
	processor, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer processor.Close()

	jsonStr := `{"name":"test","value":123}`
	parsed, err := processor.PreParse(jsonStr)
	if err != nil {
		t.Fatalf("PreParse() error = %v", err)
	}

	data := parsed.Data()
	if data == nil {
		t.Error("Data() returned nil")
		return
	}

	dataMap, ok := data.(map[string]any)
	if !ok {
		t.Errorf("Data() returned %T, want map[string]any", data)
		return
	}

	if dataMap["name"] != "test" {
		t.Errorf("Data()[name] = %v, want test", dataMap["name"])
	}
	if dataMap["value"] != 123.0 {
		t.Errorf("Data()[value] = %v, want 123", dataMap["value"])
	}
}

// ============================================================================
// ACCESSRESULT METHODS TESTS - Ok, Unwrap, UnwrapOr
// ============================================================================

func TestAccessResultMethods(t *testing.T) {
	t.Run("Ok", func(t *testing.T) {
		tests := []struct {
			name     string
			result   AccessResult
			expected bool
		}{
			{"Exists", AccessResult{Value: "test", Exists: true, Type: "string"}, true},
			{"NotExists", AccessResult{Value: nil, Exists: false, Type: ""}, false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := tt.result.Ok(); got != tt.expected {
					t.Errorf("AccessResult.Ok() = %v, want %v", got, tt.expected)
				}
			})
		}
	})

	t.Run("Unwrap", func(t *testing.T) {
		tests := []struct {
			name     string
			result   AccessResult
			expected any
		}{
			{"Exists", AccessResult{Value: "test", Exists: true, Type: "string"}, "test"},
			{"NotExists", AccessResult{Value: nil, Exists: false, Type: ""}, nil},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := tt.result.Unwrap(); got != tt.expected {
					t.Errorf("AccessResult.Unwrap() = %v, want %v", got, tt.expected)
				}
			})
		}
	})

	t.Run("UnwrapOr", func(t *testing.T) {
		tests := []struct {
			name         string
			result       AccessResult
			defaultValue any
			expected     any
		}{
			{"Exists", AccessResult{Value: "test", Exists: true, Type: "string"}, "default", "test"},
			{"NotExists", AccessResult{Value: nil, Exists: false, Type: ""}, "default", "default"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := tt.result.UnwrapOr(tt.defaultValue); got != tt.expected {
					t.Errorf("AccessResult.UnwrapOr() = %v, want %v", got, tt.expected)
				}
			})
		}
	})
}

// ============================================================================
// RESULT[T] METHODS TESTS - NewResult, Ok, Unwrap, UnwrapOr
// ============================================================================

func TestResultMethods(t *testing.T) {
	t.Run("NewResult", func(t *testing.T) {
		result := NewResult("test", true, nil)
		if result.Value != "test" || !result.Exists || result.Error != nil {
			t.Errorf("NewResult() = %+v, want {Value:test Exists:true Error:nil}", result)
		}
	})

	t.Run("Ok", func(t *testing.T) {
		tests := []struct {
			name     string
			result   Result[string]
			expected bool
		}{
			{"Success", Result[string]{Value: "test", Exists: true, Error: nil}, true},
			{"NoExists", Result[string]{Value: "", Exists: false, Error: nil}, false},
			{"Error", Result[string]{Value: "", Exists: true, Error: ErrPathNotFound}, false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := tt.result.Ok(); got != tt.expected {
					t.Errorf("Result.Ok() = %v, want %v", got, tt.expected)
				}
			})
		}
	})

	t.Run("Unwrap", func(t *testing.T) {
		tests := []struct {
			name     string
			result   Result[string]
			expected string
		}{
			{"Success", Result[string]{Value: "test", Exists: true, Error: nil}, "test"},
			{"NoExists", Result[string]{Value: "ignored", Exists: false, Error: nil}, ""},
			{"Error", Result[string]{Value: "ignored", Exists: true, Error: ErrPathNotFound}, ""},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := tt.result.Unwrap(); got != tt.expected {
					t.Errorf("Result.Unwrap() = %v, want %v", got, tt.expected)
				}
			})
		}
	})

	t.Run("UnwrapOr", func(t *testing.T) {
		tests := []struct {
			name         string
			result       Result[string]
			defaultValue string
			expected     string
		}{
			{"Success", Result[string]{Value: "test", Exists: true, Error: nil}, "default", "test"},
			{"NoExists", Result[string]{Value: "", Exists: false, Error: nil}, "default", "default"},
			{"Error", Result[string]{Value: "", Exists: true, Error: ErrPathNotFound}, "default", "default"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := tt.result.UnwrapOr(tt.defaultValue); got != tt.expected {
					t.Errorf("Result.UnwrapOr() = %v, want %v", got, tt.expected)
				}
			})
		}
	})
}

// ============================================================================
// SCHEMA WITH CONFIG TESTS
// ============================================================================

func TestSchemaWithConfig(t *testing.T) {
	t.Run("DefaultSchemaConfig", func(t *testing.T) {
		cfg := DefaultSchemaConfig()
		if cfg.MaxLength != nil && *cfg.MaxLength <= 0 {
			t.Errorf("DefaultSchemaConfig().MaxLength = %d, want > 0 or nil", *cfg.MaxLength)
		}
	})

	t.Run("NewSchemaWithConfig", func(t *testing.T) {
		maxLen := 1000
		cfg := SchemaConfig{
			Type:      "object",
			MaxLength: &maxLen,
		}
		schema := NewSchemaWithConfig(cfg)
		if schema == nil {
			t.Error("NewSchemaWithConfig() returned nil")
		}
	})

	t.Run("NewSchemaWithConfigDefaults", func(t *testing.T) {
		cfg := SchemaConfig{} // Zero values
		schema := NewSchemaWithConfig(cfg)
		if schema == nil {
			t.Error("NewSchemaWithConfig() returned nil for zero config")
		}
	})
}

// ============================================================================
// SECURITY PUBLIC API TESTS
// ============================================================================

func TestSecurityPublicAPI(t *testing.T) {
	// Clear any existing patterns first
	ClearDangerousPatterns()

	t.Run("RegisterAndListDangerousPatterns", func(t *testing.T) {
		pattern := DangerousPattern{
			Pattern: "test_dangerous",
			Name:    "Test Pattern",
			Level:   PatternLevelWarning,
		}

		RegisterDangerousPattern(pattern)
		patterns := ListDangerousPatterns()

		if len(patterns) != 1 {
			t.Errorf("ListDangerousPatterns() count = %d, want 1", len(patterns))
		}

		// Cleanup
		ClearDangerousPatterns()
	})

	t.Run("UnregisterDangerousPattern", func(t *testing.T) {
		pattern := DangerousPattern{
			Pattern: "to_remove",
			Name:    "Will be removed",
			Level:   PatternLevelInfo,
		}

		RegisterDangerousPattern(pattern)
		UnregisterDangerousPattern("to_remove")
		patterns := ListDangerousPatterns()

		if len(patterns) != 0 {
			t.Errorf("ListDangerousPatterns() after unregister count = %d, want 0", len(patterns))
		}
	})

	t.Run("GetDefaultPatterns", func(t *testing.T) {
		patterns := GetDefaultPatterns()
		if len(patterns) == 0 {
			t.Error("GetDefaultPatterns() returned empty slice")
		}

		// All default patterns should be critical level
		for _, p := range patterns {
			if p.Level != PatternLevelCritical {
				t.Errorf("Default pattern level = %v, want PatternLevelCritical", p.Level)
			}
		}
	})

	t.Run("GetCriticalPatterns", func(t *testing.T) {
		patterns := GetCriticalPatterns()
		if len(patterns) == 0 {
			t.Error("GetCriticalPatterns() returned empty slice")
		}
	})

	t.Run("ClearDangerousPatterns", func(t *testing.T) {
		RegisterDangerousPattern(DangerousPattern{
			Pattern: "clear_test",
			Name:    "Clear Test",
			Level:   PatternLevelInfo,
		})

		ClearDangerousPatterns()
		patterns := ListDangerousPatterns()

		if len(patterns) != 0 {
			t.Errorf("ListDangerousPatterns() after clear count = %d, want 0", len(patterns))
		}
	})
}

// ============================================================================
// BOUNDARY CONDITIONS TESTS
// ============================================================================

func TestBoundaryConditionsConsolidated(t *testing.T) {
	t.Run("EmptyInputs", func(t *testing.T) {
		processor, _ := New()
		defer processor.Close()

		// Empty JSON object
		_, err := processor.Get(`{}`, "any")
		if err == nil {
			t.Error("Get({}) should return error for missing path")
		}

		// Empty JSON array returns nil for out-of-bounds
		result, err := processor.Get(`[]`, "0")
		// Behavior: returns nil without error for missing index
		if err != nil {
			t.Logf("Get([])[0] error = %v", err)
		}
		if result != nil {
			t.Logf("Get([])[0] result = %v", result)
		}
	})

	t.Run("NullHandling", func(t *testing.T) {
		processor, _ := New()
		defer processor.Close()

		result, err := processor.Get(`{"value":null}`, "value")
		if err != nil {
			t.Errorf("Get(null) error = %v", err)
		}
		if result != nil {
			t.Errorf("Get(null) = %v, want nil", result)
		}
	})

	t.Run("VeryDeepNesting", func(t *testing.T) {
		processor, _ := New()
		defer processor.Close()

		// Build deeply nested JSON
		deepJSON := `{"a":{"b":{"c":{"d":{"e":"found"}}}}}`
		result, err := processor.Get(deepJSON, "a.b.c.d.e")
		if err != nil {
			t.Errorf("Get(deep) error = %v", err)
		}
		if result != "found" {
			t.Errorf("Get(deep) = %v, want found", result)
		}
	})

	t.Run("UnicodePaths", func(t *testing.T) {
		processor, _ := New()
		defer processor.Close()

		unicodeJSON := `{"名前":"太郎","年齢":25}`
		result, err := processor.Get(unicodeJSON, "名前")
		if err != nil {
			t.Errorf("Get(unicode) error = %v", err)
		}
		if result != "太郎" {
			t.Errorf("Get(unicode) = %v, want 太郎", result)
		}
	})

	t.Run("SpecialCharactersInPath", func(t *testing.T) {
		processor, _ := New()
		defer processor.Close()

		// Test dot in key name (escaped)
		specialJSON := `{"a.b":"value"}`
		result, err := processor.Get(specialJSON, "a\\.b")
		if err != nil {
			t.Logf("Get(escaped) error = %v (may be expected)", err)
		} else if result != "value" {
			t.Errorf("Get(escaped) = %v, want value", result)
		}
	})
}

// ============================================================================
// ARRAY EDGE CASES TESTS
// ============================================================================

func TestArrayEdgeCasesConsolidated(t *testing.T) {
	processor, _ := New()
	defer processor.Close()

	t.Run("NegativeIndex", func(t *testing.T) {
		// Array must be inside an object for proper indexing
		objJSON := `{"items":["a","b","c","d","e"]}`

		tests := []struct {
			path     string
			expected string
		}{
			{"items.-1", "e"},
			{"items.-2", "d"},
			{"items.-5", "a"},
			{"items.0", "a"},
			{"items.4", "e"},
		}

		for _, tt := range tests {
			result, err := processor.Get(objJSON, tt.path)
			if err != nil {
				t.Errorf("Get(%s) error = %v", tt.path, err)
				continue
			}
			if result != tt.expected {
				t.Errorf("Get(%s) = %v, want %v", tt.path, result, tt.expected)
			}
		}
	})

	t.Run("SliceOperations", func(t *testing.T) {
		objJSON := `{"items":["a","b","c","d","e"]}`

		tests := []struct {
			path        string
			expectCount int
		}{
			{"items.[0:2]", 2},
			{"items.[1:3]", 2},
			{"items.[:3]", 3},
			{"items.[2:]", 3},
		}

		for _, tt := range tests {
			result, err := processor.Get(objJSON, tt.path)
			if err != nil {
				t.Errorf("Get(%s) error = %v", tt.path, err)
				continue
			}
			arr, ok := result.([]any)
			if !ok {
				t.Errorf("Get(%s) returned %T, want []any", tt.path, result)
				continue
			}
			if len(arr) != tt.expectCount {
				t.Errorf("Get(%s) length = %d, want %d", tt.path, len(arr), tt.expectCount)
			}
		}
	})

	t.Run("OutOfBoundsIndex", func(t *testing.T) {
		objJSON := `{"items":["a","b","c"]}`

		// Out of bounds returns nil (no error)
		result, err := processor.Get(objJSON, "items.10")
		if err != nil {
			t.Logf("Get(items.10) error = %v", err)
		}
		if result != nil {
			t.Errorf("Get(items.10) = %v, want nil", result)
		}

		// Negative out of bounds returns nil
		result, err = processor.Get(objJSON, "items.-10")
		if err != nil {
			t.Logf("Get(items.-10) error = %v", err)
		}
		if result != nil {
			t.Errorf("Get(items.-10) = %v, want nil", result)
		}
	})
}

// ============================================================================
// TOP-LEVEL PARSE FUNCTION TEST
// ============================================================================

func TestTopLevelParse(t *testing.T) {
	t.Run("ValidJSON", func(t *testing.T) {
		result, err := Parse(`{"key":"value"}`)
		if err != nil {
			t.Errorf("Parse() error = %v", err)
		}
		if result == nil {
			t.Error("Parse() returned nil result")
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		_, err := Parse(`{invalid}`)
		if err == nil {
			t.Error("Parse() should return error for invalid JSON")
		}
	})

	t.Run("WithConfig", func(t *testing.T) {
		cfg := DefaultConfig()
		result, err := Parse(`{"key":"value"}`, cfg)
		if err != nil {
			t.Errorf("Parse() with config error = %v", err)
		}
		if result == nil {
			t.Error("Parse() with config returned nil result")
		}
	})
}
