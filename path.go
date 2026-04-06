package json

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/cybergodev/json/internal"
)

// isPrimitiveType checks if data is a JSON primitive type
func (p *Processor) isPrimitiveType(data any) bool {
	switch data.(type) {
	case string, int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64, bool:
		return true
	default:
		return false
	}
}

// Parse parses a JSON string into the provided target with improved error handling
func (p *Processor) Parse(jsonStr string, target any, opts ...Config) error {
	if err := p.checkClosed(); err != nil {
		return err
	}

	options, err := p.prepareOptions(opts...)
	if err != nil {
		return err
	}

	if err := p.validateInput(jsonStr); err != nil {
		return err
	}

	if target == nil {
		return &JsonsError{
			Op:      "parse",
			Message: "target cannot be nil, use Parse for any type result",
			Err:     ErrOperationFailed,
		}
	}

	// Parse with number preservation to maintain original format
	if options.PreserveNumbers {
		// Use numberPreservingDecoder to keep json.Number as-is
		decoder := newNumberPreservingDecoder(true)
		data, err := decoder.DecodeToAny(jsonStr)
		if err != nil {
			return &JsonsError{
				Op:      "parse",
				Message: fmt.Sprintf("invalid JSON for target type %T: %v", target, err),
				Err:     ErrInvalidJSON,
			}
		}

		// For *any type, directly assign the result
		if anyPtr, ok := target.(*any); ok {
			*anyPtr = data
			return nil
		}

		// For other types, use custom encoder/decoder to preserve numbers
		config := PrettyConfig()
		config.PreserveNumbers = true

		encoder := newCustomEncoder(config)
		defer encoder.Close()

		encodedJson, err := encoder.Encode(data)
		if err != nil {
			return &JsonsError{
				Op:      "parse",
				Message: fmt.Sprintf("failed to encode data for target type %T: %v", target, err),
				Err:     ErrOperationFailed,
			}
		}

		// Use number-preserving unmarshal for final conversion
		if err := preservingUnmarshal(stringToBytes(encodedJson), target, true); err != nil {
			return &JsonsError{
				Op:      "parse",
				Message: fmt.Sprintf("invalid JSON for target type %T: %v", target, err),
				Err:     ErrInvalidJSON,
			}
		}
	} else {
		// Standard parsing without number preservation
		if err := preservingUnmarshal(stringToBytes(jsonStr), target, false); err != nil {
			return &JsonsError{
				Op:      "parse",
				Message: fmt.Sprintf("invalid JSON for target type %T: %v", target, err),
				Err:     ErrInvalidJSON,
			}
		}
	}

	return nil
}

// ParseAny parses a JSON string and returns the result as any.
// This method provides the same behavior as the package-level Parse function.
// Use Parse when you need to unmarshal into a specific target type.
//
// Example:
//
//	data, err := processor.ParseAny(`{"name": "Alice"}`)
func (p *Processor) ParseAny(jsonStr string, opts ...Config) (any, error) {
	if err := p.checkClosed(); err != nil {
		return nil, err
	}

	var data any
	if err := p.Parse(jsonStr, &data, opts...); err != nil {
		return nil, err
	}
	return data, nil
}

// Valid validates JSON format without parsing the entire structure
func (p *Processor) Valid(jsonStr string, opts ...Config) (bool, error) {
	if err := p.checkClosed(); err != nil {
		return false, err
	}

	if err := p.validateInput(jsonStr); err != nil {
		return false, err
	}

	// Prepare options
	options, err := p.prepareOptions(opts...)
	if err != nil {
		return false, err
	}

	// Check cache first
	cacheKey := p.createCacheKey("validate", jsonStr, "", options)
	if cached, ok := p.getCachedResult(cacheKey); ok {
		return cached.(bool), nil
	}

	// Valid JSON by attempting to parse
	decoder := newNumberPreservingDecoder(options.PreserveNumbers)
	_, err = decoder.DecodeToAny(jsonStr)

	if err != nil {
		// Return error for invalid JSON
		return false, &JsonsError{
			Op:      "validate",
			Message: fmt.Sprintf("invalid JSON: %v", err),
			Err:     ErrInvalidJSON,
		}
	}

	// Cache result if enabled
	p.setCachedResult(cacheKey, true, options)

	return true, nil
}

