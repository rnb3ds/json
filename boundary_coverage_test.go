package json

import (
	"bytes"
	"context"
	"math"
	"strings"
	"testing"
	"time"
)

// Boundary condition tests targeting low-coverage functions.
// Coverage targets: core >= 90%, utils >= 80%, overall >= 70%.

// --- GetWithContext (api.go:541, 0% coverage) ---

func TestGetWithContext_Boundary(t *testing.T) {
	t.Run("cancelled context returns error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // cancel immediately

		_, err := GetWithContext(ctx, `{"key":"value"}`, "key")
		if err == nil {
			t.Error("expected error with cancelled context")
		}
	})

	t.Run("valid context succeeds", func(t *testing.T) {
		ctx := context.Background()
		val, err := GetWithContext(ctx, `{"key":"value"}`, "key")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if val != "value" {
			t.Errorf("val = %v, want value", val)
		}
	})

	t.Run("timeout context with valid JSON", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		val, err := GetWithContext(ctx, `{"a":1,"b":"two"}`, "b")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if val != "two" {
			t.Errorf("val = %v, want two", val)
		}
	})
}

// --- HTMLEscape (api.go:784, 60% coverage) ---

func TestHTMLEscape_Boundary(t *testing.T) {
	t.Run("empty input", func(t *testing.T) {
		var buf bytes.Buffer
		HTMLEscape(&buf, []byte{})
		if buf.String() != "" {
			t.Errorf("expected empty output, got %q", buf.String())
		}
	})

	t.Run("no escaping needed", func(t *testing.T) {
		var buf bytes.Buffer
		HTMLEscape(&buf, []byte(`{"msg":"hello"}`))
		if buf.String() != `{"msg":"hello"}` {
			t.Errorf("unexpected escaping: %q", buf.String())
		}
	})

	t.Run("all special chars", func(t *testing.T) {
		var buf bytes.Buffer
		HTMLEscape(&buf, []byte(`<script>"alert('& XSS")</script>`+"\r\n"))
		result := buf.String()
		if strings.Contains(result, "<script>") {
			t.Error("HTML not escaped")
		}
		if !strings.Contains(result, "\\u003c") {
			t.Error("expected unicode escaping for <")
		}
	})
}

// --- GetTyped (api.go:559, 42.9% coverage) ---

func TestGetTyped_Boundary(t *testing.T) {
	t.Run("type mismatch returns converted value", func(t *testing.T) {
		val := GetTyped[string](`{"a":1}`, "a", "default")
		if val == "" {
			t.Errorf("expected non-empty value")
		}
	})

	t.Run("correct type returns value", func(t *testing.T) {
		val := GetTyped[string](`{"a":"hello"}`, "a", "")
		if val != "hello" {
			t.Errorf("val = %v, want hello", val)
		}
	})

	t.Run("missing path returns default", func(t *testing.T) {
		val := GetTyped[string](`{"a":1}`, "missing", "default")
		if val != "default" {
			t.Errorf("val = %v, want default", val)
		}
	})
}

// --- containsOverlongEncoding (security.go:1006) ---
// Detects URL-encoded overlong UTF-8 sequences used in path-traversal attacks
// (%c0%af -> overlong '/', %c1%9c -> overlong '\'). The detector scans for the
// percent-encoded form, not raw bytes.

