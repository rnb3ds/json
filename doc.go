// Package json provides a high-performance, thread-safe JSON processing library
// with 100% encoding/json compatibility and advanced path operations.
//
// The package uses an internal package for implementation details:
//
//   - internal: Private implementation including path parsing, navigation, extraction,
//     caching, array utilities, security helpers, and encoding utilities
//
// Most users can simply import the root package:
//
//	import "github.com/cybergodev/json"
//
// # Basic Usage
//
// Simple operations (100% compatible with encoding/json):
//
//	data, err := json.Marshal(value)
//	err = json.Unmarshal(data, &target)
//
// Advanced path operations:
//
//	value, err := json.Get(`{"user":{"name":"John"}}`, "user.name")
//	result, err := json.Set(`{"user":{}}`, "user.age", 30)
//
// Type-safe operations:
//
//	name := json.GetString(jsonStr, "user.name", "")
//	age := json.GetInt(jsonStr, "user.age", 0)
//
// Advanced processor for complex operations:
//
//	processor, err := json.New() // Use default config
//	if err != nil {
//	    // handle error
//	}
//	defer processor.Close()
//	value, err := processor.Get(jsonStr, "complex.path[0].field")
//
// # Configuration
//
// Use DefaultConfig and optional parameters for custom configuration:
//
//	cfg := json.DefaultConfig()
//	cfg.EnableCache = true
//	processor, err := json.New(cfg)
//	if err != nil {
//	    // handle error
//	}
//	defer processor.Close()
//
// # Key Features
//
//   - 100% encoding/json compatibility - drop-in replacement
//   - High-performance path operations with smart caching
//   - Thread-safe concurrent operations
//   - Type-safe generic operations with Go 1.22+ features
//   - Memory-efficient resource pooling
//   - Production-ready error handling and validation
//
// # Package Structure
//
// The package is organized with all public API in the root package:
//
//   - Core types: Processor, Config, ProcessorOptions, EncodeConfig
//   - Error types: JsonsError, various error constructors
//   - Encoding types: Number
//
// Implementation details are in the internal/ package:
//
//   - Path parsing and navigation utilities
//   - Extraction and segment handling
//   - Cache and array utilities
//   - Security and encoding helpers
//
// # Core Types Organization
//
// Core types are organized in the following files:
//
//   - types.go: All type definitions (Config, ProcessorOptions, Stats, etc.)
//   - processor.go: Processor struct and all methods
//   - ops.go: Internal operation implementations
//   - path.go: Path parsing and navigation
//   - encoding.go: JSON encoding/decoding
//   - api.go: Package-level API functions
//   - file.go: File operations
//   - iterator.go: Iteration utilities
//   - recursive.go: Recursive processing
package json
