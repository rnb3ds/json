package json

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"golang.org/x/text/unicode/norm"
)

// globalRand is a random source for sampling operations
// Initialized with current time for different seeds across runs
// SECURITY FIX: Protected by globalRandMu for concurrent access
var globalRand = rand.New(rand.NewSource(time.Now().UnixNano()))
var globalRandMu sync.Mutex

// randIntn returns a random integer in [0, n) in a thread-safe manner
func randIntn(n int) int {
	globalRandMu.Lock()
	defer globalRandMu.Unlock()
	return globalRand.Intn(n)
}

// LoadFromFile loads JSON data from a file and returns the raw JSON string.
func (p *Processor) LoadFromFile(filePath string, opts ...Config) (string, error) {
	if err := p.checkClosed(); err != nil {
		return "", err
	}

	// Validate file path for security
	if err := p.validateFilePath(filePath); err != nil {
		return "", err
	}

	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", &JsonsError{
			Op:      "load_from_file",
			Message: fmt.Sprintf("failed to read file %s: %v", filePath, err),
			Err:     err,
		}
	}

	return string(data), nil
}

// LoadFromFileAsData loads JSON data from a file and returns the parsed data structure.
func (p *Processor) LoadFromFileAsData(filePath string, opts ...Config) (any, error) {
	if err := p.checkClosed(); err != nil {
		return nil, err
	}

	// Validate file path for security
	if err := p.validateFilePath(filePath); err != nil {
		return nil, err
	}

	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, &JsonsError{
			Op:      "load_from_file_as_data",
			Message: fmt.Sprintf("failed to read file %s: %v", filePath, err),
			Err:     err,
		}
	}

	// Parse JSON
	var jsonData any
	err = p.Parse(string(data), &jsonData, opts...)
	return jsonData, err
}

// LoadFromReader loads JSON data from an io.Reader and returns the raw JSON string.
func (p *Processor) LoadFromReader(reader io.Reader, opts ...Config) (string, error) {
	if err := p.checkClosed(); err != nil {
		return "", err
	}

	// Use LimitReader to prevent excessive memory usage
	limitedReader := io.LimitReader(reader, p.config.MaxJSONSize)

	// Read all data
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return "", &JsonsError{
			Op:      "load_from_reader",
			Message: fmt.Sprintf("failed to read from reader: %v", err),
			Err:     err,
		}
	}

	// Check if we hit the size limit
	if int64(len(data)) >= p.config.MaxJSONSize {
		return "", &JsonsError{
			Op:      "load_from_reader",
			Message: fmt.Sprintf("JSON size exceeds maximum %d bytes", p.config.MaxJSONSize),
			Err:     ErrSizeLimit,
		}
	}

	return string(data), nil
}

// LoadFromReaderAsData loads JSON data from an io.Reader and returns the parsed data structure.
func (p *Processor) LoadFromReaderAsData(reader io.Reader, opts ...Config) (any, error) {
	if err := p.checkClosed(); err != nil {
		return nil, err
	}

	// Use LimitReader to prevent excessive memory usage
	limitedReader := io.LimitReader(reader, p.config.MaxJSONSize)

	// Read all data
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, &JsonsError{
			Op:      "load_from_reader_as_data",
			Message: fmt.Sprintf("failed to read from reader: %v", err),
			Err:     err,
		}
	}

	// Check if we hit the size limit
	if int64(len(data)) >= p.config.MaxJSONSize {
		return nil, &JsonsError{
			Op:      "load_from_reader_as_data",
			Message: fmt.Sprintf("JSON size exceeds maximum %d bytes", p.config.MaxJSONSize),
			Err:     ErrSizeLimit,
		}
	}

	// Parse JSON
	var jsonData any
	err = p.Parse(string(data), &jsonData, opts...)
	return jsonData, err
}

// preprocessDataForEncoding normalizes string/[]byte inputs to prevent double-encoding.
func (p *Processor) preprocessDataForEncoding(data any) (any, error) {
	switch v := data.(type) {
	case string:
		// Parse JSON string to prevent double-encoding
		var parsed any
		if err := p.Parse(v, &parsed); err != nil {
			return nil, &JsonsError{
				Op:      "preprocess_data",
				Message: "invalid JSON string input",
				Err:     err,
			}
		}
		return parsed, nil
	case []byte:
		// Parse JSON bytes to prevent double-encoding
		var parsed any
		if err := p.Parse(string(v), &parsed); err != nil {
			return nil, &JsonsError{
				Op:      "preprocess_data",
				Message: "invalid JSON byte input",
				Err:     err,
			}
		}
		return parsed, nil
	default:
		// Return other types as-is (will be encoded normally)
		return data, nil
	}
}