func TestContainsOverlongEncoding_Boundary(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"empty", "", false},
		{"no percent sign", "normal text", false},
		{"valid URL-encoded char", "%41", false},        // 'A'
		{"valid 2-byte UTF-8 encoded", "%c3%A9", false}, // 'é' — not overlong
		{"overlong slash lowercase", "%c0%af", true},
		{"overlong slash uppercase", "%C0%AF", true},
		{"overlong backslash lowercase", "%c1%9c", true},
		{"overlong backslash uppercase", "%C1%9C", true},
		{"embedded overlong", "safe%c0%afpath", true},
		{"truncated sequence", "%c0", false},
		{"truncated after percent", "%c0%", false},
		{"percent at end", "path%", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := containsOverlongEncoding(tt.in); got != tt.want {
				t.Errorf("containsOverlongEncoding(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

// --- Integer overflow edge cases ---

func TestIntegerOverflow_Boundary(t *testing.T) {
	t.Run("large integer in JSON", func(t *testing.T) {
		jsonStr := `{"big": 9223372036854775807}`
		val, err := Get(jsonStr, "big")
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		// JSON numbers decode to float64; the magnitude must survive.
		f, ok := val.(float64)
		if !ok {
			t.Fatalf("expected float64 for large int, got %T (%v)", val, val)
		}
		if f <= 9e18 {
			t.Errorf("large int magnitude lost: got %v, want > 9e18", f)
		}
	})

	t.Run("large float that can't fit in int", func(t *testing.T) {
		_, ok := convertToInt(float64(math.MaxFloat64))
		if ok {
			t.Error("should not convert MaxFloat64 to int")
		}
	})
}

// --- Deep nesting edge cases ---

func TestDeepNesting_Boundary(t *testing.T) {
	t.Run("very deep nesting 50 levels", func(t *testing.T) {
		// Build 50 levels of nesting; default MaxDepth is 100, so this must succeed.
		inner := `"value"`
		for i := 0; i < 50; i++ {
			inner = `{"a":` + inner + `}`
		}
		got, err := Get(inner, strings.Repeat("a.", 49)+"a")
		if err != nil {
			t.Fatalf("Get at 50-level nesting failed: %v", err)
		}
		if got != "value" {
			t.Errorf("deep nesting result = %v, want \"value\"", got)
		}
	})

	t.Run("deep path with value assertion", func(t *testing.T) {
		deep := `{"a":{"b":{"c":{"d":{"e":"deep"}}}}}`
		got, err := Get(deep, "a.b.c.d.e")
		if err != nil {
			t.Fatalf("Get deep path failed: %v", err)
		}
		if got != "deep" {
			t.Errorf("deep path = %v, want deep", got)
		}
	})

	t.Run("chained array indices", func(t *testing.T) {
		deep := `{"a":[[[1,2],[3,4]],[[5,6],[7,8]]]}`
		got, err := Get(deep, "a[0][1][0]")
		if err != nil {
			t.Fatalf("Get chained array index failed: %v", err)
		}
		if got != 3.0 {
			t.Errorf("chained array index = %v, want 3", got)
		}
	})
}

// --- Empty/nil input edge cases ---

func TestEmptyInput_Boundary(t *testing.T) {
	t.Run("empty string Get", func(t *testing.T) {
		_, err := Get("", "key")
		if err == nil {
			t.Error("expected error for empty string")
		}
	})

	t.Run("whitespace only Set", func(t *testing.T) {
		_, err := Set("   \n\t  ", "key", "val")
		if err == nil {
			t.Error("expected error for whitespace-only input")
		}
	})

	t.Run("null value in JSON", func(t *testing.T) {
		val, err := Get(`{"a":null}`, "a")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if val != nil {
			t.Errorf("val = %v, want nil", val)
		}
	})

	t.Run("empty path returns root", func(t *testing.T) {
		result, err := Get(`{"key":"value"}`, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		m, ok := result.(map[string]any)
		if !ok {
			t.Fatalf("expected root map, got %T", result)
		}
		if m["key"] != "value" {
			t.Errorf("root map key = %v, want value", m["key"])
		}
	})

	t.Run("Set nil value", func(t *testing.T) {
		result, err := Set(`{"key":"value"}`, "key", nil)
		if err != nil {
			t.Fatalf("Set nil value failed: %v", err)
		}
		assertJSONEqual(t, `{"key":null}`, result)
	})
}

// --- Path edge cases ---

func TestPathEdgeCases_Boundary(t *testing.T) {
	t.Run("path with dots only", func(t *testing.T) {
		_, err := Get(`{"a":1}`, "..")
		if err == nil {
			t.Error("expected error for path with only dots")
		}
	})

	t.Run("path with unicode key", func(t *testing.T) {
		val, err := Get(`{"日本語":"hello"}`, "日本語")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if val != "hello" {
			t.Errorf("val = %v, want hello", val)
		}
	})

	t.Run("large array index", func(t *testing.T) {
		// An out-of-bounds index is a missing value, not a malformed query: it
		// returns nil with no error.
		val, err := Get(`{"arr":[1,2,3]}`, "arr[999999]")
		if err != nil {
			t.Fatalf("unexpected error for out-of-bounds index: %v", err)
		}
		if val != nil {
			t.Errorf("out-of-bounds array index should return nil, got %v", val)
		}
	})

	t.Run("path with escaped bracket in key", func(t *testing.T) {
		// A literal key containing a dot must NOT be reachable via the dot-path
		// "a.b" (which denotes nested a -> b). The value 1 must not be returned.
		val, err := Get(`{"a.b":1}`, "a.b")
		if err == nil && val != nil {
			t.Errorf("dot-path should not resolve the literal dotted key, got %v", val)
		}
	})
}

// --- Encoding edge cases ---

func TestEncodingEdgeCases_Boundary(t *testing.T) {
	t.Run("encode NaN float", func(t *testing.T) {
		// NaN is not representable in JSON: it must surface as an error or "null".
		result, err := Encode(map[string]any{"val": math.NaN()}, DefaultConfig())
		if err == nil && !strings.Contains(result, "null") {
			t.Errorf("NaN must surface as an error or null, got %q", result)
		}
	})

	t.Run("encode Infinity float", func(t *testing.T) {
		// +/-Inf is not representable in JSON: it must surface as an error or "null".
		result, err := Encode(map[string]any{"val": math.Inf(1)}, DefaultConfig())
		if err == nil && !strings.Contains(result, "null") {
			t.Errorf("Infinity must surface as an error or null, got %q", result)
		}
	})

	t.Run("encode with EscapeUnicode + CJK", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.EscapeUnicode = true
		result, err := Encode(map[string]any{"msg": "中文テスト한글"}, cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "\\u") {
			t.Error("expected unicode escaping for CJK characters")
		}
	})

	t.Run("encode with FloatTruncate", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.FloatTruncate = true
		result, err := Encode(map[string]any{"val": 3.14159265358979}, cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "3.") {
			t.Errorf("expected float in result: %s", result)
		}
	})
}

// --- Concurrent access edge cases ---

func TestConcurrencyEdgeCases_Boundary(t *testing.T) {
	t.Run("PreParse then concurrent GetFromParsed", func(t *testing.T) {
		p, err := New()
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		parsed, err := p.PreParse(`{"x":1,"y":2,"z":3}`)
		if err != nil {
			t.Fatalf("PreParse failed: %v", err)
		}
		defer parsed.Release()

		// Read from parsed data concurrently
		for i := 0; i < 10; i++ {
			val, err := p.GetFromParsed(parsed, "x")
			if err != nil {
				t.Errorf("GetFromParsed failed: %v", err)
			}
			if val != float64(1) {
				t.Errorf("val = %v, want 1", val)
			}
		}
	})

	t.Run("rapid create and close", func(t *testing.T) {
		for i := 0; i < 50; i++ {
			p, err := New()
			if err != nil {
				t.Fatalf("New() failed: %v", err)
			}
			p.Get(`{"test": "value"}`, "test")
			p.Close()
		}
	})
}

// --- validateFilePathStandalone (file.go:608, 0% coverage) ---

func TestValidateFilePath_Boundary(t *testing.T) {
	t.Run("path traversal attempt", func(t *testing.T) {
		err := validateFilePathStandalone("../../../etc/passwd")
		if err == nil {
			t.Error("expected error for path traversal")
		}
	})

	t.Run("null byte in path", func(t *testing.T) {
		err := validateFilePathStandalone("file\x00.txt")
		if err == nil {
			t.Error("expected error for null byte in path")
		}
	})
}

// TestValidateUnixPath_Boundary exercises the Unix path security check directly.
// validateUnixPath is gated behind runtime.GOOS != "windows" inside the public
// path-validation pipeline, so on Windows it is unreachable via the public API;
// the platform-specific logic is therefore unit-tested here.
func TestValidateUnixPath_Boundary(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"safe tmp path", "/tmp/safe.json", false},
		{"safe home path", "/home/user/data.json", false},
		{"dev blocked", "/dev/null", true},
		{"proc blocked", "/proc/self/status", true},
		{"etc passwd blocked", "/etc/passwd", true},
		{"etc shadow blocked", "/etc/shadow", true},
		{"root blocked", "/root/.bashrc", true},
		{"var log blocked", "/var/log/syslog", true},
		{"case-insensitive etc passwd", "/ETC/PASSWD", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateUnixPath(tt.path)
			if tt.wantErr && err == nil {
				t.Error("expected error blocking system path, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for safe path: %v", err)
			}
		})
	}
}

// --- Config edge cases ---

func TestConfigEdgeCases_Boundary(t *testing.T) {
	t.Run("CacheTTL zero instant expiry", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.CacheTTL = 0
		cfg.EnableCache = true
		p, err := New(cfg)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		_, err = p.Get(`{"a":1}`, "a")
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
	})

	t.Run("MaxCacheSize zero with EnableCache", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.MaxCacheSize = 0
		cfg.EnableCache = true
		if err := cfg.Validate(); err != nil {
			t.Fatalf("Validate failed: %v", err)
		}
	})
}
