package json

import (
	"bytes"
	"strings"
	"testing"
)

// TestCompactBufferMethod tests CompactBuffer method (encoding/json.Compact compatibility)
func TestCompactBufferMethod(t *testing.T) {
	prettyJSON := "{\"name\": \"Alice\", \"age\": 30}"

	processor, _ := New()
	defer processor.Close()

	var buf bytes.Buffer
	err := processor.CompactBuffer(&buf, []byte(prettyJSON))
	if err != nil {
		t.Fatalf("CompactBuffer error: %v", err)
	}

	result := buf.String()
	if strings.Contains(result, "\n") {
		t.Errorf("CompactBuffer should remove newlines, got: %s", result)
	}
	if !strings.Contains(result, `"name"`) || !strings.Contains(result, `"Alice"`) {
		t.Errorf("CompactBuffer lost data, got: %s", result)
	}
}

// TestIndentBufferMethod tests IndentBuffer method (encoding/json.Indent compatibility)
func TestIndentBufferMethod(t *testing.T) {
	compactJSON := `{"name":"Alice","age":30}`

	processor, _ := New()
	defer processor.Close()

	var buf bytes.Buffer
	err := processor.IndentBuffer(&buf, []byte(compactJSON), "", "  ")
	if err != nil {
		t.Fatalf("IndentBuffer error: %v", err)
	}

	result := buf.String()
	if !strings.Contains(result, "\n") {
		t.Errorf("IndentBuffer should add newlines, got: %s", result)
	}
	if !strings.Contains(result, "  ") {
		t.Errorf("IndentBuffer should add indentation, got: %s", result)
	}
	if !strings.Contains(result, `"name"`) || !strings.Contains(result, `"Alice"`) {
		t.Errorf("IndentBuffer lost data, got: %s", result)
	}
}

// TestCompactBufferEmptyInput tests CompactBuffer with empty input
func TestCompactBufferEmptyInput(t *testing.T) {
	processor, _ := New()
	defer processor.Close()

	var buf bytes.Buffer
	err := processor.CompactBuffer(&buf, []byte("{}"))
	if err != nil {
		t.Fatalf("CompactBuffer error: %v", err)
	}

	result := buf.String()
	if result != "{}" {
		t.Errorf("CompactBuffer should preserve empty object, got: %s", result)
	}
}

// TestIndentBufferEmptyInput tests IndentBuffer with empty input
func TestIndentBufferEmptyInput(t *testing.T) {
	processor, _ := New()
	defer processor.Close()

	var buf bytes.Buffer
	err := processor.IndentBuffer(&buf, []byte("{}"), "", "  ")
	if err != nil {
		t.Fatalf("IndentBuffer error: %v", err)
	}

	result := buf.String()
	if !strings.Contains(result, "{") || !strings.Contains(result, "}") {
		t.Errorf("IndentBuffer lost braces, got: %s", result)
	}
}