// createDirectoryIfNotExists creates the directory structure for a file path if needed.
func (p *Processor) createDirectoryIfNotExists(filePath string) error {
	dir := filepath.Dir(filePath)
	if dir == "." || dir == "/" {
		return nil // No directory to create
	}

	// Check if directory already exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		// Create directory with appropriate permissions
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return nil
}

// SaveToFile saves data to a JSON file using Config.
// This is the unified API that accepts variadic Config.
//
// Example:
//
//	err := processor.SaveToFile("data.json", data, json.PrettyConfig())
func (p *Processor) SaveToFile(filePath string, data any, cfg ...Config) error {
	if err := p.checkClosed(); err != nil {
		return err
	}

	// Validate file path for security
	if err := p.validateFilePath(filePath); err != nil {
		return err
	}

	// Create directory if it doesn't exist
	if err := p.createDirectoryIfNotExists(filePath); err != nil {
		return &JsonsError{
			Op:      "save_to_file",
			Message: fmt.Sprintf("failed to create directory for %s", filePath),
			Err:     fmt.Errorf("directory creation error: %w", err),
		}
	}

	// Preprocess data to prevent double-encoding of string/[]byte inputs
	processedData, err := p.preprocessDataForEncoding(data)
	if err != nil {
		return err
	}

	// Encode data to JSON
	config := DefaultConfig()
	if len(cfg) > 0 {
		config = cfg[0]
	}
	jsonStr, err := p.EncodeWithConfig(processedData, config)
	if err != nil {
		return err
	}

	// Write to file
	err = os.WriteFile(filePath, []byte(jsonStr), 0644)
	if err != nil {
		return &JsonsError{
			Op:      "save_to_file",
			Message: fmt.Sprintf("failed to write file %s", filePath),
			Err:     fmt.Errorf("write file error: %w", err),
		}
	}

	return nil
}

// SaveToWriter saves data to an io.Writer using Config.
// This is the unified API that accepts variadic Config.
//
// Example:
//
//	var buf bytes.Buffer
//	err := processor.SaveToWriter(&buf, data, json.PrettyConfig())
func (p *Processor) SaveToWriter(writer io.Writer, data any, cfg ...Config) error {
	if err := p.checkClosed(); err != nil {
		return err
	}

	// Preprocess data to prevent double-encoding of string/[]byte inputs
	processedData, err := p.preprocessDataForEncoding(data)
	if err != nil {
		return err
	}

	// Encode data to JSON
	config := DefaultConfig()
	if len(cfg) > 0 {
		config = cfg[0]
	}
	jsonStr, err := p.EncodeWithConfig(processedData, config)
	if err != nil {
		return err
	}

	// Write to writer
	_, err = writer.Write([]byte(jsonStr))
	if err != nil {
		return &JsonsError{
			Op:      "save_to_writer",
			Message: fmt.Sprintf("failed to write to writer: %v", err),
			Err:     ErrOperationFailed,
		}
	}

	return nil
}

// MarshalToFile converts data to JSON and saves it to the specified file using Config.
// This is the unified API that accepts variadic Config.
//
// Example:
//
//	err := processor.MarshalToFile("data.json", data, json.PrettyConfig())
func (p *Processor) MarshalToFile(path string, data any, cfg ...Config) error {
	if err := p.checkClosed(); err != nil {
		return err
	}

	// Validate file path for security
	if err := p.validateFilePath(path); err != nil {
		return err
	}

	// Create directory if it doesn't exist
	if err := p.createDirectoryIfNotExists(path); err != nil {
		return &JsonsError{
			Op:      "marshal_to_file",
			Message: fmt.Sprintf("failed to create directory for %s", path),
			Err:     err,
		}
	}

	// Preprocess data to prevent double-encoding of string/[]byte inputs
	processedData, err := p.preprocessDataForEncoding(data)
	if err != nil {
		return err
	}

	// Determine formatting preference
	config := DefaultConfig()
	if len(cfg) > 0 {
		config = cfg[0]
	}

	// Marshal data to JSON bytes
	var jsonBytes []byte
	if config.Pretty {
		jsonBytes, err = p.MarshalIndent(processedData, "", "  ")
	} else {
		jsonBytes, err = p.Marshal(processedData)
	}

	if err != nil {
		return &JsonsError{
			Op:      "marshal_to_file",
			Message: "failed to marshal data to JSON",
			Err:     err,
		}
	}

	// Write JSON bytes to file
	if err := os.WriteFile(path, jsonBytes, 0644); err != nil {
		return &JsonsError{
			Op:      "marshal_to_file",
			Path:    path,
			Message: fmt.Sprintf("failed to write file %s", path),
			Err:     err,
		}
	}

	return nil
}