// ValidBytes validates JSON format from byte slice (matches encoding/json.Valid signature)
// This method provides compatibility with the standard library's json.Valid function
func (p *Processor) ValidBytes(data []byte) bool {
	jsonStr := string(data)
	valid, err := p.Valid(jsonStr)
	return err == nil && valid
}

// stringToBytes converts string to []byte efficiently
// Using standard conversion for safety and compatibility
// While unsafe.StringData could provide zero-copy conversion,
// we prioritize safety over marginal performance gains
func stringToBytes(s string) []byte {
	return internal.StringToBytes(s)
}

func (p *Processor) splitPath(path string, segments []internal.PathSegment) []internal.PathSegment {
	segments = segments[:0]

	// Direct call to internal package - reduces method call overhead
	if !internal.NeedsPathPreprocessing(path) {
		return internal.SplitPathIntoSegments(path, segments)
	}

	sb := p.getStringBuilder()
	defer p.putStringBuilder(sb)

	processedPath := internal.PreprocessPath(path, sb)
	return internal.SplitPathIntoSegments(processedPath, segments)
}

// preprocessPath preprocesses a path string (exported for testing)
func (p *Processor) preprocessPath(path string, sb *strings.Builder) string {
	return internal.PreprocessPath(path, sb)
}

func (p *Processor) parsePath(path string) ([]string, error) {
	if path == "" {
		return []string{}, nil
	}

	if !p.isComplexPath(path) {
		return strings.Split(path, "."), nil
	}

	segments := p.getPathSegments()
	defer p.putPathSegments(segments)

	segments = p.splitPath(path, segments)

	result := make([]string, len(segments))
	for i, segment := range segments {
		result[i] = segment.String()
	}

	return result, nil
}

func (p *Processor) isDistributedOperationPath(path string) bool {
	return internal.IsExtractionPath(path)
}

func (p *Processor) isDistributedOperationSegment(segment internal.PathSegment) bool {
	return internal.IsExtractionSegment(segment)
}

func (p *Processor) handleDistributedOperation(data any, segments []internal.PathSegment) (any, error) {
	return p.getValueWithDistributedOperation(data, p.reconstructPath(segments))
}

func (p *Processor) reconstructPath(segments []internal.PathSegment) string {
	return internal.ReconstructPath(segments)
}

// parseArraySegment parses array access segments like [0], [1:3], etc.
func (p *Processor) parseArraySegment(part string, segments []internal.PathSegment) []internal.PathSegment {
	return internal.ParseArraySegment(part, segments)
}

// parseExtractionSegment parses extraction segments like {key}, {flat:key}, etc.
func (p *Processor) parseExtractionSegment(part string, segments []internal.PathSegment) []internal.PathSegment {
	return internal.ParseExtractionSegment(part, segments)
}

// Prettify formats JSON string with indentation.
// This is the recommended method for formatting JSON strings.
func (p *Processor) Prettify(jsonStr string, opts ...Config) (string, error) {
	if err := p.checkClosed(); err != nil {
		return "", err
	}

	options, err := p.prepareOptions(opts...)
	if err != nil {
		return "", err
	}

	if err := p.validateInput(jsonStr); err != nil {
		return "", err
	}

	// Check cache first
	cacheKey := p.createCacheKey("pretty", jsonStr, "", options)
	if cached, ok := p.getCachedResult(cacheKey); ok {
		return cached.(string), nil
	}

	// Parse with number preservation to maintain original number types
	decoder := newNumberPreservingDecoder(options.PreserveNumbers)
	data, err := decoder.DecodeToAny(jsonStr)
	if err != nil {
		return "", &JsonsError{
			Op:      "pretty",
			Message: fmt.Sprintf("failed to parse JSON: %v", err),
			Err:     ErrInvalidJSON,
		}
	}

	// Use custom encoder with pretty formatting to preserve number types
	config := PrettyConfig()
	config.PreserveNumbers = options.PreserveNumbers
	// Respect caller's Indent/Prefix if explicitly provided via options
	if options.Indent != "" {
		config.Indent = options.Indent
	}
	if options.Prefix != "" {
		config.Prefix = options.Prefix
	}

	encoder := newCustomEncoder(config)
	defer encoder.Close()

	result, err := encoder.Encode(data)
	if err != nil {
		return "", &JsonsError{
			Op:      "pretty",
			Message: fmt.Sprintf("failed to format JSON: %v", err),
			Err:     ErrOperationFailed,
		}
	}

	// Cache result if enabled
	p.setCachedResult(cacheKey, result, options)

	return result, nil
}

