// Package json provides extension interfaces for customizing JSON processing behavior.
// These interfaces enable users to inject custom validation, encoding, and middleware logic.
package json

import (
	"reflect"
	"time"

	"github.com/cybergodev/json/internal"
)

// =============================================================================
// 1. Encoder Interfaces
// =============================================================================

// CustomEncoder provides custom JSON encoding capability.
// Implement this interface to replace the default encoder entirely.
//
// Example:
//
//	type UpperCaseEncoder struct{}
//	func (e *UpperCaseEncoder) Encode(value any) (string, error) {
//	    // Custom encoding logic
//	}
//
//	cfg := json.DefaultConfig()
//	cfg.CustomEncoder = &UpperCaseEncoder{}
type CustomEncoder interface {
	// Encode converts a Go value to JSON string.
	Encode(value any) (string, error)
}

// EncoderConfig provides configuration access for custom encoders.
// Implemented by Config struct.
type EncoderConfig interface {
	// HTML escaping
	IsHTMLEscapeEnabled() bool

	// Pretty printing
	IsPrettyEnabled() bool
	GetIndent() string
	GetPrefix() string

	// Key handling
	IsSortKeysEnabled() bool

	// Float handling
	GetFloatPrecision() int
	IsTruncateFloatEnabled() bool

	// Depth control
	GetMaxDepth() int

	// Null handling
	ShouldIncludeNulls() bool

	// UTF-8 validation
	ShouldValidateUTF8() bool

	// Unknown field handling
	IsDisallowUnknownEnabled() bool

	// Unicode escaping
	ShouldEscapeUnicode() bool

	// Slash escaping
	ShouldEscapeSlash() bool

	// Newline escaping
	ShouldEscapeNewlines() bool

	// Tab escaping
	ShouldEscapeTabs() bool
}

// TypeEncoder handles encoding for specific reflect.Types.
// Register via Config.CustomTypeEncoders map.
//
// Example:
//
//	type TimeEncoder struct{}
//	func (e *TimeEncoder) Encode(v reflect.Value) (string, error) {
//	    t := v.Interface().(time.Time)
//	    return `"` + t.Format(time.RFC3339) + `"`, nil
//	}
//
//	cfg := json.DefaultConfig()
//	cfg.CustomTypeEncoders = map[reflect.Type]json.TypeEncoder{
//	    reflect.TypeOf(time.Time{}): &TimeEncoder{},
//	}
type TypeEncoder interface {
	// Encode converts a specific type to its JSON representation.
	// Return the JSON string (including quotes for strings) or an error.
	Encode(v reflect.Value) (string, error)
}

// =============================================================================
// 2. Validator Interfaces
// =============================================================================

// Validator validates JSON input before processing.
// Implement this interface to add custom validation logic.
//
// Example:
//
//	type SizeValidator struct { MaxSize int64 }
//	func (v *SizeValidator) Validate(jsonStr string) error {
//	    if int64(len(jsonStr)) > v.MaxSize {
//	        return fmt.Errorf("JSON exceeds max size: %d", v.MaxSize)
//	    }
//	    return nil
//	}
type Validator interface {
	// Validate checks JSON string for issues.
	// Returns nil if valid, or an error describing the problem.
	Validate(jsonStr string) error
}

// PathValidator validates path syntax before navigation.
// Implement to add custom path validation rules.
type PathValidator interface {
	ValidatePath(path string) error
}

// SecurityValidator checks for dangerous patterns in content.
// Extends Validator with content-specific security checks.
type SecurityValidator interface {
	Validator
	// ValidateContent checks raw content for security issues.
	// This is called on the raw input before JSON parsing.
	ValidateContent(content string) error
}

// ValidationChain runs multiple validators in sequence.
// Stops at the first error encountered.
type ValidationChain []Validator

// Validate executes all validators in order, stopping at first error.
func (vc ValidationChain) Validate(jsonStr string) error {
	for _, v := range vc {
		if err := v.Validate(jsonStr); err != nil {
			return err
		}
	}
	return nil
}