// UnmarshalFromFile reads JSON data from the specified file and unmarshals it into the provided value.
func (p *Processor) UnmarshalFromFile(path string, v any, opts ...Config) error {
	if err := p.checkClosed(); err != nil {
		return err
	}

	// Validate input parameters
	if v == nil {
		return &JsonsError{
			Op:      "unmarshal_from_file",
			Message: "unmarshal target cannot be nil",
			Err:     ErrOperationFailed,
		}
	}

	// Validate file path for security
	if err := p.validateFilePath(path); err != nil {
		return err
	}

	// Read file contents with size validation
	data, err := os.ReadFile(path)
	if err != nil {
		return &JsonsError{
			Op:      "unmarshal_from_file",
			Path:    path,
			Message: fmt.Sprintf("failed to read file %s", path),
			Err:     err,
		}
	}

	// Check file size against processor limits
	if int64(len(data)) > p.config.MaxJSONSize {
		return &JsonsError{
			Op:      "unmarshal_from_file",
			Path:    path,
			Message: fmt.Sprintf("file size %d exceeds maximum allowed size %d", len(data), p.config.MaxJSONSize),
			Err:     ErrSizeLimit,
		}
	}

	// Unmarshal JSON data using processor's Unmarshal method
	if err := p.Unmarshal(data, v, opts...); err != nil {
		return &JsonsError{
			Op:      "unmarshal_from_file",
			Path:    path,
			Message: fmt.Sprintf("failed to unmarshal JSON from file %s", path),
			Err:     err,
		}
	}

	return nil
}

// validateFilePath provides enhanced security validation for file paths.
// Uses smaller helper functions for better maintainability and testability.
func (p *Processor) validateFilePath(filePath string) error {
	// Step 1: Basic validation
	if err := validatePathBasic(filePath); err != nil {
		return err
	}

	// Step 2: Security pattern validation
	if err := validatePathSecurity(filePath); err != nil {
		return err
	}

	// Step 3: Normalize and get absolute path
	absPath, err := normalizeAndAbsPath(filePath)
	if err != nil {
		return err
	}

	// Step 4: Platform-specific validation on absolute path
	if err := validatePathPlatform(absPath); err != nil {
		return err
	}

	// Step 5: Symlink validation
	if err := validatePathSymlinks(absPath); err != nil {
		return err
	}

	// Step 6: File size validation
	return p.validatePathFileSize(absPath)
}

// validatePathBasic performs basic path validation
func validatePathBasic(filePath string) error {
	if filePath == "" {
		return newOperationError("validate_file_path", "file path cannot be empty", ErrOperationFailed)
	}

	// SECURITY: Check for null bytes before any processing
	if strings.Contains(filePath, "\x00") {
		return newSecurityError("validate_file_path", "null byte in path")
	}

	return nil
}

// validatePathSecurity checks for path traversal and platform-specific security issues
func validatePathSecurity(filePath string) error {
	// SECURITY: Check for path traversal patterns BEFORE normalization
	if containsPathTraversal(filePath) {
		return newSecurityError("validate_file_path", "path traversal pattern detected")
	}

	// Platform-specific security checks on original path (before normalization)
	if runtime.GOOS == "windows" {
		if err := validateWindowsPath(filePath); err != nil {
			return err
		}
	}

	return nil
}

// normalizeAndAbsPath normalizes the path and returns its absolute form
func normalizeAndAbsPath(filePath string) (string, error) {
	// Normalize the path after security checks
	cleanPath := filepath.Clean(filePath)

	// Check path length after cleaning
	if len(cleanPath) > MaxPathLength {
		return "", newOperationError("validate_file_path",
			fmt.Sprintf("path too long: %d > %d", len(cleanPath), MaxPathLength),
			ErrOperationFailed)
	}

	// Convert to absolute path for further validation
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return "", newOperationError("validate_file_path", "invalid path", err)
	}

	return absPath, nil
}

// validatePathPlatform performs platform-specific security checks on absolute path
func validatePathPlatform(absPath string) error {
	if runtime.GOOS != "windows" {
		if err := validateUnixPath(absPath); err != nil {
			return err
		}
	}
	return nil
}

