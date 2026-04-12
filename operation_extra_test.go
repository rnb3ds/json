package json

import (
	"testing"
)

// ============================================================================
// operation_extra_test.go - Tests for operation.go internal code paths
// exercised through public Get/Set/Delete API
// ============================================================================

// TestOperationExtra_GetArraySlice tests handleArraySlice via public Get API
func TestOperationExtra_GetArraySlice(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		path    string
		wantLen int
		wantErr bool
	}{
		{"basic slice", `{"items":[10,20,30,40,50]}`, "items[1:3]", 2, false},
		{"slice with step", `{"items":[10,20,30,40,50]}`, "items[::2]", 3, false},
		{"slice from end", `{"items":[10,20,30,40,50]}`, "items[3:]", 2, false},
		{"slice to end", `{"items":[10,20,30,40,50]}`, "items[:2]", 2, false},
		{"negative start", `{"items":[10,20,30,40,50]}`, "items[-2:]", 2, false},
		{"single element slice", `{"items":[10,20,30]}`, "items[1:2]", 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Get(tt.json, tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get(%q, %q) error = %v, wantErr %v", tt.json, tt.path, err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				arr, ok := result.([]any)
				if !ok {
					t.Fatalf("result is not []any, got %T", result)
				}
				if len(arr) != tt.wantLen {
					t.Errorf("result length = %d, want %d", len(arr), tt.wantLen)
				}
			}
		})
	}
}

// TestOperationExtra_DeleteArraySlice tests deleteArraySlice via public Delete API
func TestOperationExtra_DeleteArraySlice(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		path     string
		wantJSON string
	}{
		{"delete slice range", `{"items":[1,2,3,4,5]}`, "items[1:3]", `{"items":[1,4,5]}`},
		{"delete single index", `{"items":[1,2,3]}`, "items[0]", `{"items":[2,3]}`},
		{"delete last element", `{"items":[1,2,3]}`, "items[-1]", `{"items":[1,2]}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Delete(tt.json, tt.path)
			if err != nil {
				t.Fatalf("Delete(%q, %q) error: %v", tt.json, tt.path, err)
			}
			assertJSONEqual(t, tt.wantJSON, result)
		})
	}
}

// TestOperationExtra_DeleteNestedKeys tests deleting nested keys via dot notation
func TestOperationExtra_DeleteNestedKeys(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		path     string
		wantJSON string
	}{
		{
			"delete nested key",
			`{"data":{"x":1,"y":2}}`,
			"data.x",
			`{"data":{"y":2}}`,
		},
		{
			"delete from object in array using extraction",
			`{"items":[{"meta":{"x":1}},{"meta":{"x":2}}]}`,
			"items.{meta}.x",
			`{"items":[{"meta":{}},{"meta":{}}]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Delete(tt.json, tt.path)
			if err != nil {
				t.Fatalf("Delete(%q, %q) error: %v", tt.json, tt.path, err)
			}
			assertJSONEqual(t, tt.wantJSON, result)
		})
	}
}

// TestOperationExtra_DeleteComplexPaths tests deleteComplexSlice and related paths
func TestOperationExtra_DeleteComplexPaths(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		path    string
		wantErr bool
	}{
		{"delete nested array element", `{"a":{"b":[1,2,3]}}`, "a.b[1]", false},
		{"delete from nested slice", `{"items":[{"x":1},{"x":2},{"x":3}]}`, "items[0:2]", false},
		{"delete deep path", `{"a":{"b":{"c":"val"}}}`, "a.b.c", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Delete(tt.json, tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("Delete(%q, %q) error = %v, wantErr %v", tt.json, tt.path, err, tt.wantErr)
			}
		})
	}
}

