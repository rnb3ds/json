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

// TestIndentMethod tests Indent method (encoding/json.Indent compatibility)
func TestIndentMethod(t *testing.T) {
	compactJSON := `{"name":"Alice","age":30}`

	processor, _ := New()
	defer processor.Close()

	var buf bytes.Buffer
	err := processor.Indent(&buf, []byte(compactJSON), "", "  ")
	if err != nil {
		t.Fatalf("Indent error: %v", err)
	}

	result := buf.String()
	if !strings.Contains(result, "\n") {
		t.Errorf("Indent should add newlines, got: %s", result)
	}
	if !strings.Contains(result, "  ") {
		t.Errorf("Indent should add indentation, got: %s", result)
	}
	if !strings.Contains(result, `"name"`) || !strings.Contains(result, `"Alice"`) {
		t.Errorf("Indent lost data, got: %s", result)
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

// TestIndentEmptyInput tests Indent with empty input
func TestIndentEmptyInput(t *testing.T) {
	processor, _ := New()
	defer processor.Close()

	var buf bytes.Buffer
	err := processor.Indent(&buf, []byte("{}"), "", "  ")
	if err != nil {
		t.Fatalf("Indent error: %v", err)
	}

	result := buf.String()
	if !strings.Contains(result, "{") || !strings.Contains(result, "}") {
		t.Errorf("Indent lost braces, got: %s", result)
	}
}