// validatePathSymlinks checks for symlink security issues
func validatePathSymlinks(absPath string) error {
	info, err := os.Lstat(absPath)
	if err != nil {
		// File doesn't exist yet, no symlink check needed
		return nil
	}

	if info.Mode()&os.ModeSymlink == 0 {
		// Not a symlink, no check needed
		return nil
	}

	realPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		return newOperationError("validate_file_path", "cannot resolve symlink", err)
	}

	// Ensure symlink doesn't escape to restricted areas
	if runtime.GOOS != "windows" {
		return validateUnixPath(realPath)
	}
	return validateWindowsPath(realPath)
}

// validatePathFileSize checks if file size is within limits
func (p *Processor) validatePathFileSize(absPath string) error {
	info, err := os.Stat(absPath)
	if err != nil {
		// File doesn't exist yet, no size check needed
		return nil
	}

	if info.Size() > p.config.MaxJSONSize {
		return newSizeLimitError("validate_file_path", info.Size(), p.config.MaxJSONSize)
	}
	return nil
}

// containsPathTraversal checks for path traversal patterns comprehensively.
// Uses case-insensitive matching with Unicode normalization and recursive URL decoding.
func containsPathTraversal(path string) bool {
	// SECURITY: Apply Unicode NFC normalization to detect homograph attacks
	normalized := norm.NFC.String(path)
	// SECURITY: Recursively decode URL encoding to catch multi-layered obfuscation
	decoded := recursiveURLDecode(normalized)

	// Check both decoded and original for all pattern types
	for _, s := range []string{decoded, path} {
		if containsBasicTraversalPattern(s) || containsEncodedPattern(s) || containsUnicodeLookalike(s) {
			return true
		}
	}
	return false
}

// containsBasicTraversalPattern checks for standalone ".." path components.
func containsBasicTraversalPattern(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '.' && i+1 < len(s) && s[i+1] == '.' {
			beforeOK := i == 0 || s[i-1] == '/' || s[i-1] == '\\'
			afterOK := i+2 >= len(s) || s[i+2] == '/' || s[i+2] == '\\'
			if beforeOK && afterOK {
				return true
			}
			i++ // Skip past ".." to avoid false matches
		}
	}
	return containsConsecutiveDots(s, 3)
}

// traversalPatterns contains all known traversal attack patterns (encoded, control chars, partial encoding).
var traversalPatterns = []string{
	// URL encoded patterns
	"%2e%2e", "%252e%252e", "%25252e%25252e",
	// Mixed encoding patterns
	"..%2f", "..%5c", "..%c0%af", "..%c1%9c",
	// Partial encoding patterns
	".%2e", "%2e.", "%2e%2e%2f", "%2e%2e%5c",
	// Windows patterns
	"..\\", "..\\/",
	// Injection patterns (control chars)
	"..%00", "..%0a", "..%0d", "..%09", "..%20",
	"%00", "%0a", "%0d", "%09", "%20",
	// Double patterns
	"....//", "....\\\\", ".....", "......",
	// Mixed case patterns
	"%2E%2E", "%2E%2e", "%2e%2E", "..%2F", "..%5C",
	// UTF-8 overlong encoding
	"%c0%ae", "%c1%1c", "%c1%9c", "..%255c",
	// Fullwidth encoding
	"%uff0e%uff0e", "..%ef%bc%8f",
	// Partial double encoding
	"%2e%2", "%25%2e", "%2f%2", "%5c%2",
}

// containsEncodedPattern checks for encoded path traversal patterns.
func containsEncodedPattern(s string) bool {
	for _, pattern := range traversalPatterns {
		if fastIndexIgnoreCase(s, pattern) != -1 {
			return true
		}
	}
	return false
}

// recursiveURLDecode recursively decodes URL-encoded strings (max 3 levels).
func recursiveURLDecode(s string) string {
	decoded := s
	for i := 0; i < 3; i++ {
		newDecoded, err := url.PathUnescape(decoded)
		if err != nil || newDecoded == decoded {
			break
		}
		decoded = newDecoded
	}
	return decoded
}

// containsUnicodeLookalike checks for Unicode characters that resemble path separators or dots.
func containsUnicodeLookalike(s string) bool {
	for _, r := range s {
		switch r {
		// Dot lookalikes
		case '\uFF0E', '\u2024', '\u2025', '\u2026':
			return true
		// Slash lookalikes
		case '\uFF0F', '\uFF3C', '\u2044', '\u2215', '\u29F8', '\uFE68':
			return true
		// Dangerous invisible/formatting characters
		case '\uFEFF', '\u2060', '\u200B', '\u200C', '\u200D', '\u3000', '\u00AD', '\u034F', '\u061C', '\u115F', '\u1160', '\u180E':
			return true
		}
	}
	return false
}

