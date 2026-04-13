package json

import (
	"errors"
	"testing"
)

// TestIsRetryable verifies retryable error classification.
func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name  string
		input error
		want  bool
	}{
		{"nil error", nil, false},
		{"ErrOperationTimeout is retryable", ErrOperationTimeout, true},
		{"ErrConcurrencyLimit is retryable", ErrConcurrencyLimit, true},
		{"ErrInvalidJSON is not retryable", ErrInvalidJSON, false},
		{"ErrPathNotFound is not retryable", ErrPathNotFound, false},
		{"ErrTypeMismatch is not retryable", ErrTypeMismatch, false},
		{"errOperationFailed is not retryable", errOperationFailed, false},
		{"ErrInvalidPath is not retryable", ErrInvalidPath, false},
		{"ErrProcessorClosed is not retryable", ErrProcessorClosed, false},
		{"errInternalError is not retryable", errInternalError, false},
		{"ErrSizeLimit is not retryable", ErrSizeLimit, false},
		{"ErrDepthLimit is not retryable", ErrDepthLimit, false},
		{"ErrSecurityViolation is not retryable", ErrSecurityViolation, false},
		{"ErrUnsupportedPath is not retryable", ErrUnsupportedPath, false},
				{"errCacheDisabled is not retryable", errCacheDisabled, false},
		{"ErrResourceExhausted is not retryable", ErrResourceExhausted, false},
				{
			"concurrent_operation JsonsError is retryable",
			&JsonsError{Op: "concurrent_operation", Message: "too many ops", Err: ErrConcurrencyLimit},
			true,
		},
		{
			"other JsonsError Op is not retryable",
			&JsonsError{Op: "get", Message: "failed", Err: ErrPathNotFound},
			false,
		},
		{
			"wrapped ErrOperationTimeout is retryable",
			newOperationError("test", "timeout", ErrOperationTimeout),
			true,
		},
		{
			"plain sentinel wrapped via fmt.Errorf is not retryable",
			errors.New("some random error"),
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isRetryable(tt.input)
			if got != tt.want {
				t.Errorf("isRetryable() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIsUserError verifies user vs internal error classification.
func TestIsUserError(t *testing.T) {
	tests := []struct {
		name  string
		input error
		want  bool
	}{
		{"nil error", nil, false},
		{"ErrInvalidJSON is user error", ErrInvalidJSON, true},
		{"ErrPathNotFound is user error", ErrPathNotFound, true},
		{"ErrTypeMismatch is user error", ErrTypeMismatch, true},
		{"ErrInvalidPath is user error", ErrInvalidPath, true},
		{"ErrUnsupportedPath is user error", ErrUnsupportedPath, true},
		{"errOperationFailed is not user error", errOperationFailed, false},
		{"ErrProcessorClosed is not user error", ErrProcessorClosed, false},
		{"errInternalError is not user error", errInternalError, false},
		{"ErrSizeLimit is not user error", ErrSizeLimit, false},
		{"ErrDepthLimit is not user error", ErrDepthLimit, false},
		{"ErrConcurrencyLimit is not user error", ErrConcurrencyLimit, false},
		{"ErrSecurityViolation is not user error", ErrSecurityViolation, false},
				{"errCacheDisabled is not user error", errCacheDisabled, false},
		{"ErrOperationTimeout is not user error", ErrOperationTimeout, false},
		{"ErrResourceExhausted is not user error", ErrResourceExhausted, false},
		{
			"wrapped user error via JsonsError is user error",
			&JsonsError{Op: "get", Path: "data", Message: "bad path", Err: ErrInvalidPath},
			true,
		},
		{
			"wrapped non-user error is not user error",
			&JsonsError{Op: "get", Message: "failed", Err: errInternalError},
			false,
		},
		{
			"arbitrary error is not user error",
			errors.New("something else"),
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isUserError(tt.input)
			if got != tt.want {
				t.Errorf("isUserError() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetErrorSuggestion verifies suggestion generation for common errors.
func TestGetErrorSuggestion(t *testing.T) {
	tests := []struct {
		name  string
		input error
		want  string
	}{
		{"nil error returns empty", nil, ""},
		{
			"ErrInvalidJSON",
			ErrInvalidJSON,
			"Check JSON syntax - ensure proper quotes, brackets, and commas",
		},
		{
			"ErrPathNotFound",
			ErrPathNotFound,
			"Verify the path exists in the JSON structure",
		},
		{
			"ErrTypeMismatch",
			ErrTypeMismatch,
			"Check that the path points to the expected data type",
		},
		{
			"ErrInvalidPath",
			ErrInvalidPath,
			"Use valid path syntax: 'key.subkey', 'array[0]', or 'object{field}'",
		},
		{
			"ErrSizeLimit",
			ErrSizeLimit,
			"Reduce JSON size or increase MaxJSONSize in configuration",
		},
		{
			"ErrDepthLimit",
			ErrDepthLimit,
			"Reduce nesting depth or increase MaxNestingDepth in configuration",
		},
		{
			"ErrConcurrencyLimit",
			ErrConcurrencyLimit,
			"Reduce concurrent operations or increase MaxConcurrency in configuration",
		},
		{
			"ErrSecurityViolation",
			ErrSecurityViolation,
			"Input contains potentially dangerous patterns - review and sanitize",
		},
		{
			"unknown error returns generic suggestion",
			errors.New("something unexpected"),
			"Check the error message for specific details",
		},
		{
			"wrapped ErrInvalidJSON returns JSON suggestion",
			&JsonsError{Op: "parse", Message: "bad json", Err: ErrInvalidJSON},
			"Check JSON syntax - ensure proper quotes, brackets, and commas",
		},
		{
			"errOperationFailed returns generic suggestion",
			errOperationFailed,
			"Check the error message for specific details",
		},
		{
			"ErrProcessorClosed returns generic suggestion",
			ErrProcessorClosed,
			"Check the error message for specific details",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getErrorSuggestion(tt.input)
			if got != tt.want {
				t.Errorf("getErrorSuggestion() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestJsonsErrorIs verifies the Is method including nil receiver.
func TestJsonsErrorIs(t *testing.T) {
	tests := []struct {
		name   string
		receiver *JsonsError
		target error
		want    bool
	}{
		{
			name:     "nil receiver with nil target",
			receiver: nil,
			target:   nil,
			want:     true,
		},
		{
			name:     "nil receiver with non-nil target",
			receiver: nil,
			target:   ErrInvalidJSON,
			want:     false,
		},
		{
			name:     "non-nil receiver with nil target",
			receiver: &JsonsError{Op: "get", Message: "test"},
			target:   nil,
			want:     false,
		},
		{
			name:     "matching Op, Path, Err",
			receiver: &JsonsError{Op: "get", Path: "data", Err: ErrPathNotFound},
			target:   &JsonsError{Op: "get", Path: "data", Err: ErrPathNotFound},
			want:     true,
		},
		{
			name:     "mismatched Op",
			receiver: &JsonsError{Op: "get", Path: "data", Err: ErrPathNotFound},
			target:   &JsonsError{Op: "set", Path: "data", Err: ErrPathNotFound},
			want:     false,
		},
		{
			name:     "mismatched Path",
			receiver: &JsonsError{Op: "get", Path: "data", Err: ErrPathNotFound},
			target:   &JsonsError{Op: "get", Path: "other", Err: ErrPathNotFound},
			want:     false,
		},
		{
			name:     "mismatched Err",
			receiver: &JsonsError{Op: "get", Path: "data", Err: ErrPathNotFound},
			target:   &JsonsError{Op: "get", Path: "data", Err: ErrInvalidJSON},
			want:     false,
		},
		{
			name:     "nil Err fields match",
			receiver: &JsonsError{Op: "get", Path: "data"},
			target:   &JsonsError{Op: "get", Path: "data"},
			want:     true,
		},
		{
			name:     "one nil Err other non-nil Err",
			receiver: &JsonsError{Op: "get", Path: "data"},
			target:   &JsonsError{Op: "get", Path: "data", Err: ErrPathNotFound},
			want:     false,
		},
		{
			name:     "underlying sentinel matches via errors.Is",
			receiver: &JsonsError{Op: "get", Message: "not found", Err: ErrPathNotFound},
			target:   ErrPathNotFound,
			want:     true,
		},
		{
			name:     "underlying sentinel does not match",
			receiver: &JsonsError{Op: "get", Message: "not found", Err: ErrPathNotFound},
			target:   ErrInvalidJSON,
			want:     false,
		},
		{
			name:     "no underlying error, target is sentinel",
			receiver: &JsonsError{Op: "get", Message: "test"},
			target:   ErrInvalidJSON,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.receiver.Is(tt.target)
			if got != tt.want {
				t.Errorf("JsonsError.Is() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIsSecurityRelated verifies security vs non-security error classification.
func TestIsSecurityRelated(t *testing.T) {
	tests := []struct {
		name  string
		input error
		want  bool
	}{
		{"nil error", nil, false},
		{"ErrSecurityViolation is security related", ErrSecurityViolation, true},
		{"ErrInvalidJSON is not security related", ErrInvalidJSON, false},
		{"ErrPathNotFound is not security related", ErrPathNotFound, false},
		{"ErrTypeMismatch is not security related", ErrTypeMismatch, false},
		{"errOperationFailed is not security related", errOperationFailed, false},
		{"ErrInvalidPath is not security related", ErrInvalidPath, false},
		{"ErrProcessorClosed is not security related", ErrProcessorClosed, false},
		{"errInternalError is not security related", errInternalError, false},
		{"ErrSizeLimit is not security related", ErrSizeLimit, false},
		{"ErrDepthLimit is not security related", ErrDepthLimit, false},
		{"ErrConcurrencyLimit is not security related", ErrConcurrencyLimit, false},
		{"ErrUnsupportedPath is not security related", ErrUnsupportedPath, false},
				{"errCacheDisabled is not security related", errCacheDisabled, false},
		{"ErrOperationTimeout is not security related", ErrOperationTimeout, false},
		{"ErrResourceExhausted is not security related", ErrResourceExhausted, false},
		{
			"wrapped ErrSecurityViolation is security related",
			&JsonsError{Op: "validate", Message: "dangerous pattern", Err: ErrSecurityViolation},
			true,
		},
		{
			"newSecurityError result is security related",
			newSecurityError("parse", "prototype pollution"),
			true,
		},
		{
			"arbitrary error is not security related",
			errors.New("random error"),
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSecurityRelated(tt.input)
			if got != tt.want {
				t.Errorf("isSecurityRelated() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestSafeError verifies SafeError strips internal context from errors.
func TestSafeError(t *testing.T) {
	tests := []struct {
		name  string
		input error
		want  string
	}{
		{"nil returns empty", nil, ""},
		{"JsonsError returns sentinel message", &JsonsError{Op: "get", Path: "users.admin.password", Message: "not found", Err: ErrPathNotFound}, "path not found"},
		{"plain error returns full message", errors.New("something went wrong"), "something went wrong"},
		{"security error strips context", newSecurityError("parse", "dangerous input"), "security violation detected"},
		{"size limit error strips context", newSizeLimitError("load", 1<<30, 1<<20), "size limit exceeded"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SafeError(tt.input)
			if got != tt.want {
				t.Errorf("SafeError() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestRedactedPath verifies path redaction for safe logging.
func TestRedactedPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"empty returns empty", "", ""},
		{"short path returns masked", "users.name", "***"},
		{"exactly 32 returns masked", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "***"},
		{"long path returns truncated", "this.is.a.very.long.path.that.exceeds.thirty.two.characters", "this.is....aracters"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RedactedPath(tt.path)
			if got != tt.want {
				t.Errorf("RedactedPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}
