# 🚀 cybergodev/json - High-Performance Go JSON Processing Library

[![Go Version](https://img.shields.io/badge/Go-1.24+-blue.svg)](https://golang.org)
[![pkg.go.dev](https://pkg.go.dev/badge/github.com/cybergodev/json.svg)](https://pkg.go.dev/github.com/cybergodev/json)
[![MIT license](https://img.shields.io/badge/license-MIT-brightgreen.svg)](https://opensource.org/licenses/MIT)
[![Performance](https://img.shields.io/badge/performance-high%20performance-green.svg)](https://github.com/cybergodev/json)
[![Thread Safe](https://img.shields.io/badge/thread%20safe-yes-brightgreen.svg)](https://github.com/cybergodev/json)

> A high-performance, feature-rich Go JSON processing library with 100% `encoding/json` compatibility, providing powerful path operations, type safety, and production-ready performance.

**[📖 中文文档](README_zh-CN.md)**

---

## 🏆 Why cybergodev/json?

| Feature | Description |
|---------|-------------|
| 🔄 **100% Compatible** | Drop-in replacement for `encoding/json` - zero learning curve |
| 🎯 **Powerful Paths** | JSONPath-like syntax for complex data extraction in one line |
| 🚀 **High Performance** | Smart caching, memory optimization, concurrent safety |
| 🛡️ **Type Safe** | Generics support with compile-time type checking |
| 🔧 **Feature Rich** | Batch operations, streaming, file handling, validation |
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

    // 3. Array access with negative index
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
| `[start:end:step]` | Array slice with step | `items[::2]` (every 2nd element) |
| `{field}` | Extract field from all array elements | `users{name}` |
| `{flat:field}` | Flatten nested arrays | `users{flat:tags}` |

---

## 🎯 Core API

### Data Retrieval

```go
// Basic retrieval
json.Get(data, "user.name")           // any
json.GetString(data, "user.name")     // string
json.GetInt(data, "user.age")         // int
json.GetFloat64(data, "user.score")   // float64
json.GetBool(data, "user.active")     // bool
json.GetArray(data, "user.tags")      // []any
json.GetObject(data, "user.profile")  // map[string]any

// Type-safe generic retrieval
json.GetTyped[string](data, "user.name")
json.GetTyped[[]int](data, "numbers")
json.GetTyped[User](data, "user")     // Custom struct

// With default values (recommended for missing fields)
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
// Basic set - returns modified JSON on success, original on failure
result, err := json.Set(data, "user.name", "Bob")

// Auto-create paths with config
cfg := json.DefaultConfig()
cfg.CreatePaths = true
result, err := json.Set(data, "user.profile.level", "gold", cfg)

// Batch set
updates := map[string]any{
    "user.name": "Bob",
    "user.age":  30,
}
result, err := json.SetMultiple(data, updates)

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
jsonStr, err := json.Encode(data, json.PrettyConfig())  // Pretty output
pretty, err := json.FormatPretty(jsonStr)                // Format string
compact, err := json.CompactString(jsonStr)              // Minify string

// Direct output
json.Print(data)        // Print compact to stdout
json.PrintPretty(data)  // Print pretty to stdout

// Encode with configuration
cfg := json.DefaultConfig()
cfg.Pretty = true
cfg.SortKeys = true
result, err := json.Encode(data, cfg)

// Buffer operations (encoding/json compatible)
json.Compact(dst, src)
json.Indent(dst, src, prefix, indent)
json.HTMLEscape(dst, src)
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

### Processor with Custom Config

```go
// Create processor with configuration
cfg := &json.Config{
    EnableCache:      true,
    MaxCacheSize:     256,
    CacheTTL:         5 * time.Minute,
    MaxJSONSize:      100 * 1024 * 1024, // 100MB
    MaxConcurrency:   50,
    EnableValidation: true,
    CreatePaths:      true,  // Auto-create paths in Set operations
    CleanupNulls:     true,  // Remove null values after delete
}

processor := json.New(cfg)
defer processor.Close()

// Use processor methods
result, err := processor.Get(jsonStr, "user.name")
stats := processor.GetStats()
health := processor.GetHealthStatus()
processor.ClearCache()
```

### Configuration Presets

```go
// Default configuration
cfg := json.DefaultConfig()

// Security-focused configuration (for untrusted input)
cfg := json.SecurityConfig()

// Pretty output configuration
cfg := json.PrettyConfig()
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
    // Process each item
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

### API Response Processing

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
fmt.Printf("Cache hit ratio: %.2f%%\n", stats.HitRatio*100)

health := json.GetHealthStatus()
fmt.Printf("Healthy: %v\n", health.Healthy)

// Cache management
json.ClearCache()

// Cache warmup
paths := []string{"user.name", "user.age", "user.profile"}
result, err := json.WarmupCache(jsonStr, paths)
```

---

## 🛡️ Security Configuration

```go
// For processing untrusted JSON input
secureConfig := json.SecurityConfig()
// Features:
// - Full security scan enabled
// - Conservative size limits
// - Strict mode validation
// - Protection against prototype pollution

processor := json.New(secureConfig)
defer processor.Close()
```

---

## 📚 Examples & Resources

### Example Files

| File | Description |
|------|-------------|
| [1_basic_usage.go](examples/1_basic_usage.go) | Core operations getting started |
| [2_advanced_features.go](examples/2_advanced_features.go) | Complex paths and file operations |
| [3_production_ready.go](examples/3_production_ready.go) | Thread-safe patterns and monitoring |
| [10_file_operations.go](examples/10_file_operations.go) | File I/O operations |
| [11_with_defaults.go](examples/11_with_defaults.go) | Default value handling |

### Documentation

- **[API Reference](https://pkg.go.dev/github.com/cybergodev/json)** - Complete API documentation
- **[Security Guide](docs/SECURITY.md)** - Security best practices

---

## 🔄 Migration from encoding/json

Simply change the import:

```go
// Before
import "encoding/json"

// After
import "github.com/cybergodev/json"
```

All standard functions work identically:
- `json.Marshal()` / `json.Unmarshal()`
- `json.MarshalIndent()`
- `json.Valid()`
- `json.Compact()` / `json.Indent()` / `json.HTMLEscape()`

---

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

---

If this project helps you, please give it a Star! ⭐
