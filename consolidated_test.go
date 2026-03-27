package json

import (
	"testing"

	"github.com/cybergodev/json/internal"
)

// ============================================================================
// CONSOLIDATED GETDEFAULT TESTS
// ============================================================================

// TestGetDefaultConsolidated tests all GetDefault variants in one place
func TestGetDefaultConsolidated(t *testing.T) {
	tests := []struct {
		name        string
		jsonStr     string
		path        string
		def         any
		checkResult func(t *testing.T, result any)
	}{
		// Slice tests
		{
			name:    "slice_existing",
			jsonStr: `{"items": [1, 2, 3]}`,
			path:    "items",
			def:     []any{"default"},
			checkResult: func(t *testing.T, result any) {
				arr := result.([]any)
				if len(arr) != 3 {
					t.Errorf("expected length 3, got %d", len(arr))
				}
			},
		},
		{
			name:    "slice_missing_returns_default",
			jsonStr: `{"items": [1, 2, 3]}`,
			path:    "missing",
			def:     []any{"default"},
			checkResult: func(t *testing.T, result any) {
				arr := result.([]any)
				if len(arr) != 1 || arr[0] != "default" {
					t.Errorf("expected default, got %v", arr)
				}
			},
		},

		// Bool tests
		{
			name:    "bool_existing_true",
			jsonStr: `{"enabled": true, "disabled": false}`,
			path:    "enabled",
			def:     false,
			checkResult: func(t *testing.T, result any) {
				if result != true {
					t.Errorf("expected true, got %v", result)
				}
			},
		},
		{
			name:    "bool_existing_false",
			jsonStr: `{"enabled": true, "disabled": false}`,
			path:    "disabled",
			def:     true,
			checkResult: func(t *testing.T, result any) {
				if result != false {
					t.Errorf("expected false, got %v", result)
				}
			},
		},
		{
			name:    "bool_missing_returns_default",
			jsonStr: `{"enabled": true}`,
			path:    "missing",
			def:     true,
			checkResult: func(t *testing.T, result any) {
				if result != true {
					t.Errorf("expected default true, got %v", result)
				}
			},
		},

		// Float64 tests
		{
			name:    "float64_existing",
			jsonStr: `{"price": 19.99, "count": 5}`,
			path:    "price",
			def:     0.0,
			checkResult: func(t *testing.T, result any) {
				if result != 19.99 {
					t.Errorf("expected 19.99, got %v", result)
				}
			},
		},
		{
			name:    "float64_int_converted",
			jsonStr: `{"price": 19.99, "count": 5}`,
			path:    "count",
			def:     0.0,
			checkResult: func(t *testing.T, result any) {
				if result != 5.0 {
					t.Errorf("expected 5.0, got %v", result)
				}
			},
		},
		{
			name:    "float64_missing_returns_default",
			jsonStr: `{"price": 19.99}`,
			path:    "missing",
			def:     99.99,
			checkResult: func(t *testing.T, result any) {
				if result != 99.99 {
					t.Errorf("expected 99.99, got %v", result)
				}
			},
		},

		// Map tests
		{
			name:    "map_existing",
			jsonStr: `{"nested": {"key": "value"}}`,
			path:    "nested",
			def:     map[string]any{"default": "value"},
			checkResult: func(t *testing.T, result any) {
				m := result.(map[string]any)
				if m["key"] != "value" {
					t.Errorf("expected key=value, got %v", m)
				}
			},
		},
		{
			name:    "map_missing_returns_default",
			jsonStr: `{"nested": {"key": "value"}}`,
			path:    "missing",
			def:     map[string]any{"default": "value"},
			checkResult: func(t *testing.T, result any) {
				m := result.(map[string]any)
				if m["default"] != "value" {
					t.Errorf("expected default map, got %v", m)
				}
			},
		},

		// String tests
		{
			name:    "string_existing",
			jsonStr: `{"name": "Alice"}`,
			path:    "name",
			def:     "default",
			checkResult: func(t *testing.T, result any) {
				if result != "Alice" {
					t.Errorf("expected Alice, got %v", result)
				}
			},
		},
		{
			name:    "string_missing_returns_default",
			jsonStr: `{"name": "Alice"}`,
			path:    "missing",
			def:     "default",
			checkResult: func(t *testing.T, result any) {
				if result != "default" {
					t.Errorf("expected default, got %v", result)
				}
			},
		},

		// Int tests
		{
			name:    "int_existing",
			jsonStr: `{"age": 30}`,
			path:    "age",
			def:     0,
			checkResult: func(t *testing.T, result any) {
				if result != 30 {
					t.Errorf("expected 30, got %v", result)
				}
			},
		},
		{
			name:    "int_missing_returns_default",
			jsonStr: `{"age": 30}`,
			path:    "missing",
			def:     99,
			checkResult: func(t *testing.T, result any) {
				if result != 99 {
					t.Errorf("expected 99, got %v", result)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch def := tt.def.(type) {
			case []any:
				result := GetDefault[[]any](tt.jsonStr, tt.path, def)
				tt.checkResult(t, result)
			case bool:
				result := GetDefault[bool](tt.jsonStr, tt.path, def)
				tt.checkResult(t, result)
			case float64:
				result := GetDefault[float64](tt.jsonStr, tt.path, def)
				tt.checkResult(t, result)
			case int:
				result := GetDefault[int](tt.jsonStr, tt.path, def)
				tt.checkResult(t, result)
			case string:
				result := GetDefault[string](tt.jsonStr, tt.path, def)
				tt.checkResult(t, result)
			case map[string]any:
				result := GetDefault[map[string]any](tt.jsonStr, tt.path, def)
				tt.checkResult(t, result)
			default:
				t.Fatalf("unsupported default type: %T", def)
			}
		})
	}
}

// ============================================================================
// CONSOLIDATED PRINT TESTS
// ============================================================================

// TestPrintFunctionsConsolidated tests all Print* functions
func TestPrintFunctionsConsolidated(t *testing.T) {
	t.Run("PrintE", func(t *testing.T) {
		tests := []struct {
			name    string
			data    any
			wantErr bool
		}{
			{"valid_data", `{"test": "value"}`, false},
			{"valid_map", map[string]any{"test": "value"}, false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := PrintE(tt.data)
				if (err != nil) != tt.wantErr {
					t.Errorf("PrintE() error = %v, wantErr %v", err, tt.wantErr)
				}
			})
		}
	})

	t.Run("PrintPrettyE", func(t *testing.T) {
		tests := []struct {
			name    string
			data    any
			wantErr bool
		}{
			{"valid_data", `{"test": "value"}`, false},
			{"valid_map", map[string]any{"test": "value"}, false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := PrintPrettyE(tt.data)
				if (err != nil) != tt.wantErr {
					t.Errorf("PrintPrettyE() error = %v, wantErr %v", err, tt.wantErr)
				}
			})
		}
	})

	t.Run("Print", func(t *testing.T) {
		// Print never returns error, just verify it doesn't panic
		Print(`{"test": "value"}`)
		Print(map[string]any{"test": "value"})
	})

	t.Run("PrintPretty", func(t *testing.T) {
		// PrintPretty never returns error, just verify it doesn't panic
		PrintPretty(`{"test": "value"}`)
		PrintPretty(map[string]any{"test": "value"})
	})
}

