package json

import (
	"fmt"
	"time"

	"github.com/cybergodev/json/internal"
)

// Set sets a value in JSON at the specified path
// Returns:
//   - On success: modified JSON string and nil error
//   - On failure: original unmodified JSON string and error information
func (p *Processor) Set(jsonStr, path string, value any, cfg ...Config) (result string, err error) {
	options, err := p.prepareOperation(jsonStr, path, cfg...)
	if err != nil {
		return jsonStr, err
	}
	// Release in reverse-acquire order: options first, then governance slot.
	// Defers run LIFO, so endGovernedOp (registered first) runs last.
	defer p.endGovernedOp()
	defer releaseConfig(options)

	// Run registered hooks around the operation. A Before hook may abort; an
	// After hook may observe or transform the result/error. Registered last so
	// it unwinds first (hooks see the raw result). snapshotHooks is nil in the
	// common no-hook case, so the whole block is skipped.
	hc := p.snapshotHooks()
	if len(hc) > 0 {
		hookCtx := HookContext{
			Operation: "set",
			JSONStr:   jsonStr,
			Path:      path,
			Value:     value,
			Config:    options,
			StartTime: time.Now(),
		}
		if hookErr := hc.executeBefore(hookCtx); hookErr != nil {
			return jsonStr, hookErr
		}
		defer func() {
			result, err = hc.executeAfterString(hookCtx, result, err)
		}()
	}

	data, err := p.parseJSON(jsonStr, "set", path, options)
	if err != nil {
		return jsonStr, err
	}

	// Note: We directly modify the parsed data without deep copy.
	// Parse() always creates fresh data via json.Unmarshal, so the cached
	// parse results from Get() are never shared with this scope.
	// If the operation fails, the original jsonStr string is returned unchanged.

	// Determine if we should create paths. A per-call Config (when supplied)
	// fully overrides the processor's setting — including disabling CreatePaths
	// when the processor default has it on. When no Config is supplied, the
	// processor's own setting applies (not the global default singleton, which
	// would silently re-enable CreatePaths on a processor built with it off).
	createPaths := p.config.CreatePaths
	if len(cfg) > 0 {
		createPaths = options.CreatePaths
	}

	// Set the value at the specified path
	err = p.setValueAtPathWithOptions(data, path, value, createPaths)
	if err != nil {
		// Return original data and detailed error information
		var setError *JsonsError
		if _, ok := err.(*rootDataTypeConversionError); ok && createPaths {
			setError = &JsonsError{
				Op:      "set",
				Path:    path,
				Message: fmt.Sprintf("root data type conversion failed: %v", err),
				Err:     err,
			}
		} else {
			setError = &JsonsError{
				Op:      "set",
				Path:    path,
				Message: fmt.Sprintf("set operation failed: %v", err),
				Err:     err,
			}
		}
		return jsonStr, setError
	}

	// Invalidate cached results for this JSON string since the data changed
	p.invalidateJSONCache(jsonStr)

	// Convert modified data back to JSON string
	// PERFORMANCE: Use FastMarshalToString instead of json.Marshal to avoid
	// double allocation (bytes -> string) and leverage optimized encoder pools
	result, err = internal.FastMarshalToString(data)
	if err != nil {
		// Return original data if marshaling fails
		return jsonStr, &JsonsError{
			Op:      "set",
			Path:    path,
			Message: "failed to marshal modified data",
			Err:     err,
		}
	}

	return result, nil
}

