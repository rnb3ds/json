package internal

import (
	"errors"
	"strings"
	"testing"
)

func TestSplitPathIntoSegments(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantLen int
	}{
		{"simple", "a.b.c", 3},
		{"single", "name", 1},
		{"empty", "", 0},
		{"trailing dot", "a.b.", 2},
		{"leading dot", ".a.b", 2},
		{"consecutive dots", "a..b", 2},
		{"escaped dot", `a\.b`, 1},
		{"escaped backslash", `a\\b`, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			segs := SplitPathIntoSegments(tt.path, nil)
			if len(segs) != tt.wantLen {
				t.Errorf("got %d segments, want %d: %+v", len(segs), tt.wantLen, segs)
			}
		})
	}
}

func TestNormalizePathSeparators(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"", ""},
		{"a.b", "a.b"},
		{"a..b", "a.b"},
		{"...a..b...", "a.b"},
		{".a.b.", "a.b"},
		{"a...b", "a.b"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := NormalizePathSeparators(tt.path); got != tt.want {
				t.Errorf("NormalizePathSeparators(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestIsValidPropertyName(t *testing.T) {
	tests := []struct {
		prop string
		want bool
	}{
		{"userName", true},
		{"", false},
		{"a.b", false},
		{"a[0]", false},
		{"a{b}", false},
		{"a(b)", false},
		{"user_name", true},
	}

	for _, tt := range tests {
		t.Run(tt.prop, func(t *testing.T) {
			if got := IsValidPropertyName(tt.prop); got != tt.want {
				t.Errorf("IsValidPropertyName(%q) = %v, want %v", tt.prop, got, tt.want)
			}
		})
	}
}

func TestIsValidArrayIndex(t *testing.T) {
	tests := []struct {
		idx  string
		want bool
	}{
		{"0", true},
		{"42", true},
		{"-1", true},
		{"", false},
		{"abc", false},
		{"1.5", false},
	}

	for _, tt := range tests {
		t.Run(tt.idx, func(t *testing.T) {
			if got := IsValidArrayIndex(tt.idx); got != tt.want {
				t.Errorf("IsValidArrayIndex(%q) = %v, want %v", tt.idx, got, tt.want)
			}
		})
	}
}

func TestIsValidSliceRange(t *testing.T) {
	tests := []struct {
		rng  string
		want bool
	}{
		{"1:3", true},
		{"1:3:2", true},
		{":3", true},
		{"1:", true},
		{"", false},
		{"1", false},
		{"1:2:3:4", false},
		{"a:b", false},
	}

	for _, tt := range tests {
		t.Run(tt.rng, func(t *testing.T) {
			if got := IsValidSliceRange(tt.rng); got != tt.want {
				t.Errorf("IsValidSliceRange(%q) = %v, want %v", tt.rng, got, tt.want)
			}
		})
	}
}

func TestIsArrayType(t *testing.T) {
	tests := []struct {
		val  any
		want bool
	}{
		{[]any{1, 2, 3}, true},
		{map[string]any{"a": 1}, false},
		{"hello", false},
		{nil, false},
		{42, false},
	}

	for i, tt := range tests {
		if got := IsArrayType(tt.val); got != tt.want {
			t.Errorf("IsArrayType[%d] = %v, want %v", i, got, tt.want)
		}
	}
}

func TestIsObjectType(t *testing.T) {
	tests := []struct {
		val  any
		want bool
	}{
		{map[string]any{"a": 1}, true},
		{map[any]any{"a": 1}, true},
		{[]any{1}, false},
		{nil, false},
		{"hello", false},
	}

	for i, tt := range tests {
		if got := IsObjectType(tt.val); got != tt.want {
			t.Errorf("IsObjectType[%d] = %v, want %v", i, got, tt.want)
		}
	}
}

func TestIsSliceType(t *testing.T) {
	tests := []struct {
		val  any
		want bool
	}{
		{[]any{1, 2}, true},
		{[]int{1, 2}, true},
		{[]string{"a"}, true},
		{nil, false},
		{"hello", false},
		{42, false},
	}

	for i, tt := range tests {
		if got := IsSliceType(tt.val); got != tt.want {
			t.Errorf("IsSliceType[%d](%T) = %v, want %v", i, tt.val, got, tt.want)
		}
	}
}

func TestWrapError(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		if got := WrapError(nil, "ctx"); got != nil {
			t.Error("WrapError(nil, ctx) should return nil")
		}
	})

	t.Run("wrap error", func(t *testing.T) {
		base := errors.New("base")
		wrapped := WrapError(base, "processing")
		if wrapped == nil {
			t.Fatal("expected non-nil error")
		}
		if !strings.Contains(wrapped.Error(), "processing") {
			t.Errorf("expected 'processing' in error, got: %v", wrapped)
		}
		if !strings.Contains(wrapped.Error(), "base") {
			t.Errorf("expected 'base' in error, got: %v", wrapped)
		}
	})
}

func TestCreatePathError(t *testing.T) {
	base := errors.New("not found")
	err := CreatePathError("a.b.c", "get", base)
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "get") || !strings.Contains(msg, "a.b.c") || !strings.Contains(msg, "not found") {
		t.Errorf("unexpected error message: %q", msg)
	}
}

func TestHasComplexSegments(t *testing.T) {
	tests := []struct {
		name     string
		segments []PathSegment
		want     bool
	}{
		{"empty", nil, false},
		{"property only", []PathSegment{{Type: PropertySegment, Key: "a"}}, false},
		{"array index", []PathSegment{{Type: ArrayIndexSegment, Index: 0}}, false},
		{"slice", []PathSegment{{Type: ArraySliceSegment}}, true},
		{"extract", []PathSegment{{Type: ExtractSegment, Key: "x"}}, true},
		{"mixed", []PathSegment{{Type: PropertySegment}, {Type: ArraySliceSegment}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasComplexSegments(tt.segments); got != tt.want {
				t.Errorf("HasComplexSegments() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconstructPath(t *testing.T) {
	tests := []struct {
		name string
		segs []PathSegment
		want string
	}{
		{"empty", nil, ""},
		{"single", []PathSegment{{Type: PropertySegment, Key: "a"}}, "a"},
		{"multiple", []PathSegment{
			{Type: PropertySegment, Key: "a"},
			{Type: PropertySegment, Key: "b"},
		}, "a.b"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ReconstructPath(tt.segs); got != tt.want {
				t.Errorf("ReconstructPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsValidCacheKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want bool
	}{
		{"valid", "user.name", true},
		{"empty", "", false},
		{"too long", strings.Repeat("a", MaxCacheKeyLength+1), false},
		{"at limit", strings.Repeat("a", MaxCacheKeyLength), true},
		{"control char", "key\x00name", false},
		{"tab", "key\tname", false},
		{"unicode", "user.名前", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidCacheKey(tt.key); got != tt.want {
				t.Errorf("IsValidCacheKey() = %v, want %v", got, tt.want)
			}
		})
	}
}