// ============================================================================
// CONSOLIDATED ERROR TESTS
// ============================================================================

// TestErrorTypesConsolidated tests all error type methods
func TestErrorTypesConsolidated(t *testing.T) {
	t.Run("JsonsError", func(t *testing.T) {
		tests := []struct {
			name        string
			err         *JsonsError
			checkString func(t *testing.T, s string)
		}{
			{
				name: "basic_error",
				err: &JsonsError{
					Op:      "Get",
					Path:    "test.path",
					Message: "test error",
					Err:     ErrOperationFailed,
				},
				checkString: func(t *testing.T, s string) {
					if s == "" {
						t.Error("Error() should not be empty")
					}
				},
			},
			{
				name: "error_with_wrapped",
				err: &JsonsError{
					Op:      "Set",
					Path:    "data.field",
					Message: "failed to set",
					Err:     ErrInvalidJSON,
				},
				checkString: func(t *testing.T, s string) {
					if s == "" {
						t.Error("Error() should not be empty")
					}
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				tt.checkString(t, tt.err.Error())
				unwrapped := tt.err.Unwrap()
				if unwrapped == nil {
					t.Error("Unwrap() should return non-nil")
				}
			})
		}
	})
}

// ============================================================================
// CONSOLIDATED CONFIG TESTS
// ============================================================================

