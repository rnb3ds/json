package json

import (
	"testing"

	"github.com/cybergodev/json/internal"
)

// ============================================================================
// CONSOLIDATED GETDEFAULT TESTS
// ============================================================================

// TestGetOrConsolidated tests all GetTypedOr variants in one place
func TestGetOrConsolidated(t *testing.T) {
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
				result := GetTypedOr[[]any](tt.jsonStr, tt.path, def)
				tt.checkResult(t, result)
			case bool:
				result := GetTypedOr[bool](tt.jsonStr, tt.path, def)
				tt.checkResult(t, result)
			case float64:
				result := GetTypedOr[float64](tt.jsonStr, tt.path, def)
				tt.checkResult(t, result)
			case int:
				result := GetTypedOr[int](tt.jsonStr, tt.path, def)
				tt.checkResult(t, result)
			case string:
				result := GetTypedOr[string](tt.jsonStr, tt.path, def)
				tt.checkResult(t, result)
			case map[string]any:
				result := GetTypedOr[map[string]any](tt.jsonStr, tt.path, def)
				tt.checkResult(t, result)
			default:
				t.Fatalf("unsupported default type: %T", def)
			}
		})
	}
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