// Compact removes whitespace from JSON string
func (p *Processor) Compact(jsonStr string, opts ...Config) (string, error) {
	if err := p.checkClosed(); err != nil {
		return "", err
	}

	options, err := p.prepareOptions(opts...)
	if err != nil {
		return "", err
	}

	if err := p.validateInput(jsonStr); err != nil {
		return "", err
	}

	// Check cache first
	cacheKey := p.createCacheKey("compact", jsonStr, "", options)
	if cached, ok := p.getCachedResult(cacheKey); ok {
		return cached.(string), nil
	}

	// Parse with number preservation to maintain original number types
	decoder := newNumberPreservingDecoder(options.PreserveNumbers)
	data, err := decoder.DecodeToAny(jsonStr)
	if err != nil {
		return "", &JsonsError{
			Op:      "compact",
			Message: fmt.Sprintf("failed to parse JSON: %v", err),
			Err:     ErrInvalidJSON,
		}
	}

	// Use custom encoder with compact formatting to preserve number types
	config := DefaultConfig()
	config.PreserveNumbers = options.PreserveNumbers

	encoder := newCustomEncoder(config)
	defer encoder.Close()

	result, err := encoder.Encode(data)
	if err != nil {
		return "", &JsonsError{
			Op:      "compact",
			Message: fmt.Sprintf("failed to compact JSON: %v", err),
			Err:     ErrOperationFailed,
		}
	}

	// Cache result if enabled
	p.setCachedResult(cacheKey, result, options)

	return result, nil
}

// FormatCompact removes whitespace from JSON string (alias for Compact)
func (p *Processor) FormatCompact(jsonStr string, opts ...Config) (string, error) {
	return p.Compact(jsonStr, opts...)
}

// CompactString removes whitespace from JSON string.
// This is an alias for Compact for consistency with the package-level CompactString function.
func (p *Processor) CompactString(jsonStr string, opts ...Config) (string, error) {
	return p.Compact(jsonStr, opts...)
}

// CompactBuffer appends to dst the JSON-encoded src with insignificant space characters elided.
// Compatible with encoding/json.Compact with optional Config support.
func (p *Processor) CompactBuffer(dst *bytes.Buffer, src []byte, opts ...Config) error {
	compacted, err := p.Compact(string(src), opts...)
	if err != nil {
		return err
	}
	_, err = dst.WriteString(compacted)
	return err
}

// IndentBuffer appends to dst an indented form of the JSON-encoded src.
// Compatible with encoding/json.Indent with optional Config support.
func (p *Processor) IndentBuffer(dst *bytes.Buffer, src []byte, prefix, indent string, opts ...Config) error {
	var data any
	if err := p.Unmarshal(src, &data, opts...); err != nil {
		return err
	}
	indented, err := p.MarshalIndent(data, prefix, indent, opts...)
	if err != nil {
		return err
	}
	_, err = dst.Write(indented)
	return err
}

// CompactBytes appends to dst the JSON-encoded src with insignificant space characters elided.
// This is an alias for CompactBuffer without optional Config, providing encoding/json.Compact compatibility.
// Use this method when you need the exact encoding/json.Compact signature.
//
// Example:
//
//	var buf bytes.Buffer
//	err := processor.CompactBytes(&buf, []byte(`{"name": "Alice"}`))
func (p *Processor) CompactBytes(dst *bytes.Buffer, src []byte) error {
	return p.CompactBuffer(dst, src)
}

// IndentBytes appends to dst an indented form of the JSON-encoded src.
// This is an alias for IndentBuffer without optional Config, providing encoding/json.Indent compatibility.
// Use this method when you need the exact encoding/json.Indent signature.
//
// Example:
//
//	var buf bytes.Buffer
//	err := processor.IndentBytes(&buf, []byte(`{"name":"Alice"}`), "", "  ")
func (p *Processor) IndentBytes(dst *bytes.Buffer, src []byte, prefix, indent string) error {
	return p.IndentBuffer(dst, src, prefix, indent)
}