// =============================================================================
// 3. Security Pattern Registration
// =============================================================================

// PatternLevel represents the severity level of a dangerous pattern.
type PatternLevel int

const (
	// PatternLevelCritical always blocks the operation.
	// Use for patterns that pose immediate security risks (e.g., prototype pollution).
	PatternLevelCritical PatternLevel = iota

	// PatternLevelWarning blocks in strict mode, logs warning in lenient mode.
	// Use for patterns that may indicate malicious intent but have legitimate uses.
	PatternLevelWarning

	// PatternLevelInfo logs but never blocks.
	// Use for audit/tracking purposes without interrupting operations.
	PatternLevelInfo
)

// String returns the string representation of PatternLevel.
func (pl PatternLevel) String() string {
	switch pl {
	case PatternLevelCritical:
		return "critical"
	case PatternLevelWarning:
		return "warning"
	case PatternLevelInfo:
		return "info"
	default:
		return "unknown"
	}
}

// DangerousPattern represents a security risk pattern to detect.
type DangerousPattern struct {
	// Pattern is the substring to detect in input.
	Pattern string

	// Name is a human-readable description of the security risk.
	Name string

	// Level determines how the pattern is handled.
	Level PatternLevel
}

// PatternRegistry manages dangerous patterns with thread-safe operations.
type PatternRegistry interface {
	// Add registers a new dangerous pattern.
	Add(pattern DangerousPattern)

	// Remove unregisters a pattern by its pattern string.
	Remove(pattern string)

	// List returns all registered patterns.
	List() []DangerousPattern

	// ListByLevel returns patterns filtered by severity level.
	ListByLevel(level PatternLevel) []DangerousPattern

	// Clear removes all registered patterns.
	Clear()
}

// =============================================================================
// 4. Path Segment Types (Public API)
// These are type aliases to internal types to avoid duplication.
// =============================================================================

// PathSegmentType identifies the type of a path segment.
// This is an alias to internal.PathSegmentType.
type PathSegmentType = internal.PathSegmentType

// Path segment type constants - aliases to internal package
const (
	// PathSegmentProperty represents object property access (e.g., "user.name").
	PathSegmentProperty PathSegmentType = internal.PropertySegment

	// PathSegmentArrayIndex represents array index access (e.g., "items[0]").
	PathSegmentArrayIndex PathSegmentType = internal.ArrayIndexSegment

	// PathSegmentArraySlice represents array slice (e.g., "items[1:5:2]").
	PathSegmentArraySlice PathSegmentType = internal.ArraySliceSegment

	// PathSegmentWildcard represents wildcard access (e.g., "items[*]").
	PathSegmentWildcard PathSegmentType = internal.WildcardSegment

	// PathSegmentExtract represents field extraction (e.g., "{name,email}").
	PathSegmentExtract PathSegmentType = internal.ExtractSegment

	// PathSegmentAppend represents append operation (e.g., "items[+]").
	PathSegmentAppend PathSegmentType = internal.AppendSegment
)

// PathSegmentFlags are bit flags for path segment options.
// This is an alias to internal.PathSegmentFlags.
type PathSegmentFlags = internal.PathSegmentFlags

// Path flag constants - aliases to internal package
const (
	// PathFlagNegative indicates a negative array index.
	PathFlagNegative PathSegmentFlags = internal.PathFlagNegative

	// PathFlagWildcard indicates a wildcard segment.
	PathFlagWildcard PathSegmentFlags = internal.PathFlagWildcard

	// PathFlagFlat indicates flat extraction mode.
	PathFlagFlat PathSegmentFlags = internal.PathFlagFlat

	// PathFlagHasStart indicates slice has start value.
	PathFlagHasStart PathSegmentFlags = internal.PathFlagHasStart

	// PathFlagHasEnd indicates slice has end value.
	PathFlagHasEnd PathSegmentFlags = internal.PathFlagHasEnd

	// PathFlagHasStep indicates slice has step value.
	PathFlagHasStep PathSegmentFlags = internal.PathFlagHasStep
)