// hasPrefixIgnoreCase checks if s starts with prefix case-insensitively
func hasPrefixIgnoreCase(s, prefix string) bool {
	if len(prefix) > len(s) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		c1 := s[i]
		c2 := prefix[i]
		if c1 >= 'A' && c1 <= 'Z' {
			c1 += 32
		}
		if c2 >= 'A' && c2 <= 'Z' {
			c2 += 32
		}
		if c1 != c2 {
			return false
		}
	}
	return true
}

// containsConsecutiveDots checks for consecutive dots in any form
func containsConsecutiveDots(path string, minCount int) bool {
	dotCount := 0
	for _, r := range path {
		if r == '.' {
			dotCount++
			if dotCount >= minCount {
				return true
			}
		} else {
			dotCount = 0
		}
	}
	return false
}

// validateUnixPath validates Unix-specific path security
func validateUnixPath(absPath string) error {
	// Block access to critical system directories using case-insensitive matching
	criticalDirs := []string{
		"/dev/",
		"/proc/",
		"/sys/",
		"/etc/passwd",
		"/etc/shadow",
		"/etc/sudoers",
		"/etc/hosts",
		"/etc/fstab",
		"/etc/crontab",
		"/root/",
		"/boot/",
		"/var/log/",
		"/usr/bin/",
		"/usr/sbin/",
		"/sbin/",
		"/bin/",
	}

	for _, dir := range criticalDirs {
		if hasPrefixIgnoreCase(absPath, dir) {
			return newSecurityError("validate_unix_path", "access to system directory not allowed")
		}
	}

	// Additional security checks for Unix systems
	if strings.Contains(absPath, "/..") || strings.Contains(absPath, "../") {
		return newSecurityError("validate_unix_path", "path traversal detected")
	}

	return nil
}

// validateWindowsPath validates Windows-specific path security
func validateWindowsPath(absPath string) error {
	// Check for UNC paths
	if strings.HasPrefix(absPath, "\\\\") || strings.HasPrefix(absPath, "//") {
		return newSecurityError("validate_windows_path", "UNC paths not allowed")
	}

	// Extract filename for device name checking
	filename := strings.ToUpper(filepath.Base(absPath))
	if idx := strings.LastIndex(filename, "."); idx > 0 {
		filename = filename[:idx]
	}

	// Check reserved device names (complete list)
	reserved := []string{"CON", "PRN", "AUX", "NUL", "CONIN$", "CONOUT$"}
	for _, name := range reserved {
		if filename == name {
			return newSecurityError("validate_windows_path", "Windows reserved device name")
		}
	}

	// Additional check for alternate data streams (ADS)
	if strings.Contains(absPath, ":") {
		parts := strings.SplitN(absPath, ":", 2)
		if len(parts) == 2 {
			// Check if it looks like a drive letter pattern
			if len(parts[0]) == 1 && parts[0][0] >= 'A' && parts[0][0] <= 'Z' {
				// This is a drive letter path, not ADS
			} else {
				return newSecurityError("validate_windows_path", "alternate data streams not allowed")
			}
		}
	}

	// Check COM1-9 and LPT1-9
	if len(filename) == 4 && filename[3] >= '0' && filename[3] <= '9' {
		prefix := filename[:3]
		if prefix == "COM" || prefix == "LPT" {
			return newSecurityError("validate_windows_path", "Windows reserved device name")
		}
	}

	// Check COM0 and LPT0 (explicitly invalid in Windows)
	if filename == "COM0" || filename == "LPT0" {
		return newSecurityError("validate_windows_path", "Windows reserved device name")
	}

	// Check for invalid characters in Windows paths
	pathToCheck := absPath
	if len(absPath) > 2 && absPath[1] == ':' {
		pathToCheck = absPath[2:]
	}

	invalidChars := []string{"<", ">", ":", "\"", "|", "?", "*"}
	for _, char := range invalidChars {
		if strings.Contains(pathToCheck, char) {
			return newSecurityError("validate_windows_path", "invalid character in path")
		}
	}

	return nil
}

// ============================================================================
// LARGE JSON FILE PROCESSOR
// Provides memory-efficient processing for very large JSON files
// PERFORMANCE: Memory-mapped file support and chunked processing
// ============================================================================

// LargeFileConfig holds configuration for large file processing
type LargeFileConfig struct {
	ChunkSize       int64 // Size of each chunk in bytes
	MaxMemory       int64 // Maximum memory to use
	BufferSize      int   // Buffer size for reading
	SamplingEnabled bool  // Enable sampling for very large files
	SampleSize      int   // Number of samples to take
}

