# 🚀 cybergodev/json - High-Performance Go JSON Library

[![Go Version](https://img.shields.io/badge/Go-1.24+-blue.svg)](https://golang.org)
[![pkg.go.dev](https://pkg.go.dev/badge/github.com/cybergodev/json.svg)](https://pkg.go.dev/github.com/cybergodev/json)
[![MIT license](https://img.shields.io/badge/license-MIT-brightgreen.svg)](https://opensource.org/licenses/MIT)
[![Performance](https://img.shields.io/badge/performance-high%20performance-green.svg)](https://github.com/cybergodev/json)
[![Thread Safe](https://img.shields.io/badge/thread%20safe-yes-brightgreen.svg)](https://github.com/cybergodev/json)

> A high-performance, feature-rich Go JSON library with 100% `encoding/json` compatibility, providing powerful path operations, type safety, and production-grade performance.

**[📖 中文文档](README_zh-CN.md)**

---

## 🏆 Features

| Feature | Description |
|---------|-------------|
| 🔄 **100% Compatible** | Drop-in replacement for `encoding/json` with zero learning curve |
| 🎯 **Powerful Paths** | Concise path syntax like `users[0].name` for easy nested data access |
| 🚀 **High Performance** | Smart caching, memory optimization, thread-safe |
| 🛡️ **Type Safe** | Generics support with compile-time type checking |
| 🔧 **Feature Rich** | Batch operations, streaming, file operations, data validation |
| 🏗️ **Production Ready** | Thread-safe, comprehensive error handling, security features |

---

## 📦 Installation

```bash
go get github.com/cybergodev/json
```

---

## ⚡ Quick Start (5 Minutes)

```go
package main

import (
    "fmt"
    "github.com/cybergodev/json"
)

func main() {
    // Sample JSON data
    data := `{
        "user": {
            "name": "Alice",
            "age": 28,
            "tags": ["premium", "verified"]
        }
    }`

    // 1. Simple field access
    name, _ := json.GetString(data, "user.name")
    fmt.Println(name) // "Alice"

    // 2. Type-safe retrieval
    age, _ := json.GetInt(data, "user.age")
    fmt.Println(age) // 28

    // 3. Negative index array access
    lastTag, _ := json.Get(data, "user.tags[-1]")
    fmt.Println(lastTag) // "verified"

    // 4. Modify data
    updated, _ := json.Set(data, "user.age", 29)
    newAge, _ := json.GetInt(updated, "user.age")
    fmt.Println(newAge) // 29

    // 5. 100% encoding/json compatible
    bytes, _ := json.Marshal(map[string]any{"status": "ok"})
    fmt.Println(string(bytes)) // {"status":"ok"}
}
```

---

## 📋 Path Syntax Reference

| Syntax | Description | Example |
|--------|-------------|---------|
| `.property` | Access property | `user.name` |
| `[n]` | Array index | `items[0]` |
| `[-n]` | Negative index (from end) | `items[-1]` (last element) |
| `[start:end]` | Array slice | `items[1:3]` (elements 1-2) |
| `[start:end:step]` | Slice with step | `items[::2]` (every other element) |
| `[+]` | Append to array | `items[+]` |
| `{field}` | Extract field from all array elements | `users{name}` |
| `{flat:field}` | Flatten nested arrays | `users{flat:tags}` |

---

## 🎯 Core API

### Data Retrieval

```go
// Basic getters
json.Get(data, "user.name")           // (any, error)
json.GetString(data, "user.name")     // (string, error)
json.GetInt(data, "user.age")         // (int, error)
json.GetFloat64(data, "user.score")   // (float64, error)
json.GetBool(data, "user.active")     // (bool, error)
json.GetArray(data, "user.tags")      // ([]any, error)
json.GetObject(data, "user.profile")  // (map[string]any, error)

// Type-safe generic retrieval
json.GetTyped[string](data, "user.name")
json.GetTyped[[]int](data, "numbers")
json.GetTyped[User](data, "user")     // Custom struct

// With default values (recommended for optional fields)
json.GetDefault(data, "user.name", "Anonymous")     // string
json.GetDefault(data, "user.age", 0)                // int
json.GetDefault(data, "user.score", 0.0)            // float64
json.GetDefault[[]any](data, "user.tags", []any{})  // Generic

// Any type with default
json.GetWithDefault(data, "user.name", "default")

// Batch retrieval
results, err := json.GetMultiple(data, []string{"user.name", "user.age"})
```

### Data Modification

```go
// Basic set - returns modified JSON on success, original data on failure
result, err := json.Set(data, "user.name", "Bob")

// Auto-create paths with config
cfg := json.DefaultConfig()
cfg.CreatePaths = true
result, err := json.Set(data, "user.profile.level", "gold", cfg)

// Append to array
result, _ := json.Set(data, "user.tags[+]", "new-tag")

// Batch set
result, _ := json.SetMultiple(data, map[string]any{
    "user.name": "Bob",
    "user.age":  30,
})

// Delete
result, err := json.Delete(data, "user.temp")
```

### Encoding & Formatting

```go
// Standard encoding (100% compatible with encoding/json)
bytes, err := json.Marshal(data)
err = json.Unmarshal(bytes, &target)
bytes, err := json.MarshalIndent(data, "", "  ")

// Quick formatting
pretty, _ := json.FormatPretty(jsonStr)    // Pretty print
compact, _ := json.CompactString(jsonStr)  // Compact

// Direct output
json.Print(data)        // Compact format to stdout
json.PrintPretty(data)  // Pretty format to stdout

// Encode with config
cfg := json.DefaultConfig()
cfg.Pretty = true
cfg.SortKeys = true
result, err := json.Encode(data, cfg)

// Preset configs
result, _ := json.Encode(data, json.PrettyConfig())
```

### File Operations

```go
// Load and save
jsonStr, err := json.LoadFromFile("data.json")
err = json.SaveToFile("output.json", data, json.PrettyConfig())

// Struct/Map serialization
err = json.MarshalToFile("user.json", user)
err = json.UnmarshalFromFile("user.json", &user)
```

### Type Conversion Utilities

```go
// Safe type conversion
intVal, ok := json.ConvertToInt(value)
floatVal, ok := json.ConvertToFloat64(value)
boolVal, ok := json.ConvertToBool(value)
strVal := json.ConvertToString(value)

// Generic conversion
result, err := json.TypeSafeConvert[string](value)

// JSON utilities
equal, err := json.CompareJson(json1, json2)
merged, err := json.MergeJson(json1, json2)
copy, err := json.DeepCopy(data)
```

---

## 🔧 Configuration

### Create Processor with Custom Config

```go
// Create processor with config
cfg := &json.Config{
    EnableCache:      true,
    MaxCacheSize:     256,
    CacheTTL:         5 * time.Minute,
    MaxJSONSize:      100 * 1024 * 1024, // 100MB
    MaxConcurrency:   50,
    EnableValidation: true,
    CreatePaths:      true,  // Auto-create paths on Set
    CleanupNulls:     true,  // Cleanup nulls after Delete
}

processor := json.New(cfg)
defer processor.Close()

// Use processor methods
result, err := processor.Get(jsonStr, "user.name")
stats := processor.GetStats()
health := processor.GetHealthStatus()
processor.ClearCache()
```

### Preset Configurations

```go
cfg := json.DefaultConfig()    // Balanced default config
cfg := json.SecurityConfig()   // For untrusted input
cfg := json.PrettyConfig()     // For pretty output
```

---

## 📁 Advanced Features

### Data Iteration

```go
// Basic iteration
json.Foreach(data, func(key any, item *json.IterableValue) {
    name := item.GetString("name")
    fmt.Printf("Key: %v, Name: %s\n", key, name)
})

// With path and control flow
json.ForeachWithPathAndControl(data, "users", func(key any, value any) json.IteratorControl {
    if shouldStop {
        return json.IteratorBreak  // Early termination
    }
    return json.IteratorContinue
})
```

### Batch Operations

```go
operations := []json.BatchOperation{
    {Type: "get", Path: "user.name"},
    {Type: "set", Path: "user.age", Value: 25},
    {Type: "delete", Path: "user.temp"},
}
results, err := json.ProcessBatch(operations)
```

### Streaming (Large Files)

```go
// Stream array elements
processor := json.NewStreamingProcessor(reader, 64*1024)
err := processor.StreamArray(func(index int, item any) bool {
    // Process each element
    return true // continue
})

// JSONL processing
jsonlProcessor := json.NewJSONLProcessor(reader)
err := jsonlProcessor.ProcessReader(func(lineNum int, obj map[string]any) error {
    // Process each line
    return nil
})
```

### Schema Validation

```go
schema := &json.Schema{
    Type:     "object",
    Required: []string{"name", "email"},
    Properties: map[string]*json.Schema{
        "name":  {Type: "string", MinLength: 1, MaxLength: 100},
        "email": {Type: "string", Format: "email"},
        "age":   {Type: "integer", Minimum: 0, Maximum: 150},
    },
}

errors, err := json.ValidateSchema(jsonStr, schema)
```

---

## 🎯 Common Use Cases

### API Response Handling

```go
apiResponse := `{
    "status": "success",
    "data": {
        "users": [
            {"id": 1, "name": "Alice", "permissions": ["read", "write"]}
        ],
        "pagination": {"total": 25, "page": 1}
    }
}`

// Quick extraction
status, _ := json.GetString(apiResponse, "status")
total, _ := json.GetInt(apiResponse, "data.pagination.total")

// Extract all user names
names, _ := json.Get(apiResponse, "data.users{name}")
// Result: ["Alice"]

// Flatten all permissions
permissions, _ := json.Get(apiResponse, "data.users{flat:permissions}")
// Result: ["read", "write"]
```

### Configuration Management

```go
config := `{
    "database": {"host": "localhost", "port": 5432},
    "cache": {"enabled": true}
}`

// Type-safe with defaults
dbHost := json.GetDefault(config, "database.host", "localhost")
dbPort := json.GetDefault(config, "database.port", 5432)
cacheEnabled := json.GetDefault(config, "cache.enabled", false)

// Dynamic updates
updated, _ := json.SetMultiple(config, map[string]any{
    "database.host": "prod-db.example.com",
    "cache.ttl":     3600,
})
```

---

## 📊 Performance Monitoring

```go
// Package-level monitoring
stats := json.GetStats()
fmt.Printf("Operations: %d\n", stats.OperationCount)
fmt.Printf("Cache Hit Rate: %.2f%%\n", stats.HitRatio*100)

health := json.GetHealthStatus()
fmt.Printf("Healthy: %v\n", health.Healthy)

// Cache management
json.ClearCache()

// Cache warmup
paths := []string{"user.name", "user.age", "user.profile"}
result, err := json.WarmupCache(jsonStr, paths)
```

---

## Migrating from encoding/json

Simply change the import:

```go
// Before
import "encoding/json"

// After
import "github.com/cybergodev/json"
```

All standard functions are fully compatible:
- `json.Marshal()` / `json.Unmarshal()`
- `json.MarshalIndent()`
- `json.Valid()`
- `json.Compact()` / `json.Indent()` / `json.HTMLEscape()`

---

## 🛡️ Security Configuration

```go
// For handling untrusted JSON input
secureConfig := json.SecurityConfig()
// Features:
// - Full security scanning enabled
// - Conservative size limits
// - Strict mode validation
// - Prototype pollution protection

processor := json.New(secureConfig)
defer processor.Close()
```

---

## Example Code

| File | Description |
|------|-------------|
| [1_basic_usage.go](examples/1_basic_usage.go) | Core operations |
| [2_advanced_features.go](examples/2_advanced_features.go) | Complex paths, file I/O |
| [3_production_ready.go](examples/3_production_ready.go) | Thread-safe patterns |
| [4_error_handling.go](examples/4_error_handling.go) | Error handling patterns |
| [5_encoding_options.go](examples/5_encoding_options.go) | Encoding configuration |
| [10_file_operations.go](examples/10_file_operations.go) | File I/O operations |
| [11_with_defaults.go](examples/11_with_defaults.go) | Default value handling |
| [12_advanced_delete.go](examples/12_advanced_delete.go) | Advanced delete operations |
| [13_streaming_ndjson.go](examples/13_streaming_ndjson.go) | Streaming & JSONL |
| [14_batch_operations.go](examples/14_batch_operations.go) | Batch operations |
| [15_array_append.go](examples/15_array_append.go) | Array append `[+]` |

---

## Documentation

- **[API Reference](docs/API_REFERENCE.md)** - Complete API documentation
- **[Security Guide](docs/SECURITY.md)** - Security best practices
- **[pkg.go.dev](https://pkg.go.dev/github.com/cybergodev/json)** - GoDoc

---

## 📄 License

MIT License - See [LICENSE](LICENSE) file for details.

---

If this project helps you, please give it a Star! ⭐