// PathSegment represents a parsed path segment.
// This is an alias to internal.PathSegment.
// Methods like HasStart(), HasEnd(), etc. are available through the internal type.
type PathSegment = internal.PathSegment

// =============================================================================
// 5. Hook Interface - Unified Interception Mechanism
// =============================================================================

// HookContext provides context for operation hooks.
type HookContext struct {
	// Operation is the type of operation being performed.
	// Values: "get", "set", "delete", "marshal", "unmarshal"
	Operation string

	// JSONStr is the input JSON string (may be empty for marshal).
	JSONStr string

	// Path is the target path (may be empty for marshal/unmarshal).
	Path string

	// Value is the value for set operations.
	Value any

	// Config is the active configuration.
	Config *Config

	// StartTime is when the operation started (set before After is called).
	StartTime time.Time
}

// Hook intercepts operations before/after execution.
// Implement this interface to add cross-cutting concerns like logging,
// metrics, tracing, or request transformation.
//
// Example:
//
//	type LoggingHook struct{ logger *slog.Logger }
//
//	func (h *LoggingHook) Before(ctx json.HookContext) error {
//	    h.logger.Info("operation starting", "op", ctx.Operation, "path", ctx.Path)
//	    return nil
//	}
//
//	func (h *LoggingHook) After(ctx json.HookContext, result any, err error) (any, error) {
//	    h.logger.Info("operation completed", "op", ctx.Operation, "error", err)
//	    return result, err
//	}
type Hook interface {
	// Before is called before an operation.
	// Return error to abort the operation.
	Before(ctx HookContext) error

	// After is called after an operation completes.
	// Modify result or error as needed.
	After(ctx HookContext, result any, err error) (any, error)
}

// HookFunc is an adapter to use ordinary functions as Hooks.
// Useful for simple hooks that don't need both Before and After.
//
// Example:
//
//	// Only need After
//	p.AddHook(&json.HookFunc{
//	    AfterFn: func(ctx json.HookContext, result any, err error) (any, error) {
//	        log.Printf("%s completed in %v", ctx.Operation, time.Since(ctx.StartTime))
//	        return result, err
//	    },
//	})
type HookFunc struct {
	BeforeFn func(ctx HookContext) error
	AfterFn  func(ctx HookContext, result any, err error) (any, error)
}

// Before calls the BeforeFn if set, otherwise returns nil.
func (h *HookFunc) Before(ctx HookContext) error {
	if h.BeforeFn != nil {
		return h.BeforeFn(ctx)
	}
	return nil
}

// After calls the AfterFn if set, otherwise returns the original result.
func (h *HookFunc) After(ctx HookContext, result any, err error) (any, error) {
	if h.AfterFn != nil {
		return h.AfterFn(ctx, result, err)
	}
	return result, err
}

// =============================================================================
// Convenience Hook Implementations
// =============================================================================

// LoggingHook creates a hook that logs all operations.
// The logger must implement Info(msg string, args ...any).
//
// Example:
//
//	p.AddHook(json.LoggingHook(slog.Default()))
func LoggingHook(logger interface{ Info(msg string, args ...any) }) Hook {
	return &HookFunc{
		BeforeFn: func(ctx HookContext) error {
			logger.Info("operation starting", "op", ctx.Operation, "path", ctx.Path)
			return nil
		},
		AfterFn: func(ctx HookContext, result any, err error) (any, error) {
			logger.Info("operation completed",
				"op", ctx.Operation,
				"path", ctx.Path,
				"duration", time.Since(ctx.StartTime),
				"error", err)
			return result, err
		},
	}
}

