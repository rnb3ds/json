# cybergodev/json

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8.svg)](https://golang.org)
[![GoDoc](https://pkg.go.dev/badge/github.com/cybergodev/json.svg)](https://pkg.go.dev/github.com/cybergodev/json)
[![MIT License](https://img.shields.io/badge/License-MIT-green.svg)](https://opensource.org/licenses/MIT)
[![Thread Safe](https://img.shields.io/badge/Thread_Safe-Yes-brightgreen.svg)](https://pkg.go.dev/github.com/cybergodev/json)
[![Security](https://img.shields.io/badge/Security-Hardened-red.svg)](docs/SECURITY.md)
[![Zero Deps](https://img.shields.io/badge/deps-zero-brightgreen.svg)](go.mod)

> A high-performance, feature-rich Go JSON processing library with 100% `encoding/json` compatibility.
> Powerful path syntax, type safety, streaming processing, production-grade performance.

**[中文文档](README_zh-CN.md)**

---

## Why cybergodev/json

| Feature | encoding/json | cybergodev/json |
|--------|---------------|-----------------|
| Path-based access | Manual unmarshal | `json.Get(data, "users[0].name")` |
| Negative index | ❌ | `items[-1]` gets last element |
| Flatten nested arrays | ❌ | `users{flat:tags}` |
| Type-safe defaults | ❌ | `GetStringOr(data, "path", "default")` |
| Streaming large files | ❌ | Built-in streaming processors |
| Schema validation | ❌ | JSON Schema validation |
| Memory pooling | ❌ | `sync.Pool` for hot paths |
| Caching | ❌ | Smart path cache with TTL |
| 100% Compatibility | ✅ Native | Drop-in replacement |

---

## Features

- **100% Compatible** - Drop-in replacement for `encoding/json`, zero learning curve
- **Powerful Paths** - Intuitive syntax: `users[0].name`, `items[-1]`, `data{flat:tags}`
- **High Performance** - Smart caching, memory pooling, optimized hot paths
- **Type Safe** - Generics support with `GetTyped[T]` and `GetTypedOr[T]`
- **Feature Rich** - Batch operations, streaming, file I/O, schema validation, deep merge
- **Production Ready** - Thread-safe, comprehensive error handling, security hardened

---

## Installation

```bash
go get github.com/cybergodev/json
```

**Requirements**: Go 1.25.0 or later

---

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/cybergodev/json"
)

func main() {
    data := `{"user": {"name": "Alice", "age": 28, "tags": ["premium", "verified"]}}`

    // Simple field access
    name, _ := json.GetString(data, "user.name")
    fmt.Println(name) // "Alice"

    // Type-safe retrieval with generics
    age, _ := json.GetTyped[int](data, "user.age")
    fmt.Println(age) // 28

    // With default value (no error on missing path)
    email := json.GetTypedOr[string](data, "user.email", "unknown@example.com")
    fmt.Println(email) // "unknown@example.com"

    // Negative indexing (last element)
    lastTag, _ := json.Get(data, "user.tags[-1]")
    fmt.Println(lastTag) // "verified"

    // Modify data
    updated, _ := json.Set(data, "user.age", 29)
    newAge, _ := json.GetInt(updated, "user.age")
    fmt.Println(newAge) // 29

    // 100% encoding/json compatible
    bytes, _ := json.Marshal(map[string]any{"status": "ok"})
    fmt.Println(string(bytes)) // {"status":"ok"}
}
```

---

## Path Syntax Reference

| Syntax | Description | Example |
|--------|-------------|---------|
| `.property` | Access property | `user.name` → "Alice" |
| `[n]` | Array index | `items[0]` → first element |
| `[-n]` | Negative index (from end) | `items[-1]` → last element |
| `[start:end]` | Array slice | `items[1:3]` → elements 1-2 |
| `[start:end:step]` | Slice with step | `items[::2]` → every other element |
| `[+]` | Append to array | `items[+]` → append position |
| `{field}` | Extract field from all elements | `users{name}` → ["Alice", "Bob"] |
| `{flat:field}` | Flatten nested arrays | `users{flat:tags}` → merge all tags |

---

## Core API

### Data Retrieval

```go
// Basic getters - return (value, error)
json.Get(data, "user.name")            // (any, error)
json.GetString(data, "user.name")      // (string, error)
json.GetInt(data, "user.age")          // (int, error)
json.GetFloat(data, "user.score")      // (float64, error)
json.GetBool(data, "user.active")      // (bool, error)
json.GetArray(data, "user.tags")       // ([]any, error)
json.GetObject(data, "user.profile")   // (map[string]any, error)

// Type-safe generic retrieval
json.GetTyped[string](data, "user.name")
json.GetTyped[[]int](data, "numbers")
json.GetTyped[User](data, "user")      // custom struct

// With defaults (no error when path doesn't exist)
json.GetStringOr(data, "user.name", "Anonymous")
json.GetIntOr(data, "user.age", 0)
json.GetBoolOr(data, "user.active", false)
json.GetFloatOr(data, "user.score", 0.0)
json.GetTypedOr[[]any](data, "user.tags", []any{})

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
// Standard encoding (100% compatible)
bytes, _ := json.Marshal(data)
json.Unmarshal(bytes, &target)
bytes, _ := json.MarshalIndent(data, "", "  ")

// Quick formatting
pretty, _    := json.Prettify(jsonStr)      // pretty print
compact, _   := json.CompactString(jsonStr) // minify
json.Print(data)        // compact to stdout
json.PrintPretty(data)  // pretty to stdout

// Encoding with config
cfg := json.DefaultConfig()
cfg.Pretty = true
cfg.SortKeys = true
result, _ := json.Encode(data, cfg)

// Preset configs
result, _ := json.Encode(data, json.PrettyConfig())
```

### File Operations

```go
// Load and save
jsonStr, _ := json.LoadFromFile("data.json")
json.SaveToFile("output.json", data, json.PrettyConfig())

// Struct/Map serialization
json.MarshalToFile("user.json", user)
json.UnmarshalFromFile("user.json", &user)
```

### Type Conversion Utilities

```go
// Safe type conversion
intVal, ok   := json.ConvertToInt(value)
floatVal, ok := json.ConvertToFloat64(value)
boolVal, ok  := json.ConvertToBool(value)
strVal       := json.ConvertToString(value)

// JSON utilities
equal, _    := json.CompareJSON(json1, json2)
merged, _   := json.MergeJSON(json1, json2)                          // union (default)
merged, _   := json.MergeJSON(json1, json2, json.MergeIntersection)  // intersection
deepCopy, _ := json.DeepCopy(data)
```

---

## Configuration

### Custom Configuration

```go
cfg := json.Config{
    EnableCache:      true,
    MaxCacheSize:     256,
    CacheTTL:         5 * time.Minute,
    MaxJSONSize:      100 * 1024 * 1024, // 100MB
    MaxConcurrency:   50,
    EnableValidation: true,
    CreatePaths:      true,  // auto-create paths on Set
    CleanupNulls:     true,  // cleanup nulls after Delete
}

processor, err := json.New(cfg)
if err != nil {
    // handle configuration error
}
defer processor.Close()

// Use processor methods
result, _ := processor.Get(jsonStr, "user.name")
stats := processor.GetStats()
health := processor.GetHealthStatus()
processor.ClearCache()
```

### Preset Configurations

```go
cfg := json.DefaultConfig()   // balanced defaults
cfg := json.SecurityConfig()  // for untrusted input
cfg := json.PrettyConfig()    // for pretty output
```

---

## Advanced Features

### Data Iteration

```go
// Basic iteration
json.Foreach(data, func(key any, item *json.IterableValue) {
    name := item.GetString("name")
    fmt.Printf("Key: %v, Name: %s\n", key, name)
})

// With path
json.ForeachWithPath(data, "users", func(key any, item *json.IterableValue) {
    name := item.GetString("name")
    fmt.Printf("Key: %v, Name: %s\n", key, name)
})
```

### Batch Operations

```go
data := `{"user": {"name": "Alice", "age": 28, "temp": "value"}}`

operations := []json.BatchOperation{
    {Type: "get", JSONStr: data, Path: "user.name"},
    {Type: "set", JSONStr: data, Path: "user.age", Value: 25},
    {Type: "delete", JSONStr: data, Path: "user.temp"},
}
results, err := json.ProcessBatch(operations)
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

### Encode Utilities

```go
// EncodeStream - encode slice as JSON array
streamJSON, _ := json.EncodeStream(users, json.PrettyConfig())

// EncodeBatch - encode key-value pairs as JSON object
batchJSON, _ := json.EncodeBatch(pairs, cfg)

// EncodeFields - encode only specific fields (filter sensitive data)
fieldsJSON, _ := json.EncodeFields(user, []string{"id", "name", "email"}, cfg)
```

---

## Common Use Cases

### API Response Processing

```go
apiResponse := `{
    "status": "success",
    "data": {
        "users": [{"id": 1, "name": "Alice", "permissions": ["read", "write"]}],
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
dbHost := json.GetStringOr(config, "database.host", "localhost")
dbPort := json.GetIntOr(config, "database.port", 5432)
cacheEnabled := json.GetBoolOr(config, "cache.enabled", false)

// Dynamic update
updated, _ := json.SetMultiple(config, map[string]any{
    "database.host": "prod-db.example.com",
    "cache.ttl":     3600,
})
```

---

## Performance Monitoring

```go
// Package-level monitoring
stats := json.GetStats()
fmt.Printf("Operations: %d\n", stats.OperationCount)
fmt.Printf("Cache Hit Rate: %.2f%%\n", stats.HitRatio*100)

health := json.GetHealthStatus()
fmt.Printf("Health Status: %v\n", health.Healthy)

// Cache management
json.ClearCache()

// Cache warmup
paths := []string{"user.name", "user.age", "user.profile"}
result, _ := json.WarmupCache(jsonStr, paths)
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

## Security Configuration

```go
// For handling untrusted JSON input
secureConfig := json.SecurityConfig()
// Features:
// - Full security scanning enabled
// - Conservative size limits (max 10MB)
// - Strict mode validation
// - Prototype pollution protection
// - Path traversal protection

processor, _ := json.New(secureConfig)
defer processor.Close()
```

---

## Example Code
| File | Description |
|------|-------------|
| [1_basic_usage.go](examples/1_basic_usage.go) | Core operations |
| [2_advanced_features.go](examples/2_advanced_features.go) | Complex paths, nested extraction |
| [3_production_ready.go](examples/3_production_ready.go) | Thread-safe patterns |
| [4_error_handling.go](examples/4_error_handling.go) | Error handling patterns |
| [5_encoding_options.go](examples/5_encoding_options.go) | Encoding configuration |
| [6_validation.go](examples/6_validation.go) | Schema validation |
| [7_type_conversion.go](examples/7_type_conversion.go) | Type conversion |
| [8_helper_functions.go](examples/8_helper_functions.go) | Helper utilities |
| [9_iterator_functions.go](examples/9_iterator_functions.go) | Iteration patterns |
| [10_file_operations.go](examples/10_file_operations.go) | File I/O |
| [11_with_defaults.go](examples/11_with_defaults.go) | Default value handling |
| [12_advanced_delete.go](examples/12_advanced_delete.go) | Delete operations |
| [14_batch_operations.go](examples/14_batch_operations.go) | Batch processing |

```bash
# Run individual examples
go run -tags=example examples/1_basic_usage.go
go run -tags=example examples/2_advanced_features.go
```

---

## Documentation

- **[API Reference](docs/API_REFERENCE.md)** - Complete API documentation
- **[Security Guide](docs/SECURITY.md)** - Security best practices
- **[Quick Reference](docs/QUICK_REFERENCE.md)** - Common patterns at a glance
- **[Compatibility](docs/COMPATIBILITY.md)** - encoding/json compatibility details
- **[pkg.go.dev](https://pkg.go.dev/github.com/cybergodev/json)** - GoDoc

---

## License

MIT License - See [LICENSE](LICENSE) file for details.

---

If this project helps you, please give it a star! ⭐