// DefaultLargeFileConfig returns the default configuration
func DefaultLargeFileConfig() LargeFileConfig {
	return LargeFileConfig{
		ChunkSize:       1024 * 1024,       // 1MB chunks
		MaxMemory:       100 * 1024 * 1024, // 100MB max
		BufferSize:      64 * 1024,         // 64KB buffer
		SamplingEnabled: true,
		SampleSize:      1000,
	}
}

// LargeFileProcessor handles processing of large JSON files
type LargeFileProcessor struct {
	config LargeFileConfig
}

// NewLargeFileProcessor creates a new large file processor
func NewLargeFileProcessor(config LargeFileConfig) *LargeFileProcessor {
	return &LargeFileProcessor{config: config}
}

// ProcessFile processes a large JSON file efficiently
func (lfp *LargeFileProcessor) ProcessFile(filename string, fn func(item any) error) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Use buffered reader for efficiency
	reader := bufio.NewReaderSize(file, lfp.config.BufferSize)

	// Create streaming processor
	sp := NewStreamingProcessor(reader, int(lfp.config.ChunkSize))

	// Stream array elements
	return sp.StreamArray(func(index int, item any) bool {
		if err := fn(item); err != nil {
			return false
		}
		return true
	})
}

// ProcessFileChunked processes a large JSON file in chunks
func (lfp *LargeFileProcessor) ProcessFileChunked(filename string, chunkSize int, fn func(chunk []any) error) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	reader := bufio.NewReaderSize(file, lfp.config.BufferSize)
	sp := NewStreamingProcessor(reader, int(lfp.config.ChunkSize))

	chunk := make([]any, 0, chunkSize)

	err = sp.StreamArray(func(index int, item any) bool {
		chunk = append(chunk, item)

		if len(chunk) >= chunkSize {
			if err := fn(chunk); err != nil {
				return false
			}
			chunk = chunk[:0] // Reset chunk
		}
		return true
	})

	// Process remaining items
	if err == nil && len(chunk) > 0 {
		err = fn(chunk)
	}

	return err
}

// ============================================================================
// CHUNKED JSON READER
// ============================================================================

// ChunkedReader reads JSON in chunks for memory efficiency
type ChunkedReader struct {
	reader    *bufio.Reader
	decoder   *json.Decoder
	buffer    []byte
	chunkSize int
}

// NewChunkedReader creates a new chunked reader
func NewChunkedReader(reader io.Reader, chunkSize int) *ChunkedReader {
	if chunkSize <= 0 {
		chunkSize = 1024 * 1024 // 1MB default
	}

	bufReader := bufio.NewReaderSize(reader, 64*1024)
	return &ChunkedReader{
		reader:    bufReader,
		decoder:   json.NewDecoder(bufReader),
		buffer:    make([]byte, 0, chunkSize),
		chunkSize: chunkSize,
	}
}

// ReadArray reads array elements one at a time
func (cr *ChunkedReader) ReadArray(fn func(item any) bool) error {
	// Check for array start
	token, err := cr.decoder.Token()
	if err != nil {
		return err
	}

	if token != json.Delim('[') {
		// Not an array, try to decode as single value
		var value any
		if err := cr.decoder.Decode(&value); err != nil {
			return err
		}
		fn(value)
		return nil
	}

	for cr.decoder.More() {
		var item any
		if err := cr.decoder.Decode(&item); err != nil {
			return err
		}

		if !fn(item) {
			return nil
		}
	}

	// Consume closing bracket
	_, err = cr.decoder.Token()
	return err
}

// ReadObject reads object key-value pairs one at a time
func (cr *ChunkedReader) ReadObject(fn func(key string, value any) bool) error {
	token, err := cr.decoder.Token()
	if err != nil {
		return err
	}

	if token != json.Delim('{') {
		return nil
	}

	for cr.decoder.More() {
		key, err := cr.decoder.Token()
		if err != nil {
			return err
		}

		keyStr, ok := key.(string)
		if !ok {
			continue
		}

		var value any
		if err := cr.decoder.Decode(&value); err != nil {
			return err
		}

		if !fn(keyStr, value) {
			return nil
		}
	}

	// Consume closing brace
	_, err = cr.decoder.Token()
	return err
}

// ============================================================================
// LAZY JSON PARSER
// Parses JSON on-demand, only parsing accessed paths
// ============================================================================

// LazyParser provides lazy JSON parsing that supports both JSON objects and arrays
type LazyParser struct {
	raw      []byte
	parsed   any // Supports map[string]any, []any, or primitive types
	parseErr error
	once     sync.Once
}

// NewLazyParser creates a new lazy parser
func NewLazyParser(data []byte) *LazyParser {
	return &LazyParser{
		raw: data,
	}
}

