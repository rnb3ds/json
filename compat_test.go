package json

import (
	"bytes"
	"strings"
	"testing"
)

// TestCompactBytesMethod tests CompactBytes method (encoding/json.Compact compatibility)
func TestCompactBytesMethod(t *testing.T) {
	prettyJSON := "{\"name\": \"Alice\", \"age\": 30}"

	processor, _ := New()
	defer processor.Close()

	var buf bytes.Buffer
	err := processor.CompactBytes(&buf, []byte(prettyJSON))
	if err != nil {
		t.Fatalf("CompactBytes error: %v", err)
	}

	result := buf.String()
	if strings.Contains(result, "\n") {
		t.Errorf("CompactBytes should remove newlines, got: %s", result)
	}
	if !strings.Contains(result, `"name"`) || !strings.Contains(result, `"Alice"`) {
		t.Errorf("CompactBytes lost data, got: %s", result)
	}
}

// TestIndentBytesMethod tests IndentBytes method (encoding/json.Indent compatibility)
func TestIndentBytesMethod(t *testing.T) {
	compactJSON := `{"name":"Alice","age":30}`

	processor, _ := New()
	defer processor.Close()

	var buf bytes.Buffer
	err := processor.IndentBytes(&buf, []byte(compactJSON), "", "  ")
	if err != nil {
		t.Fatalf("IndentBytes error: %v", err)
	}

	result := buf.String()
	if !strings.Contains(result, "\n") {
		t.Errorf("IndentBytes should add newlines, got: %s", result)
	}
	if !strings.Contains(result, "  ") {
		t.Errorf("IndentBytes should add indentation, got: %s", result)
	}
	if !strings.Contains(result, `"name"`) || !strings.Contains(result, `"Alice"`) {
		t.Errorf("IndentBytes lost data, got: %s", result)
	}
}

// TestCompactBytesEmptyInput tests CompactBytes with empty input
func TestCompactBytesEmptyInput(t *testing.T) {
	processor, _ := New()
	defer processor.Close()

	var buf bytes.Buffer
	err := processor.CompactBytes(&buf, []byte("{}"))
	if err != nil {
		t.Fatalf("CompactBytes error: %v", err)
	}

	result := buf.String()
	if result != "{}" {
		t.Errorf("CompactBytes should preserve empty object, got: %s", result)
	}
}

// TestIndentBytesEmptyInput tests IndentBytes with empty input
func TestIndentBytesEmptyInput(t *testing.T) {
	processor, _ := New()
	defer processor.Close()

	var buf bytes.Buffer
	err := processor.IndentBytes(&buf, []byte("{}"), "", "  ")
	if err != nil {
		t.Fatalf("IndentBytes error: %v", err)
	}

	result := buf.String()
	if !strings.Contains(result, "{") || !strings.Contains(result, "}") {
		t.Errorf("IndentBytes lost braces, got: %s", result)
	}
}