// HTMLEscapeBuffer appends to dst the JSON-encoded src with HTML-safe escaping.
// Replaces &, <, and > with \u0026, \u003c, and \u003e for safe HTML embedding.
// Compatible with encoding/json.HTMLEscape with optional Config support.
func (p *Processor) HTMLEscapeBuffer(dst *bytes.Buffer, src []byte, opts ...Config) {
	var data any
	if err := p.Unmarshal(src, &data, opts...); err != nil {
		dst.Write(src)
		return
	}

	config := DefaultConfig()
	if len(opts) > 0 {
		config = opts[0]
	}
	config.EscapeHTML = true
	escaped, err := p.EncodeWithConfig(data, config)
	if err != nil {
		dst.Write(src)
		return
	}

	dst.WriteString(escaped)
}

func (p *Processor) navigateToPath(data any, path string) (any, error) {
	if path == "" || path == "." || path == "/" {
		return data, nil
	}

	if strings.HasPrefix(path, "/") {
		return p.navigateJSONPointer(data, path)
	}

	return p.navigateDotNotation(data, path)
}

func (p *Processor) navigateDotNotation(data any, path string) (any, error) {
	current := data

	segments := p.getPathSegments()
	defer p.putPathSegments(segments)

	segments = p.splitPath(path, segments)

	for i := 0; i < len(segments); i++ {
		segment := segments[i]
		if p.isDistributedOperationSegment(segment) {
			return p.handleDistributedOperation(current, segments[i:])
		}

		switch segment.TypeString() {
		case "property":
			result := p.handlePropertyAccess(current, segment.Key)
			if !result.exists {
				return nil, ErrPathNotFound
			}
			current = result.value

		case "array":
			result := p.handleArrayAccess(current, segment)
			if !result.exists {
				return nil, ErrPathNotFound
			}
			current = result.value

		case "slice":
			result := p.handleArraySlice(current, segment)
			if !result.exists {
				return nil, ErrPathNotFound
			}
			current = result.value

		case "extract":
			extractResult, err := p.handleExtraction(current, segment)
			if err != nil {
				return nil, err
			}
			current = extractResult

			if i+1 < len(segments) {
				nextSegment := segments[i+1]
				if nextSegment.TypeString() == "array" || nextSegment.TypeString() == "slice" {
					if segment.IsFlatExtract() {
						if nextSegment.TypeString() == "slice" {
							result := p.handleArraySlice(current, nextSegment)
							if result.exists {
								current = result.value
							}
						} else {
							result := p.handleArrayAccess(current, nextSegment)
							if result.exists {
								current = result.value
							}
						}
					} else {
						current = p.handlePostExtractionArrayAccess(current, nextSegment)
					}
					i++ // Skip the next segment since we just processed it
				}
			}

		default:
			return nil, fmt.Errorf("unsupported segment type: %v", segment.TypeString())
		}
	}

	return current, nil
}

func (p *Processor) navigateJSONPointer(data any, path string) (any, error) {
	if path == "/" {
		return data, nil
	}

	pathWithoutSlash := path[1:]
	segments := strings.Split(pathWithoutSlash, "/")

	current := data

	for _, segment := range segments {
		if segment == "" {
			continue
		}

		if strings.Contains(segment, "~") {
			segment = p.unescapeJSONPointer(segment)
		}

		result := p.handlePropertyAccess(current, segment)
		if !result.exists {
			return nil, ErrPathNotFound
		}
		current = result.value
	}

	return current, nil
}

// unescapeJSONPointer unescapes JSON Pointer special characters
func (p *Processor) unescapeJSONPointer(segment string) string {
	return internal.UnescapeJSONPointer(segment)
}

func (p *Processor) handlePropertyAccess(data any, property string) propertyAccessResult {
	switch v := data.(type) {
	case map[string]any:
		if val, exists := v[property]; exists {
			return propertyAccessResult{value: val, exists: true}
		}
		return propertyAccessResult{exists: false}

	case map[any]any:
		if val, exists := v[property]; exists {
			return propertyAccessResult{value: val, exists: true}
		}
		return propertyAccessResult{exists: false}

	case []any:
		if index := p.parseArrayIndex(property); index >= 0 && index < len(v) {
			return propertyAccessResult{value: v[index], exists: true}
		}
		return propertyAccessResult{exists: false}

	default:
		if structValue := p.handleStructAccess(data, property); structValue != nil {
			return propertyAccessResult{value: structValue, exists: true}
		}
		return propertyAccessResult{exists: false}
	}
}