// TestOperationExtra_SetArraySlice tests assignValueToSlice via public Set API
func TestOperationExtra_SetArraySlice(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		path     string
		value    any
		wantJSON string
	}{
		{"set slice range", `{"items":[1,2,3,4,5]}`, "items[1:3]", 99, `{"items":[1,99,99,4,5]}`},
		{"set slice with step", `{"items":[1,2,3,4,5]}`, "items[::2]", 0, `{"items":[0,2,0,4,0]}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Set(tt.json, tt.path, tt.value)
			if err != nil {
				t.Fatalf("Set(%q, %q, %v) error: %v", tt.json, tt.path, tt.value, err)
			}
			assertJSONEqual(t, tt.wantJSON, result)
		})
	}
}

// TestOperationExtra_SetArrayExtension tests handleArrayExtensionAndSet via Set with CreatePaths
func TestOperationExtra_SetArrayExtension(t *testing.T) {
	cfg := DefaultConfig()
	cfg.CreatePaths = true

	tests := []struct {
		name  string
		json  string
		path  string
		value any
	}{
		{"extend array by one", `{"items":[1,2]}`, "items[3]", "new"},
		{"extend array multiple", `{"items":[]}`, "items[2]", "x"},
		{"create nested path", `{}`, "a.b.c", "deep"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Set(tt.json, tt.path, tt.value, cfg)
			if err != nil {
				t.Errorf("Set(%q, %q, %v) with CreatePaths error: %v", tt.json, tt.path, tt.value, err)
			}
		})
	}
}

// TestOperationExtra_NegativeArrayIndex tests navigateToArrayIndexWithNegative via Get
func TestOperationExtra_NegativeArrayIndex(t *testing.T) {
	json := `{"items":[10,20,30,40,50]}`

	tests := []struct {
		name string
		path string
		want any
	}{
		{"last element", "items[-1]", float64(50)},
		{"second to last", "items[-2]", float64(40)},
		{"first via negative", "items[-5]", float64(10)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Get(json, tt.path)
			if err != nil {
				t.Fatalf("Get(%q, %q) error: %v", json, tt.path, err)
			}
			if result != tt.want {
				t.Errorf("Get(%q, %q) = %v, want %v", json, tt.path, result, tt.want)
			}
		})
	}
}

// TestOperationExtra_SetNegativeIndex tests setValueForArrayIndex with negative indices
func TestOperationExtra_SetNegativeIndex(t *testing.T) {
	json := `{"items":[10,20,30]}`
	result, err := Set(json, "items[-1]", 99)
	if err != nil {
		t.Fatalf("Set negative index error: %v", err)
	}
	assertJSONEqual(t, `{"items":[10,20,99]}`, result)
}

// TestOperationExtra_DeleteNegativeIndex tests deleteArrayElement with negative indices
func TestOperationExtra_DeleteNegativeIndex(t *testing.T) {
	json := `{"items":[10,20,30]}`
	result, err := Delete(json, "items[-1]")
	if err != nil {
		t.Fatalf("Delete negative index error: %v", err)
	}
	assertJSONEqual(t, `{"items":[10,20]}`, result)
}

// TestOperationExtra_StructAccess tests handleStructAccess via public Get API
func TestOperationExtra_StructAccess(t *testing.T) {
	type Person struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	p, _ := New()
	defer p.Close()

	encoded, err := p.Marshal(Person{Name: "Alice", Age: 30})
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	result, err := p.Get(string(encoded), "name")
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if result != "Alice" {
		t.Errorf("Get name = %v, want Alice", result)
	}
}

// TestOperationExtra_JSONPointerSet tests setValueJSONPointer code paths
func TestOperationExtra_JSONPointerSet(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		path     string
		value    any
		wantJSON string
	}{
		{"set new key", `{"a":1}`, "/b", 2, `{"a":1,"b":2}`},
		{"set nested", `{"a":{"b":1}}`, "/a/c", 2, `{"a":{"b":1,"c":2}}`},
		{"replace value", `{"a":1}`, "/a", 99, `{"a":99}`},
		{"escape tilde in key", `{"a/b":1}`, "/a~1b", 2, `{"a/b":2}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Set(tt.json, tt.path, tt.value)
			if err != nil {
				t.Fatalf("Set(%q, %q, %v) error: %v", tt.json, tt.path, tt.value, err)
			}
			assertJSONEqual(t, tt.wantJSON, result)
		})
	}
}