// TimingHook creates a hook that records operation duration.
// The recorder must implement Record(op string, duration time.Duration).
//
// Example:
//
//	p.AddHook(json.TimingHook(myMetricsRecorder))
func TimingHook(recorder interface {
	Record(op string, duration time.Duration)
}) Hook {
	return &HookFunc{
		AfterFn: func(ctx HookContext, result any, err error) (any, error) {
			recorder.Record(ctx.Operation, time.Since(ctx.StartTime))
			return result, err
		},
	}
}

// ValidationHook creates a hook that validates input before operations.
// Return error from the validator to abort the operation.
//
// Example:
//
//	p.AddHook(json.ValidationHook(func(jsonStr, path string) error {
//	    if len(jsonStr) > 1_000_000 {
//	        return errors.New("JSON too large")
//	    }
//	    return nil
//	}))
func ValidationHook(validator func(jsonStr, path string) error) Hook {
	return &HookFunc{
		BeforeFn: func(ctx HookContext) error {
			return validator(ctx.JSONStr, ctx.Path)
		},
	}
}

// ErrorHook creates a hook that intercepts errors.
// The handler can transform errors or log them.
//
// Example:
//
//	p.AddHook(json.ErrorHook(func(ctx json.HookContext, err error) error {
//	    sentry.CaptureException(err)
//	    return err // return original or transformed error
//	}))
func ErrorHook(handler func(ctx HookContext, err error) error) Hook {
	return &HookFunc{
		AfterFn: func(ctx HookContext, result any, err error) (any, error) {
			if err != nil {
				return result, handler(ctx, err)
			}
			return result, nil
		},
	}
}

// HookChain manages multiple hooks for sequential execution.
type HookChain []Hook

// ExecuteBefore runs all Before hooks in order, stopping at first error.
func (hc HookChain) ExecuteBefore(ctx HookContext) error {
	for _, h := range hc {
		if err := h.Before(ctx); err != nil {
			return err
		}
	}
	return nil
}

// ExecuteAfter runs all After hooks in reverse order.
func (hc HookChain) ExecuteAfter(ctx HookContext, result any, err error) (any, error) {
	for i := len(hc) - 1; i >= 0; i-- {
		result, err = hc[i].After(ctx, result, err)
	}
	return result, err
}

// =============================================================================
// 6. Path Parser Interface (Breaks Circular Dependency)
// =============================================================================

// PathParser parses path strings into segments.
// Implement to provide custom path syntax support.
type PathParser interface {
	// ParsePath parses a path string into segments.
	ParsePath(path string) ([]PathSegment, error)
}

// CachedPathParser provides path parsing with caching.
type CachedPathParser interface {
	PathParser
	// ParsePathCached parses with caching for repeated paths.
	ParsePathCached(path string) ([]PathSegment, error)
	// ClearPathCache clears the path segment cache.
	ClearPathCache()
}

// =============================================================================
// Convenience Functions for Path Segments
// These delegate to internal package constructors.
// =============================================================================

// NewPropertySegment creates a property access segment.
func NewPropertySegment(key string) PathSegment {
	return internal.NewPropertySegment(key)
}

// NewArrayIndexSegment creates an array index access segment.
func NewArrayIndexSegment(index int) PathSegment {
	return internal.NewArrayIndexSegment(index)
}

// NewArraySliceSegment creates an array slice segment.
func NewArraySliceSegment(start, end, step int, hasStart, hasEnd, hasStep bool) PathSegment {
	return internal.NewArraySliceSegment(start, end, step, hasStart, hasEnd, hasStep)
}

// NewWildcardSegment creates a wildcard segment.
func NewWildcardSegment() PathSegment {
	return internal.NewWildcardSegment()
}

// NewExtractSegment creates an extraction segment.
func NewExtractSegment(key string, flat bool) PathSegment {
	return internal.NewExtractSegmentWithFlat(key, flat)
}

// NewAppendSegment creates an append segment.
func NewAppendSegment() PathSegment {
	return PathSegment{
		Type: PathSegmentAppend,
	}
}