func (p *Processor) handlePropertyAccessValue(data any, property string) any {
	result := p.handlePropertyAccess(data, property)
	if result.exists {
		return result.value
	}
	return nil
}

// numberPreservingDecoder provides JSON decoding with optimized number format preservation
type numberPreservingDecoder struct {
	preserveNumbers bool
}

// newNumberPreservingDecoder creates a new decoder with performance and number preservation
func newNumberPreservingDecoder(preserveNumbers bool) *numberPreservingDecoder {
	return &numberPreservingDecoder{
		preserveNumbers: preserveNumbers,
	}
}

// DecodeToAny decodes JSON string to any type with performance and number preservation
func (d *numberPreservingDecoder) DecodeToAny(jsonStr string) (any, error) {
	if !d.preserveNumbers {
		// Fast path: use standard JSON decoding without number preservation
		var result any
		if err := json.Unmarshal(stringToBytes(jsonStr), &result); err != nil {
			return nil, err
		}
		return result, nil
	}

	// Create a new decoder for each call (json.Decoder cannot be reused with different inputs)
	decoder := json.NewDecoder(strings.NewReader(jsonStr))
	decoder.UseNumber()

	var result any
	if err := decoder.Decode(&result); err != nil {
		return nil, err
	}

	// Convert json.Number to our Number type for encoding/json.UseNumber compatibility
	result = d.convertStdJSONNumbers(result)
	return result, nil
}

// convertStdJSONNumbers converts standard library json.Number to our Number type.
// This preserves the original number representation for UseNumber() compatibility.
func (d *numberPreservingDecoder) convertStdJSONNumbers(value any) any {
	switch v := value.(type) {
	case json.Number:
		return Number(string(v))
	case map[string]any:
		result := make(map[string]any, len(v))
		for key, val := range v {
			result[key] = d.convertStdJSONNumbers(val)
		}
		return result
	case []any:
		result := make([]any, len(v))
		for i, val := range v {
			result[i] = d.convertStdJSONNumbers(val)
		}
		return result
	default:
		return v
	}
}

// convertNumbers recursively converts json.Number to native types (int, float64) when possible,
// falling back to Number type for very large numbers. Used by preservingUnmarshal.
func (d *numberPreservingDecoder) convertNumbers(value any) any {
	switch v := value.(type) {
	case json.Number:
		return d.convertJSONNumber(v)
	case map[string]any:
		// Pre-allocate map with known size for better performance
		result := make(map[string]any, len(v))
		for key, val := range v {
			result[key] = d.convertNumbers(val)
		}
		return result
	case []any:
		// Pre-allocate slice with known size
		result := make([]any, len(v))
		for i, val := range v {
			result[i] = d.convertNumbers(val)
		}
		return result
	default:
		return v
	}
}

