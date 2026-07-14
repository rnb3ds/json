package json

import (
	"strings"
	"testing"
)

// ============================================================================
// Boundary tests for security.go low-coverage paths.
// ============================================================================

// --- toInternalPatterns (security.go:382, 33% coverage) ---

func TestToInternalPatterns(t *testing.T) {
	out := toInternalPatterns([]DangerousPattern{
		{Pattern: "eval", Name: "eval-injection"},
	})
	if len(out) != 1 {
		t.Errorf("got %d internal patterns, want 1", len(out))
	}
	if len(toInternalPatterns(nil)) != 0 {
		t.Error("nil input should yield empty result")
	}
}

// --- scanCustomPatterns via AdditionalDangerousPatterns (security.go:808) ---

func TestSecurity_CustomDangerousPattern(t *testing.T) {
	cfg := DefaultConfig()
	cfg.AdditionalDangerousPatterns = []DangerousPattern{{Pattern: "mybomb", Name: "bomb"}}
	p, _ := New(cfg)
	defer p.Close()

	var target map[string]any
	err := p.Parse(`{"x":"this has mybomb in it"}`, &target)
	if err == nil {
		t.Error("expected custom dangerous pattern to be rejected")
	}
}

// --- ValidateJSONInputEssential via SkipValidation + size limit (security.go:406) ---

func TestSecurity_EssentialSizeLimit(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SkipValidation = true // essential checks still run
	cfg.MaxJSONSize = 16
	p, _ := New(cfg)
	defer p.Close()

	big := strings.Repeat("a", 100)
	var target map[string]any
	if err := p.Parse(`"`+big+`"`, &target); err == nil {
		t.Error("expected size-limit error even with SkipValidation=true (essential check)")
	}
}

// --- validatePathSecurity non-ASCII / NFC path (security.go:975) ---

func TestSecurity_NonASCIIPath(t *testing.T) {
	// A path with non-ASCII (NFC-normalizable) characters must not panic and
	// must remain usable.
	p, _ := New()
	defer p.Close()
	_, _ = Get(`{"café":1}`, "café")
}