// parse performs the actual parsing
func (lp *LazyParser) parse() {
	lp.once.Do(func() {
		lp.parseErr = json.Unmarshal(lp.raw, &lp.parsed)
	})
}

// Get retrieves a value at the given path
func (lp *LazyParser) Get(path string) (any, error) {
	lp.parse()
	if lp.parseErr != nil {
		return nil, lp.parseErr
	}

	// Empty path returns the root value
	if path == "" || path == "." {
		return lp.parsed, nil
	}

	processor := getDefaultProcessor()
	if processor == nil {
		return nil, ErrInternalError
	}
	return processor.GetFromParsedData(lp.parsed, path)
}

// GetObject returns the parsed data as a map (only for JSON objects).
// Returns ErrTypeMismatch if the parsed JSON is not an object.
func (lp *LazyParser) GetObject() (map[string]any, error) {
	lp.parse()
	if lp.parseErr != nil {
		return nil, lp.parseErr
	}

	if m, ok := lp.parsed.(map[string]any); ok {
		return m, nil
	}
	return nil, &JsonsError{
		Op:      "lazy_parser_get_object",
		Message: "parsed JSON is not an object (use GetValue() for arrays or other types)",
		Err:     ErrTypeMismatch,
	}
}

// GetValue returns all parsed data as interface{} (supports any JSON type)
func (lp *LazyParser) GetValue() (any, error) {
	lp.parse()
	if lp.parseErr != nil {
		return nil, lp.parseErr
	}
	return lp.parsed, nil
}

// Raw returns the raw JSON bytes
func (lp *LazyParser) Raw() []byte {
	return lp.raw
}

// IsParsed returns whether the JSON has been parsed
func (lp *LazyParser) IsParsed() bool {
	return lp.parsed != nil
}

// IsObject returns true if the parsed JSON is an object
func (lp *LazyParser) IsObject() bool {
	lp.parse()
	_, ok := lp.parsed.(map[string]any)
	return ok
}

// IsArray returns true if the parsed JSON is an array
func (lp *LazyParser) IsArray() bool {
	lp.parse()
	_, ok := lp.parsed.([]any)
	return ok
}

// Parse forces parsing and returns the parsed data
func (lp *LazyParser) Parse() (any, error) {
	lp.parse()
	return lp.parsed, lp.parseErr
}

// Parsed returns the parsed data without forcing parsing.
// Returns nil if not yet parsed.
func (lp *LazyParser) Parsed() any {
	return lp.parsed
}

// Error returns any parsing error, triggering parse if needed
func (lp *LazyParser) Error() error {
	lp.parse()
	return lp.parseErr
}

// ============================================================================
// LINE-DELIMITED JSON PROCESSOR
// For processing NDJSON (newline-delimited JSON) files
// ============================================================================

// NDJSONProcessor processes newline-delimited JSON files
type NDJSONProcessor struct {
	bufferSize int
}

// NewNDJSONProcessor creates a new NDJSON processor
func NewNDJSONProcessor(bufferSize int) *NDJSONProcessor {
	if bufferSize <= 0 {
		bufferSize = 64 * 1024
	}
	return &NDJSONProcessor{bufferSize: bufferSize}
}

// ProcessFile processes an NDJSON file line by line
func (np *NDJSONProcessor) ProcessFile(filename string, fn func(lineNum int, obj map[string]any) error) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, np.bufferSize)
	scanner.Buffer(buf, 10*1024*1024) // 10MB max line size

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()

		if len(line) == 0 {
			continue
		}

		var obj map[string]any
		if err := json.Unmarshal(line, &obj); err != nil {
			continue // Skip invalid lines
		}

		if err := fn(lineNum, obj); err != nil {
			return err
		}
	}

	return scanner.Err()
}

// ProcessReader processes NDJSON from a reader
func (np *NDJSONProcessor) ProcessReader(reader io.Reader, fn func(lineNum int, obj map[string]any) error) error {
	scanner := bufio.NewScanner(reader)
	buf := make([]byte, 0, np.bufferSize)
	scanner.Buffer(buf, 10*1024*1024)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()

		if len(line) == 0 {
			continue
		}

		var obj map[string]any
		if err := json.Unmarshal(line, &obj); err != nil {
			continue
		}

		if err := fn(lineNum, obj); err != nil {
			return err
		}
	}

	return scanner.Err()
}

// ============================================================================
// CHUNKED JSON WRITER
// ============================================================================

// ChunkedWriter writes JSON in chunks for memory efficiency
type ChunkedWriter struct {
	writer    io.Writer
	buffer    []byte
	chunkSize int
	count     int
	first     bool
	isArray   bool
}