// convertJSONNumber converts json.Number with precision handling
// PERFORMANCE: Optimized to minimize allocations and use manual parsing where possible
func (d *numberPreservingDecoder) convertJSONNumber(num json.Number) any {
	numStr := string(num)
	numLen := len(numStr)

	// Ultra-fast path for single digits
	if numLen == 1 {
		c := numStr[0]
		if c >= '0' && c <= '9' {
			return int(c - '0')
		}
	}

	// PERFORMANCE: Single scan to detect number format
	hasDecimal := false
	hasScientific := false
	for i := range numLen {
		c := numStr[i]
		switch c {
		case '.':
			hasDecimal = true
		case 'e', 'E':
			hasScientific = true
		}
	}

	// Fast path for small integers without decimal or scientific notation
	if !hasDecimal && !hasScientific && numLen <= 10 {
		// Try manual parsing for small integers
		negative := false
		start := 0
		if numStr[0] == '-' {
			negative = true
			start = 1
		}

		if numLen-start > 0 && numLen-start <= 10 {
			var result int64
			valid := true
			for i := start; i < numLen; i++ {
				c := numStr[i]
				if c < '0' || c > '9' {
					valid = false
					break
				}
				result = result*10 + int64(c-'0')
			}
			if valid {
				if negative {
					result = -result
				}
				// Check if it fits in int32
				if result >= -2147483648 && result <= 2147483647 {
					return int(result)
				}
				return result
			}
		}
	}

	// Integer parsing with optimized range checking
	if !hasDecimal && !hasScientific {
		if i, err := strconv.ParseInt(numStr, 10, 64); err == nil {
			// Use bit operations for faster range checking
			if i >= -2147483648 && i <= 2147483647 { // int32 range
				return int(i)
			}
			return i
		}

		// Try uint64 for large positive numbers
		if u, err := strconv.ParseUint(numStr, 10, 64); err == nil {
			return u
		}

		// Number too large for standard types, preserve as Number for type safety
		return Number(numStr)
	}

	// Handle "clean" floats (ending with .0)
	if hasDecimal && numLen > 2 && numStr[numLen-2] == '.' && numStr[numLen-1] == '0' {
		intStr := numStr[:numLen-2]
		if i, err := strconv.ParseInt(intStr, 10, 64); err == nil {
			if i >= -2147483648 && i <= 2147483647 {
				return int(i)
			}
			return i
		}
		// If integer conversion fails, try to parse as float
		if f, err := strconv.ParseFloat(numStr, 64); err == nil {
			return f
		}
		// Last resort: return as Number to maintain numeric type identity
		return Number(numStr)
	}

	// Handle decimal numbers with precision checking
	if hasDecimal && !hasScientific {
		if f, err := strconv.ParseFloat(numStr, 64); err == nil {
			// Always return the float64 value to maintain numeric type consistency
			// Precision checking is less important than type consistency
			return f
		}
		// If parsing fails, return as Number to maintain numeric type identity
		return Number(numStr)
	}

	// Handle scientific notation
	if hasScientific {
		if f, err := strconv.ParseFloat(numStr, 64); err == nil {
			return f
		}
	}

	// Fallback: return as Number to maintain numeric type identity
	return Number(numStr)
}

// preservingUnmarshal unmarshals JSON with number preservation
func preservingUnmarshal(data []byte, v any, preserveNumbers bool) error {
	if !preserveNumbers {
		return json.Unmarshal(data, v)
	}

	// Use json.Number for preservation
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.UseNumber()

	// First decode to any to handle json.Number conversion
	var temp any
	if err := decoder.Decode(&temp); err != nil {
		return err
	}

	// Convert numbers and then marshal/unmarshal to target type
	converted := newNumberPreservingDecoder(true).convertNumbers(temp)

	// Marshal the converted data and unmarshal to target
	convertedBytes, err := json.Marshal(converted)
	if err != nil {
		return err
	}

	return json.Unmarshal(convertedBytes, v)
}

func (p *Processor) deepCopyData(data any) (any, error) {
	return DeepCopy(data)
}

func (p *Processor) escapeJSONPointer(segment string) string {
	// Use the centralized JSON pointer escaping helper
	return EscapeJSONPointer(segment)
}

func (p *Processor) normalizePathSeparators(path string) string {
	return internal.NormalizePathSeparators(path)
}

func (p *Processor) splitPathSegments(path string) []string {
	if path == "" {
		return []string{}
	}

	// Handle JSON Pointer format
	if strings.HasPrefix(path, "/") {
		pathWithoutSlash := path[1:]
		if pathWithoutSlash == "" {
			return []string{}
		}
		return strings.Split(pathWithoutSlash, "/")
	}

	// Handle dot notation
	return strings.Split(path, ".")
}

func (p *Processor) joinPathSegments(segments []string, useJSONPointer bool) string {
	if len(segments) == 0 {
		return ""
	}

	if useJSONPointer {
		return "/" + strings.Join(segments, "/")
	}

	return strings.Join(segments, ".")
}

func (p *Processor) isValidPropertyName(name string) bool {
	return internal.IsValidPropertyName(name)
}

func (p *Processor) isValidArrayIndex(index string) bool {
	return internal.IsValidArrayIndex(index)
}

func (p *Processor) isValidSliceRange(rangeStr string) bool {
	return internal.IsValidSliceRange(rangeStr)
}