// TestConfigConsolidated tests all Config-related functionality
func TestConfigConsolidated(t *testing.T) {
	t.Run("Clone", func(t *testing.T) {
		original := DefaultConfig()
		original.MaxJSONSize = 999

		cloned := original.Clone()
		if cloned.MaxJSONSize != 999 {
			t.Errorf("Clone() MaxJSONSize = %d, want 999", cloned.MaxJSONSize)
		}

		// Modify clone should not affect original
		cloned.MaxJSONSize = 1000
		if original.MaxJSONSize != 999 {
			t.Error("Modifying clone affected original")
		}
	})

	t.Run("Validate", func(t *testing.T) {
		tests := []struct {
			name        string
			config      *Config
			wantErr     bool
			checkResult func(t *testing.T, c *Config)
		}{
			{
				name:    "zero_values_get_defaults",
				config:  &Config{},
				wantErr: false,
				checkResult: func(t *testing.T, c *Config) {
					if c.MaxJSONSize <= 0 {
						t.Error("MaxJSONSize should be set to default")
					}
				},
			},
			{
				name:    "negative_cache_size",
				config:  &Config{MaxCacheSize: -1},
				wantErr: false,
				checkResult: func(t *testing.T, c *Config) {
					if c.MaxCacheSize != 0 {
						t.Errorf("Negative MaxCacheSize should be clamped to 0, got %d", c.MaxCacheSize)
					}
				},
			},
			{
				name:    "large_cache_size_clamped",
				config:  &Config{MaxCacheSize: 5000},
				wantErr: false,
				checkResult: func(t *testing.T, c *Config) {
					if c.MaxCacheSize > 2000 {
						t.Errorf("Large MaxCacheSize should be clamped to 2000, got %d", c.MaxCacheSize)
					}
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := tt.config.Validate()
				if (err != nil) != tt.wantErr {
					t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				}
				if !tt.wantErr && tt.checkResult != nil {
					tt.checkResult(t, tt.config)
				}
			})
		}
	})

	t.Run("AccessorMethods", func(t *testing.T) {
		c := Config{
			MaxJSONSize:  1000,
			MaxPathDepth: 50,
			EnableCache:  true,
			CacheTTL:     60000000000, // 1 minute
		}

		if c.GetMaxJSONSize() != 1000 {
			t.Errorf("GetMaxJSONSize() = %d, want 1000", c.GetMaxJSONSize())
		}
		if c.GetMaxPathDepth() != 50 {
			t.Errorf("GetMaxPathDepth() = %d, want 50", c.GetMaxPathDepth())
		}
		if c.EnableCache != true {
			t.Error("EnableCache should be true")
		}
		if c.CacheTTL != 60000000000 {
			t.Errorf("CacheTTL = %d, want 60000000000", c.CacheTTL)
		}
	})
}

// ============================================================================
// CONSOLIDATED PATH PARSING TESTS
// ============================================================================