// TestOperationExtra_JSONPointerDelete tests deletion via JSON Pointer paths
func TestOperationExtra_JSONPointerDelete(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		path     string
		wantJSON string
	}{
		{"delete root key", `{"a":1,"b":2}`, "/a", `{"b":2}`},
		{"delete nested", `{"a":{"b":1,"c":2}}`, "/a/b", `{"a":{"c":2}}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Delete(tt.json, tt.path)
			if err != nil {
				t.Fatalf("Delete(%q, %q) error: %v", tt.json, tt.path, err)
			}
			assertJSONEqual(t, tt.wantJSON, result)
		})
	}
}

// TestOperationExtra_MultiFieldExtraction tests handleMultiFieldExtraction
func TestOperationExtra_MultiFieldExtraction(t *testing.T) {
	json := `{"items":[{"x":1,"y":2},{"x":3,"y":4}]}`

	result, err := Get(json, "items.{x}")
	if err != nil {
		t.Fatalf("Get extraction error: %v", err)
	}
	arr, ok := result.([]any)
	if !ok {
		t.Fatalf("result is not []any, got %T", result)
	}
	if len(arr) != 2 {
		t.Errorf("extraction result length = %d, want 2", len(arr))
	}
}

// TestOperationExtra_ConsecutiveExtractions tests consecutive extraction delete paths
func TestOperationExtra_ConsecutiveExtractions(t *testing.T) {
	json := `{"items":[{"meta":{"x":1}},{"meta":{"x":2}}]}`

	result, err := Delete(json, "items.{meta}.x")
	if err != nil {
		t.Fatalf("Delete consecutive extraction error: %v", err)
	}
	assertJSONEqual(t, `{"items":[{"meta":{}},{"meta":{}}]}`, result)
}

// TestOperationExtra_DeleteComplexSliceIntermediate tests deleteComplexSlice with intermediate segments
func TestOperationExtra_DeleteComplexSliceIntermediate(t *testing.T) {
	json := `{"items":[{"v":1},{"v":2},{"v":3},{"v":4}]}`

	result, err := Delete(json, "items[1:3]")
	if err != nil {
		t.Fatalf("Delete slice intermediate error: %v", err)
	}
	assertJSONEqual(t, `{"items":[{"v":1},{"v":4}]}`, result)
}

// TestOperationExtra_SetCreatePathInArray tests creating intermediate arrays via Set
// Note: creating a nested path inside a new array element (list[0].name) is not
// supported because the intermediate array element is nil and cannot be converted
// to a map. This test verifies the expected error behavior.
func TestOperationExtra_SetCreatePathInArray(t *testing.T) {
	cfg := DefaultConfig()
	cfg.CreatePaths = true

	_, err := Set(`{}`, "list[0].name", "first", cfg)
	if err == nil {
		t.Error("expected error for Set with nil intermediate array element, got nil")
	}
}

// TestOperationExtra_NavigateToParent tests navigateToParent through deep deletion
func TestOperationExtra_NavigateToParent(t *testing.T) {
	tests := []struct {
		name string
		json string
		path string
	}{
		{"deep nested", `{"a":{"b":{"c":"val"}}}`, "a.b.c"},
		{"array in path", `{"a":{"items":[1,2,3]}}`, "a.items"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Delete(tt.json, tt.path)
			if err != nil {
				t.Errorf("Delete(%q, %q) error: %v", tt.json, tt.path, err)
			}
		})
	}
}

// TestOperationExtra_DistributedArrayOps tests getValueWithDistributedOperation path
func TestOperationExtra_DistributedArrayOps(t *testing.T) {
	json := `{"matrix":[[1,2],[3,4],[5,6]]}`

	result, err := Get(json, "matrix[1][0]")
	if err != nil {
		t.Fatalf("Get distributed error: %v", err)
	}
	if result != float64(3) {
		t.Errorf("Get distributed = %v, want 3", result)
	}
}