// NewChunkedWriter creates a new chunked writer
func NewChunkedWriter(writer io.Writer, chunkSize int, isArray bool) *ChunkedWriter {
	if chunkSize <= 0 {
		chunkSize = 1024 * 1024
	}
	return &ChunkedWriter{
		writer:    writer,
		buffer:    make([]byte, 0, chunkSize),
		chunkSize: chunkSize,
		first:     true,
		isArray:   isArray,
	}
}

// WriteItem writes a single item to the chunk
func (cw *ChunkedWriter) WriteItem(item any) error {
	// Start array/object if first item
	if cw.first {
		if cw.isArray {
			cw.buffer = append(cw.buffer, '[')
		} else {
			cw.buffer = append(cw.buffer, '{')
		}
		cw.first = false
	} else {
		cw.buffer = append(cw.buffer, ',')
	}

	// Encode item
	data, err := json.Marshal(item)
	if err != nil {
		return err
	}
	cw.buffer = append(cw.buffer, data...)
	cw.count++

	// Flush if buffer is full
	if len(cw.buffer) >= cw.chunkSize {
		return cw.Flush(false)
	}

	return nil
}

// WriteKeyValue writes a key-value pair to the chunk
func (cw *ChunkedWriter) WriteKeyValue(key string, value any) error {
	if cw.isArray {
		return cw.WriteItem(value)
	}

	if cw.first {
		cw.buffer = append(cw.buffer, '{')
		cw.first = false
	} else {
		cw.buffer = append(cw.buffer, ',')
	}

	// Encode key-value pair
	data, err := json.Marshal(map[string]any{key: value})
	if err != nil {
		return err
	}
	// Remove the outer braces
	cw.buffer = append(cw.buffer, data[1:len(data)-1]...)
	cw.count++

	if len(cw.buffer) >= cw.chunkSize {
		return cw.Flush(false)
	}

	return nil
}

// Flush writes the buffer to the underlying writer
func (cw *ChunkedWriter) Flush(final bool) error {
	if final {
		if cw.isArray {
			cw.buffer = append(cw.buffer, ']')
		} else {
			cw.buffer = append(cw.buffer, '}')
		}
	}

	_, err := cw.writer.Write(cw.buffer)
	cw.buffer = cw.buffer[:0]
	return err
}

// Count returns the number of items written
func (cw *ChunkedWriter) Count() int {
	return cw.count
}

// ============================================================================
// SAMPLING JSON READER
// For very large files, samples data instead of reading all
// ============================================================================

// SamplingReader samples data from large JSON arrays
type SamplingReader struct {
	decoder    *json.Decoder
	sampleSize int
	totalRead  int64
}

// NewSamplingReader creates a new sampling reader
func NewSamplingReader(reader io.Reader, sampleSize int) *SamplingReader {
	return &SamplingReader{
		decoder:    json.NewDecoder(reader),
		sampleSize: sampleSize,
	}
}

// Sample reads a sample of items from a JSON array using reservoir sampling.
// The reservoir sampling algorithm ensures uniform random sampling distribution:
// each item in the array has an equal probability of being included in the sample.
func (sr *SamplingReader) Sample(fn func(index int, item any) bool) error {
	// Check for array start
	token, err := sr.decoder.Token()
	if err != nil {
		return err
	}

	if token != json.Delim('[') {
		// Not an array, read single value
		var value any
		if err := sr.decoder.Decode(&value); err != nil {
			return err
		}
		// Honor the callback's return value for consistency
		_ = fn(0, value)
		return nil
	}

	samples := make([]any, 0, sr.sampleSize)
	index := 0

	for sr.decoder.More() {
		var item any
		if err := sr.decoder.Decode(&item); err != nil {
			return err
		}

		sr.totalRead++

		// Reservoir sampling algorithm (Algorithm R)
		// Ensures uniform random sampling where each element has equal probability
		if len(samples) < sr.sampleSize {
			// Fill the reservoir first
			samples = append(samples, item)
		} else {
			// Random replacement with uniform probability k/(index+1)
			// Using math/rand for performance - for cryptographic security use crypto/rand
			// SECURITY FIX: Using thread-safe randIntn wrapper
			j := randIntn(index + 1)
			if j < sr.sampleSize {
				samples[j] = item
			}
		}
		index++
	}

	// Process samples
	for i, sample := range samples {
		if !fn(i, sample) {
			break
		}
	}

	// Consume closing bracket
	_, err = sr.decoder.Token()
	return err
}

// TotalRead returns the total number of items read
func (sr *SamplingReader) TotalRead() int64 {
	return sr.totalRead
}