// TestPathParsingConsolidated tests path parsing edge cases
func TestPathParsingConsolidated(t *testing.T) {
	processor := MustNew()
	defer processor.Close()

	tests := []struct {
		name        string
		path        string
		wantErr     bool
		checkResult func(t *testing.T, segments []internal.PathSegment)
	}{
		{
			name:    "simple_path",
			path:    "user.name",
			wantErr: false,
			checkResult: func(t *testing.T, segments []internal.PathSegment) {
				if len(segments) != 2 {
					t.Errorf("expected 2 segments, got %d", len(segments))
				}
			},
		},
		{
			name:    "array_access",
			path:    "users[0]",
			wantErr: false,
			checkResult: func(t *testing.T, segments []internal.PathSegment) {
				if len(segments) != 2 {
					t.Errorf("expected 2 segments, got %d", len(segments))
				}
				if segments[1].Type != internal.ArrayIndexSegment {
					t.Errorf("expected ArrayIndexSegment, got %v", segments[1].Type)
				}
			},
		},
		{
			name:    "slice_access",
			path:    "items[0:10]",
			wantErr: false,
			checkResult: func(t *testing.T, segments []internal.PathSegment) {
				if len(segments) != 2 {
					t.Errorf("expected 2 segments, got %d", len(segments))
				}
				if segments[1].Type != internal.ArraySliceSegment {
					t.Errorf("expected ArraySliceSegment, got %v", segments[1].Type)
				}
			},
		},
		{
			name:    "extraction",
			path:    "items{name,age}",
			wantErr: false,
			checkResult: func(t *testing.T, segments []internal.PathSegment) {
				if len(segments) != 2 {
					t.Errorf("expected 2 segments, got %d", len(segments))
				}
				if segments[1].Type != internal.ExtractSegment {
					t.Errorf("expected ExtractSegment, got %v", segments[1].Type)
				}
			},
		},
		{
			name:    "root_path",
			path:    ".",
			wantErr: false,
			checkResult: func(t *testing.T, segments []internal.PathSegment) {
				if len(segments) != 0 {
					t.Errorf("expected 0 segments for root path, got %d", len(segments))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			segments, err := processor.getCachedPathSegments(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("getCachedPathSegments() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.checkResult != nil {
				tt.checkResult(t, segments)
			}
		})
	}
}

// ============================================================================
// CONSOLIDATED SLICE OPERATION TESTS
// ============================================================================

// TestSliceOperationsConsolidated tests all slice operations
func TestSliceOperationsConsolidated(t *testing.T) {
	processor := MustNew()
	defer processor.Close()

	jsonStr := `{"items": [0, 1, 2, 3, 4, 5, 6, 7, 8, 9]}`

	tests := []struct {
		name        string
		path        string
		checkResult func(t *testing.T, result any)
	}{
		{
			name: "slice_from_start",
			path: "items[:5]",
			checkResult: func(t *testing.T, result any) {
				arr := result.([]any)
				if len(arr) != 5 {
					t.Errorf("expected length 5, got %d", len(arr))
				}
			},
		},
		{
			name: "slice_to_end",
			path: "items[5:]",
			checkResult: func(t *testing.T, result any) {
				arr := result.([]any)
				if len(arr) != 5 {
					t.Errorf("expected length 5, got %d", len(arr))
				}
			},
		},
		{
			name: "slice_middle",
			path: "items[3:7]",
			checkResult: func(t *testing.T, result any) {
				arr := result.([]any)
				if len(arr) != 4 {
					t.Errorf("expected length 4, got %d", len(arr))
				}
			},
		},
		{
			name: "slice_with_step",
			path: "items[::2]",
			checkResult: func(t *testing.T, result any) {
				arr := result.([]any)
				if len(arr) != 5 {
					t.Errorf("expected length 5 (every 2nd element), got %d", len(arr))
				}
			},
		},
		{
			name: "full_slice",
			path: "items[:]",
			checkResult: func(t *testing.T, result any) {
				arr := result.([]any)
				if len(arr) != 10 {
					t.Errorf("expected length 10, got %d", len(arr))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processor.Get(jsonStr, tt.path)
			if err != nil {
				t.Fatalf("Get() error = %v", err)
			}
			tt.checkResult(t, result)
		})
	}
}