// SetMultiple sets multiple values in JSON using a map of path-value pairs
// Returns:
//   - On success: modified JSON string and nil error
//   - On failure: original unmodified JSON string and error information
func (p *Processor) SetMultiple(jsonStr string, updates map[string]any, cfg ...Config) (string, error) {
	// Concurrency governance first (includes the closed-check via beginGovernedOp),
	// matching Set/Delete ordering. Previously SetMultiple only did a single
	// checkClosed() at entry, so it was neither concurrency-limited nor drained
	// by Close() — an inconsistency with Set/Delete.
	if err := p.beginGovernedOp(); err != nil {
		return jsonStr, err
	}
	defer p.endGovernedOp()

	// Validate input
	if len(updates) == 0 {
		return jsonStr, nil // No updates to apply
	}

	// Prepare options
	options, err := p.prepareOptions(cfg...)
	if err != nil {
		return jsonStr, err
	}
	defer releaseConfig(options)

	// Validate JSON input. Honor SkipValidation (essential DoS checks only) to
	// stay consistent with Set/Delete, which route through validateOperationInput.
	// Previously this called validateInputForOptions unconditionally, silently
	// ignoring SkipValidation — a behavioral divergence from Set.
	if options.SkipValidation {
		if err := p.validateInputEssential(jsonStr); err != nil {
			return jsonStr, err
		}
	} else {
		if err := p.validateInputForOptions(jsonStr, options); err != nil {
			return jsonStr, err
		}
	}

	// Validate all paths before processing. Path SYNTAX is always validated —
	// matching validateOperationInput's policy — so a malformed index can never
	// silently corrupt data even under SkipValidation. Under SkipValidation the
	// cheaper syntax-only internal.ValidatePath is used, as in the single-path path.
	for path := range updates {
		var pathErr error
		if options.SkipValidation {
			pathErr = internal.ValidatePath(path)
		} else {
			pathErr = p.validatePath(path)
		}
		if pathErr != nil {
			return jsonStr, &JsonsError{
				Op:      "set_multiple",
				Path:    path,
				Message: fmt.Sprintf("invalid path '%s': %v", path, pathErr),
				Err:     pathErr,
			}
		}
	}

	// Parse JSON
	var data any
	err = p.Parse(jsonStr, &data, *options)
	if err != nil {
		return jsonStr, &JsonsError{
			Op:      "set_multiple",
			Message: fmt.Sprintf("failed to parse JSON: %v", err),
			Err:     err,
		}
	}

	// Create a deep copy of the data for modification attempts
	dataCopy, copyErr := deepCopy(data)
	if copyErr != nil {
		return jsonStr, &JsonsError{
			Op:      "set_multiple",
			Message: fmt.Sprintf("failed to create data copy: %v", copyErr),
			Err:     copyErr,
		}
	}

	// Determine if we should create paths. A per-call Config (when supplied)
	// fully overrides the processor's setting — including disabling CreatePaths
	// when the processor default has it on. When no Config is supplied, the
	// processor's own setting applies (not the global default singleton, which
	// would silently re-enable CreatePaths on a processor built with it off).
	createPaths := p.config.CreatePaths
	if len(cfg) > 0 {
		createPaths = options.CreatePaths
	}

	// Apply all updates on the copy
	var lastError error
	successCount := 0

	for path, value := range updates {
		err := p.setValueAtPathWithOptions(dataCopy, path, value, createPaths)
		if err != nil {
			// Handle root data type conversion errors
			if _, ok := err.(*rootDataTypeConversionError); ok && createPaths {
				lastError = &JsonsError{
					Op:      "set_multiple",
					Path:    path,
					Message: fmt.Sprintf("root data type conversion failed for path '%s': %v", path, err),
					Err:     err,
				}
				if !options.ContinueOnError {
					return jsonStr, lastError
				}
			} else {
				lastError = &JsonsError{
					Op:      "set_multiple",
					Path:    path,
					Message: fmt.Sprintf("failed to set path '%s': %v", path, err),
					Err:     err,
				}
				if !options.ContinueOnError {
					return jsonStr, lastError
				}
			}
		} else {
			successCount++
		}
	}

	// If no updates were successful and we have errors, return original data and error
	if successCount == 0 && lastError != nil {
		return jsonStr, &JsonsError{
			Op:      "set_multiple",
			Message: fmt.Sprintf("all %d updates failed, last error: %v", len(updates), lastError),
			Err:     lastError,
		}
	}

	// Invalidate cached results for this JSON string since the data changed.
	// Mirrors Set(): without this, a subsequent Get on the same jsonStr could be
	// served stale parse/get results from cache.
	p.invalidateJSONCache(jsonStr)

	// Convert modified data back to JSON string
	// PERFORMANCE: Use FastMarshalToString instead of json.Marshal
	result, err := internal.FastMarshalToString(dataCopy)
	if err != nil {
		// Return original data if marshaling fails
		return jsonStr, &JsonsError{
			Op:      "set_multiple",
			Message: "failed to marshal modified data",
			Err:     err,
		}
	}

	return result, nil
}

// SetCreate sets a value at the specified path, creating intermediate paths as
// needed. It is a convenience wrapper: SetCreate(s, p, v, cfg) is exactly
// Set(s, p, v, cfg') where cfg' is cfg with CreatePaths forced to true.
//
// Example:
//
//	result, err := processor.SetCreate(data, "users[0].profile.name", "Alice")
func (p *Processor) SetCreate(jsonStr, path string, value any, cfg ...Config) (string, error) {
	addOpts := mergeOptionsWithOverride(cfg, func(o *Config) {
		o.CreatePaths = true
	})
	return p.Set(jsonStr, path, value, addOpts)
}

// SetMultipleCreate sets multiple values, creating intermediate paths as needed.
// It is a convenience wrapper: SetMultipleCreate(s, u, cfg) is exactly
// SetMultiple(s, u, cfg') where cfg' is cfg with CreatePaths forced to true.
//
// Example:
//
//	result, err := processor.SetMultipleCreate(data, map[string]any{"user.name": "Alice", "user.age": 30})
func (p *Processor) SetMultipleCreate(jsonStr string, updates map[string]any, cfg ...Config) (string, error) {
	addOpts := mergeOptionsWithOverride(cfg, func(o *Config) {
		o.CreatePaths = true
	})
	return p.SetMultiple(jsonStr, updates, addOpts)
}
