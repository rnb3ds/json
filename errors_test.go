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
		{"ErrOperationFailed is not retryable", ErrOperationFailed, false},
		{"ErrInvalidPath is not retryable", ErrInvalidPath, false},
		{"ErrProcessorClosed is not retryable", ErrProcessorClosed, false},
		{"ErrInternalError is not retryable", ErrInternalError, false},
		{"ErrSizeLimit is not retryable", ErrSizeLimit, false},
		{"ErrDepthLimit is not retryable", ErrDepthLimit, false},
		{"ErrSecurityViolation is not retryable", ErrSecurityViolation, false},
		{"ErrUnsupportedPath is not retryable", ErrUnsupportedPath, false},
		{"ErrCacheFull is not retryable", ErrCacheFull, false},
		{"ErrCacheDisabled is not retryable", ErrCacheDisabled, false},
		{"ErrResourceExhausted is not retryable", ErrResourceExhausted, false},
		{
			"cache_operation JsonsError is retryable",
			&JsonsError{Op: "cache_operation", Message: "cache miss", Err: ErrCacheFull},
			true,
		},
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
			wrapError(ErrOperationTimeout, "test", "timeout"),
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
		{"ErrOperationFailed is not user error", ErrOperationFailed, false},
		{"ErrProcessorClosed is not user error", ErrProcessorClosed, false},
		{"ErrInternalError is not user error", ErrInternalError, false},
		{"ErrSizeLimit is not user error", ErrSizeLimit, false},
		{"ErrDepthLimit is not user error", ErrDepthLimit, false},
		{"ErrConcurrencyLimit is not user error", ErrConcurrencyLimit, false},
		{"ErrSecurityViolation is not user error", ErrSecurityViolation, false},
		{"ErrCacheFull is not user error", ErrCacheFull, false},
		{"ErrCacheDisabled is not user error", ErrCacheDisabled, false},
		{"ErrOperationTimeout is not user error", ErrOperationTimeout, false},
		{"ErrResourceExhausted is not user error", ErrResourceExhausted, false},
		{
			"wrapped user error via JsonsError is user error",
			&JsonsError{Op: "get", Path: "data", Message: "bad path", Err: ErrInvalidPath},
			true,
		},
		{
			"wrapped non-user error is not user error",
			&JsonsError{Op: "get", Message: "failed", Err: ErrInternalError},
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
			"ErrOperationFailed returns generic suggestion",
			ErrOperationFailed,
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

// TestWrapError verifies error wrapping with context.
func TestWrapError(t *testing.T) {
	baseErr := errors.New("base error")
	innerErr := &JsonsError{Op: "inner", Message: "inner msg", Err: ErrPathNotFound}

	tests := []struct {
		name       string
		err        error
		op         string
		message    string
		wantNil    bool
		wantOp     string
		wantMsg    string
		wantUnwrap error
	}{
		{
			name:    "nil error returns nil",
			err:     nil,
			op:      "test",
			message: "should be nil",
			wantNil: true,
		},
		{
			name:       "wrap sentinel error",
			err:        ErrInvalidJSON,
			op:         "parse",
			message:    "failed to parse input",
			wantNil:    false,
			wantOp:     "parse",
			wantMsg:    "failed to parse input",
			wantUnwrap: ErrInvalidJSON,
		},
		{
			name:       "wrap arbitrary error",
			err:        baseErr,
			op:         "process",
			message:    "processing failed",
			wantNil:    false,
			wantOp:     "process",
			wantMsg:    "processing failed",
			wantUnwrap: baseErr,
		},
		{
			name:       "wrap JsonsError creates chain",
			err:        innerErr,
			op:         "outer",
			message:    "outer msg",
			wantNil:    false,
			wantOp:     "outer",
			wantMsg:    "outer msg",
			wantUnwrap: ErrPathNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wrapError(tt.err, tt.op, tt.message)
			if tt.wantNil {
				if got != nil {
					t.Errorf("wrapError() = %v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatalf("wrapError() returned nil, want non-nil")
			}
			je, ok := got.(*JsonsError)
			if !ok {
				t.Fatalf("wrapError() returned %T, want *JsonsError", got)
			}
			if je.Op != tt.wantOp {
				t.Errorf("Op = %q, want %q", je.Op, tt.wantOp)
			}
			if je.Message != tt.wantMsg {
				t.Errorf("Message = %q, want %q", je.Message, tt.wantMsg)
			}
			if tt.wantUnwrap != nil && !errors.Is(got, tt.wantUnwrap) {
				t.Errorf("Unwrap chain does not contain expected error; got Err = %v", je.Err)
			}
		})
	}
}

// TestWrapPathError verifies error wrapping with path context.
func TestWrapPathError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		op         string
		path       string
		message    string
		wantNil    bool
		wantOp     string
		wantPath   string
		wantMsg    string
		wantUnwrap error
	}{
		{
			name:    "nil error returns nil",
			err:     nil,
			op:      "get",
			path:    "data.field",
			message: "should be nil",
			wantNil: true,
		},
		{
			name:       "wrap with path",
			err:        ErrPathNotFound,
			op:         "get",
			path:       "users[0].name",
			message:    "field missing",
			wantNil:    false,
			wantOp:     "get",
			wantPath:   "users[0].name",
			wantMsg:    "field missing",
			wantUnwrap: ErrPathNotFound,
		},
		{
			name:       "wrap with empty path",
			err:        ErrInvalidJSON,
			op:         "validate",
			path:       "",
			message:    "bad input",
			wantNil:    false,
			wantOp:     "validate",
			wantPath:   "",
			wantMsg:    "bad input",
			wantUnwrap: ErrInvalidJSON,
		},
		{
			name:       "wrap chained error",
			err:        wrapError(ErrTypeMismatch, "convert", "cannot convert"),
			op:         "set",
			path:       "config.debug",
			message:    "type error during set",
			wantNil:    false,
			wantOp:     "set",
			wantPath:   "config.debug",
			wantMsg:    "type error during set",
			wantUnwrap: ErrTypeMismatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wrapPathError(tt.err, tt.op, tt.path, tt.message)
			if tt.wantNil {
				if got != nil {
					t.Errorf("wrapPathError() = %v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatalf("wrapPathError() returned nil, want non-nil")
			}
			je, ok := got.(*JsonsError)
			if !ok {
				t.Fatalf("wrapPathError() returned %T, want *JsonsError", got)
			}
			if je.Op != tt.wantOp {
				t.Errorf("Op = %q, want %q", je.Op, tt.wantOp)
			}
			if je.Path != tt.wantPath {
				t.Errorf("Path = %q, want %q", je.Path, tt.wantPath)
			}
			if je.Message != tt.wantMsg {
				t.Errorf("Message = %q, want %q", je.Message, tt.wantMsg)
			}
			if tt.wantUnwrap != nil && !errors.Is(got, tt.wantUnwrap) {
				t.Errorf("Unwrap chain does not contain expected error; got Err = %v", je.Err)
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
		{"ErrOperationFailed is not security related", ErrOperationFailed, false},
		{"ErrInvalidPath is not security related", ErrInvalidPath, false},
		{"ErrProcessorClosed is not security related", ErrProcessorClosed, false},
		{"ErrInternalError is not security related", ErrInternalError, false},
		{"ErrSizeLimit is not security related", ErrSizeLimit, false},
		{"ErrDepthLimit is not security related", ErrDepthLimit, false},
		{"ErrConcurrencyLimit is not security related", ErrConcurrencyLimit, false},
		{"ErrUnsupportedPath is not security related", ErrUnsupportedPath, false},
		{"ErrCacheFull is not security related", ErrCacheFull, false},
		{"ErrCacheDisabled is not security related", ErrCacheDisabled, false},
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
