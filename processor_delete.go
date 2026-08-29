package json

import (
	"errors"
	"time"

	"github.com/cybergodev/json/internal"
)

// Delete removes a value from JSON at the specified path
func (p *Processor) Delete(jsonStr, path string, cfg ...Config) (result string, err error) {
	options, err := p.prepareOperation(jsonStr, path, cfg...)
	if err != nil {
		// Return the original input on failure, matching every other error path
		// in this method and the contract documented by Set/SetMultiple.
		// Match Get's accounting: lifecycle rejections (closed processor,
		// concurrency limit) are not operation errors, but option and
		// input/path validation failures are.
		if !errors.Is(err, ErrProcessorClosed) && !errors.Is(err, ErrConcurrencyLimit) {
			p.incrementErrorCount()
		}
		return jsonStr, err
	}
	// Release in reverse-acquire order: options first, then governance slot.
	defer p.endGovernedOp()
	defer releaseConfig(options)

	// Count the operation for stats — see Set for the rationale (mutations
	// previously went unreported, undercounting GetStats). Error returns below
	// increment the error counter, as Get does.
	p.incrementOperationCount()

	// Run registered hooks around the operation. A Before hook may abort; an
	// After hook may observe or transform the result/error. Registered last so
	// it unwinds first (hooks see the raw result). snapshotHooks is nil in the
	// common no-hook case, so the whole block is skipped.
	hc := p.snapshotHooks()
	if len(hc) > 0 {
		hookCtx := HookContext{
			Operation: "delete",
			JSONStr:   jsonStr,
			Path:      path,
			Config:    options,
			StartTime: time.Now(),
		}
		if hookErr := hc.executeBefore(hookCtx); hookErr != nil {
			p.incrementErrorCount()
			return jsonStr, hookErr
		}
		defer func() {
			result, err = hc.executeAfterString(hookCtx, result, err)
		}()
	}

	// Determine cleanup options from prepared options and config
	cleanupNulls := options.CleanupNulls || p.config.CleanupNulls
	compactArrays := options.CompactArrays || p.config.CompactArrays

	// PERFORMANCE: Fast path for simple property delete without cache or cleanup.
	// compactArrays implies cleanupNulls below (empty arrays are compacted during
	// reconstruction), so it must also opt out of this fast path.
	if isSimplePropertyAccess(path) && !p.config.EnableCache && len(cfg) == 0 && !cleanupNulls && !compactArrays {
		m, isObj, err := unmarshalRootObject(jsonStr)
		if err != nil {
			p.incrementErrorCount()
			return jsonStr, newOperationPathError("delete", path, err.Error(), ErrInvalidJSON)
		}
		if isObj {
			if _, exists := m[path]; !exists {
				p.incrementErrorCount()
				return jsonStr, newOperationPathError("delete", path, "path not found", ErrPathNotFound)
			}
			delete(m, path)
			result, err := internal.FastMarshalToString(m)
			if err != nil {
				p.incrementErrorCount()
				return jsonStr, newOperationPathError("delete", path, "failed to marshal result", err)
			}
			return result, nil
		}
		// Not an object — fall through to full recursive processor
	}

	// Parse JSON using unified helper
	data, err := p.parseJSON(jsonStr, "delete", path, options)
	if err != nil {
		p.incrementErrorCount()
		return jsonStr, err
	}

	// If compactArrays is enabled, automatically enable cleanupNulls
	if compactArrays {
		cleanupNulls = true
	}

	// Check if path contains array access - only then we need DeletedMarker cleanup
	needsMarkerCleanup := p.isArrayDeletePath(path)

	// Delete the value at the specified path
	err = p.deleteValueAtPath(data, path)
	if err != nil {
		p.incrementErrorCount()
		return jsonStr, &JsonsError{
			Op:      "delete",
			Path:    path,
			Message: err.Error(),
			Err:     err,
		}
	}

	// Only clean up deleted markers if the path involved array operations
	if needsMarkerCleanup {
		data = p.cleanupDeletedMarkers(data)
	}

	// Invalidate cached results for this JSON string since the data changed
	p.invalidateJSONCache(jsonStr)

	// Cleanup nulls if requested
	if cleanupNulls {
		data = p.cleanupNullValuesWithReconstruction(data, compactArrays)
	}

	// Convert back to JSON string
	result, err = internal.FastMarshalToString(data)
	if err != nil {
		p.incrementErrorCount()
		return jsonStr, &JsonsError{
			Op:      "delete",
			Path:    path,
			Message: "failed to marshal result",
			Err:     err,
		}
	}

	return result, nil
}

// isArrayDeletePath checks if the path involves array operations that require marker cleanup
func (p *Processor) isArrayDeletePath(path string) bool {
	for i := 0; i < len(path); i++ {
		if path[i] == '[' {
			return true
		}
	}
	return false
}

// DeleteClean removes a value from JSON and cleans up the resulting null
// placeholders and empty array slots. It is a convenience wrapper:
// DeleteClean(s, p, cfg) is exactly Delete(s, p, cfg') where cfg' is cfg with
// CleanupNulls and CompactArrays forced to true.
func (p *Processor) DeleteClean(jsonStr, path string, cfg ...Config) (string, error) {
	cleanupOpts := p.mergeOptionsWithOverride(cfg, func(o *Config) {
		o.CleanupNulls = true
		o.CompactArrays = true
	})
	return p.Delete(jsonStr, path, cleanupOpts)
}
