# API Reference - cybergodev/json

> Complete API Reference for github.com/cybergodev/json library

---

## Table of Contents

- [Core Functions](#core-functions)
- [Data Retrieval (Get)](#data-retrieval-get)
- [Data Modification (Set)](#data-modification-set)
- [Data Deletion (Delete)](#data-deletion-delete)
- [Encoding Functions](#encoding-functions)
- [Decoding Functions](#decoding-functions)
- [File Operations](#file-operations)
- [Iteration Functions](#iteration-functions)
- [Streaming Processing](#streaming-processing)
- [JSONL Support](#jsonl-support)
- [Validation Functions](#validation-functions)
- [Type Conversion](#type-conversion)
- [Configuration](#configuration)
- [Error Types](#error-types)

---

## Core Functions

### Marshal

```go
func Marshal(v any) ([]byte, error)
```

Encodes a Go value to JSON. 100% compatible with `encoding/json.Marshal`.

**Parameters:**
- `v` - Any Go value to encode

**Returns:**
- `[]byte` - JSON encoded bytes
- `error` - Encoding error if any

**Example:**
```go
data := map[string]any{"name": "Alice", "age": 30}
jsonBytes, err := json.Marshal(data)
```

---

### Unmarshal

```go
func Unmarshal(data []byte, v any) error
```

Decodes JSON bytes into a Go value. 100% compatible with `encoding/json.Unmarshal`.

**Parameters:**
- `data` - JSON bytes to decode
- `v` - Pointer to target value

**Returns:**
- `error` - Decoding error if any

**Example:**
```go
var result map[string]any
err := json.Unmarshal(jsonBytes, &result)
```

---

### MarshalIndent

```go
func MarshalIndent(v any, prefix, indent string) ([]byte, error)
```

Encodes a Go value to formatted JSON with indentation.

**Parameters:**
- `v` - Any Go value to encode
- `prefix` - Prefix for each line
- `indent` - Indentation string

**Example:**
```go
jsonBytes, err := json.MarshalIndent(data, "", "  ")
```

---

### Valid

```go
func Valid(data []byte) bool
```

Reports whether data is valid JSON.

**Parameters:**
- `data` - JSON bytes to validate

**Returns:**
- `bool` - true if valid JSON

---

## Data Retrieval (Get)

### Get

```go
func Get(jsonStr, path string, cfg ...Config) (any, error)
```

Retrieves a value from JSON at the specified path.

**Parameters:**
- `jsonStr` - JSON string
- `path` - Path expression (e.g., "user.name", "items[0]")
- `cfg` - Optional configuration

**Returns:**
- `any` - The value at the path
- `error` - Path or parsing error

**Example:**
```go
value, err := json.Get(data, "users[0].name")
```

---

### GetString

```go
func GetString(jsonStr, path string, cfg ...Config) (string, error)
```

Retrieves a string value from JSON at the specified path.

---

### GetInt

```go
func GetInt(jsonStr, path string, cfg ...Config) (int, error)
```

Retrieves an integer value from JSON at the specified path.

---

### GetFloat

```go
func GetFloat(jsonStr, path string, cfg ...Config) (float64, error)
```

Retrieves a float64 value from JSON at the specified path.

---

### GetBool

```go
func GetBool(jsonStr, path string, cfg ...Config) (bool, error)
```

Retrieves a boolean value from JSON at the specified path.

---

### GetArray

```go
func GetArray(jsonStr, path string, cfg ...Config) ([]any, error)
```

Retrieves an array value from JSON at the specified path.

---

### GetObject

```go
func GetObject(jsonStr, path string, cfg ...Config) (map[string]any, error)
```

Retrieves an object value from JSON at the specified path.

---

### GetTyped[T]

```go
func GetTyped[T any](jsonStr, path string, cfg ...Config) (T, error)
```

Retrieves a typed value from JSON at the specified path using generics.

**Example:**
```go
name, err := json.GetTyped[string](data, "user.name")
numbers, err := json.GetTyped[[]int](data, "scores")
```

---

### GetTypedOr[T]

```go
func GetTypedOr[T any](jsonStr, path string, defaultValue T, cfg ...Config) T
```

Retrieves a typed value with a default fallback if path not found or error occurs.

**Example:**
```go
name := json.GetTypedOr[string](data, "user.name", "Anonymous")
age := json.GetTypedOr[int](data, "user.age", 0)
```

---

### GetMultiple

```go
func GetMultiple(jsonStr string, paths []string, cfg ...Config) (map[string]any, error)
```

Retrieves multiple values from JSON at the specified paths in a single operation.

**Example:**
```go
paths := []string{"user.name", "user.age", "user.email"}
results, err := json.GetMultiple(data, paths)
name := results["user.name"]
```

---

## Data Modification (Set)

### Set

```go
func Set(jsonStr, path string, value any, cfg ...Config) (string, error)
```

Sets a value in JSON at the specified path.

**Parameters:**
- `jsonStr` - JSON string
- `path` - Path expression
- `value` - Value to set
- `cfg` - Optional configuration (use `CreatePaths: true` to auto-create paths)

**Returns:**
- `string` - Modified JSON string
- `error` - Error if operation failed

**Safety Guarantee:** On failure, returns the original unmodified JSON string.

**Example:**
```go
// Basic set
result, err := json.Set(data, "user.name", "Alice")

// Auto-create nested paths
cfg := json.DefaultConfig()
cfg.CreatePaths = true
result, err := json.Set(data, "new.nested.path", "value", cfg)
```

---

### Array Append with [+]

The `[+]` syntax allows appending values to arrays in a single operation:

```go
// Append single value to array
result, err := json.Set(data, "items[+]", "new-item")

// Append object to nested array
newMember := map[string]any{
    "name": "Charlie",
    "role": "Developer",
}
result, err := json.Set(data, "departments[0].teams[0].members[+]", newMember)

// Append multiple values (slice expansion)
moreItems := []any{4, 5, 6}
result, err := json.Set(data, "numbers[+]", moreItems)
// Result: numbers becomes [1, 2, 3, 4, 5, 6]
```

**Comparison with old approach:**
```go
// OLD WAY: 3 operations
members, _ := json.GetArray(data, "users")           // Step 1: Get
members = append(members, newUser)                   // Step 2: Append
result, _ := json.Set(data, "users", members)        // Step 3: Set back

// NEW WAY: 1 operation
result, _ := json.Set(data, "users[+]", newUser)     // Single operation!
```

---

### SetMultiple

```go
func SetMultiple(jsonStr string, updates map[string]any, cfg ...Config) (string, error)
```

Sets multiple values in a single operation.

**Example:**
```go
updates := map[string]any{
    "user.name": "Bob",
    "user.age":  30,
}
result, err := json.SetMultiple(data, updates)
```

---

## Data Deletion (Delete)

### Delete

```go
func Delete(jsonStr, path string, cfg ...Config) (string, error)
```

Deletes a value from JSON at the specified path.

**Parameters:**
- `jsonStr` - JSON string
- `path` - Path expression
- `cfg` - Optional configuration (use `CleanupNulls: true` to remove null values)

**Example:**
```go
result, err := json.Delete(data, "user.temp")

// With null cleanup
cfg := json.DefaultConfig()
cfg.CleanupNulls = true
result, err := json.Delete(data, "user.temp", cfg)
```

---

## Encoding Functions

### Encode

```go
func Encode(value any, cfg ...Config) (string, error)
```

Converts any Go value to JSON string.

**Example:**
```go
result, err := json.Encode(data)
```

---

### EncodeWithConfig

```go
func EncodeWithConfig(value any, cfg Config) (string, error)
```

Converts any Go value to JSON string with custom configuration.

**Example:**
```go
cfg := json.PrettyConfig()
result, err := json.EncodeWithConfig(data, cfg)
```

---

### Prettify

```go
func Prettify(jsonStr string, cfg ...Config) (string, error)
```

Formats JSON string with pretty indentation.

---

### CompactString

```go
func CompactString(jsonStr string, cfg ...Config) (string, error)
```

Removes whitespace from JSON string.

---

## Decoding Functions

### Parse

```go
func Parse(jsonStr string, cfg ...Config) (any, error)
```

Parses a JSON string into a Go value.

**Example:**
```go
data, err := json.Parse(jsonStr)

// With security configuration
cfg := json.SecurityConfig()
data, err := json.Parse(untrustedInput, cfg)
```

---

## File Operations

### LoadFromFile

```go
func LoadFromFile(filePath string, cfg ...Config) (string, error)
```

Loads JSON data from a file.

**Example:**
```go
data, err := json.LoadFromFile("config.json")
```

---

### SaveToFile

```go
func SaveToFile(filePath string, data any, cfg ...Config) error
```

Saves JSON data to a file.

**Example:**
```go
cfg := json.PrettyConfig()
err := json.SaveToFile("output.json", data, cfg)
```

---

### MarshalToFile

```go
func MarshalToFile(filePath string, data any, cfg ...Config) error
```

Marshals data to JSON and writes to a file.

---

### UnmarshalFromFile

```go
func UnmarshalFromFile(path string, v any, cfg ...Config) error
```

Reads JSON from a file and unmarshals it into v.

---

## Iteration Functions

### Foreach

```go
func Foreach(jsonStr string, fn func(key any, item *IterableValue))
```

Iterates over JSON arrays or objects (read-only).

**Example:**
```go
json.Foreach(data, func(key any, item *json.IterableValue) {
    name := item.GetString("name")
    fmt.Printf("Key: %v, Name: %s\n", key, name)
})
```

---

### ForeachWithPath

```go
func ForeachWithPath(jsonStr, path string, fn func(key any, item *IterableValue)) error
```

Iterates over a specific path in JSON.

---

### ForeachNested

```go
func ForeachNested(jsonStr string, fn func(key any, item *IterableValue))
```

Recursively iterates through all nested levels.

---

### ForeachWithPathAndControl

```go
func ForeachWithPathAndControl(jsonStr, path string, fn func(key any, value any) IteratorControl) error
```

Iterates with early termination support using internal `IteratorControl` type.

**Note:** For user-facing iteration with control, prefer `ForeachWithPath` with `IterableValue`:

```go
json.ForeachWithPath(data, "users", func(key any, item *json.IterableValue) {
    name := item.GetString("name")
    if name == "target" {
        // Use item.Break() for early termination in file operations
        return
    }
})
```

---

### ForeachReturn

```go
func ForeachReturn(jsonStr string, fn func(key any, item *IterableValue)) (string, error)
```

Iterates and returns the original JSON string (read-only).

**Note:** This function is read-only. Use `json.Set()` for modifications.

---

## Streaming Processing

### NewStreamingProcessor

```go
func NewStreamingProcessor(reader io.Reader, bufferSize int) *StreamingProcessor
```

Creates a streaming processor for large JSON arrays.

**Example:**
```go
processor := json.NewStreamingProcessor(reader, 64*1024)
err := processor.StreamArray(func(index int, item any) bool {
    fmt.Printf("Item %d: %v\n", index, item)
    return true  // continue
})
```

---

## JSONL Support

### NewJSONLProcessor

```go
func NewJSONLProcessor(reader io.Reader) *JSONLProcessor
```

Creates a processor for JSON Lines (NDJSON) data.

---

### ParseJSONL

```go
func ParseJSONL(data []byte) ([]any, error)
```

Parses JSONL data into a slice of values.

---

### ToJSONL

```go
func ToJSONL(data any) (string, error)
```

Converts a slice to JSONL format.

---

### StreamLinesInto[T]

```go
func StreamLinesInto[T any](reader io.Reader, fn func(lineNum int, data T) error) ([]T, error)
```

Type-safe streaming of JSONL lines.

---

## Validation Functions

### ValidateSchema

```go
func ValidateSchema(jsonStr string, schema *Schema, cfg ...Config) ([]ValidationError, error)
```

Validates JSON data against a schema.

**Example:**
```go
schema := &json.Schema{
    Type: "object",
    Properties: map[string]*json.Schema{
        "name": {Type: "string", MinLength: 1},
        "age":  {Type: "number", Minimum: 0},
    },
    Required: []string{"name"},
}
errors, err := json.ValidateSchema(data, schema)
```

---

### IsValidJSON

```go
func IsValidJSON(jsonStr string) bool
```

Quickly checks if a string is valid JSON.

---

### IsValidPath

```go
func IsValidPath(path string) bool
```

Checks if a path expression is valid.

---

## Type Conversion

### ConvertToInt

```go
func ConvertToInt(value any) (int, bool)
```

Safely converts any value to int.

---

### ConvertToFloat64

```go
func ConvertToFloat64(value any) (float64, bool)
```

Safely converts any value to float64.

---

### ConvertToBool

```go
func ConvertToBool(value any) (bool, bool)
```

Safely converts any value to bool.

---

### ConvertToString

```go
func ConvertToString(value any) string
```

Converts any value to string.

---

## Data Utilities

### DeepCopy

```go
func DeepCopy(data any) (any, error)
```

Creates a deep copy of JSON-compatible data.

---

### CompareJSON

```go
func CompareJSON(json1, json2 string) (bool, error)
```

Compares two JSON strings for semantic equality.

---

### MergeJSON

```go
func MergeJSON(json1, json2 string, mode ...MergeMode) (string, error)
```

Merges two JSON objects using deep merge strategy. The optional mode parameter specifies the merge strategy (defaults to MergeUnion).

**Merge Modes:**
| Mode | Description |
|------|-------------|
| `MergeUnion` | Combines all keys from both objects (default) |
| `MergeIntersection` | Only keys present in both objects |
| `MergeDifference` | Only keys in first object but not in second |

**Example:**
```go
// Union merge (default)
result, err := json.MergeJSON(a, b)

// Union merge (explicit)
result, err := json.MergeJSON(a, b, json.MergeUnion)

// Intersection merge (only common keys)
result, err := json.MergeJSON(a, b, json.MergeIntersection)

// Difference merge (keys only in first)
result, err := json.MergeJSON(a, b, json.MergeDifference)
```

---

### MergeJSONMany

```go
func MergeJSONMany(mode MergeMode, jsons ...string) (string, error)
```

Merges multiple JSON objects with specified merge mode. Requires at least 2 JSON strings.

**Example:**
```go
result, err := json.MergeJSONMany(json.MergeUnion, config1, config2, config3)
```

---

## Configuration

### DefaultConfig

```go
func DefaultConfig() Config
```

Returns the default configuration with balanced settings.

**Default Values:**
| Setting | Value |
|---------|-------|
| MaxJSONSize | 100MB |
| MaxPathDepth | 50 |
| MaxNestingDepthSecurity | 200 |
| MaxObjectKeys | 100,000 |
| MaxArrayElements | 100,000 |
| EnableValidation | true |
| EnableCache | true |
| CacheTTL | 5 minutes |

---

### SecurityConfig

```go
func SecurityConfig() Config
```

Returns a configuration with enhanced security settings for untrusted input.

**Security Values:**
| Setting | Value |
|---------|-------|
| MaxJSONSize | 10MB |
| MaxPathDepth | 30 |
| MaxNestingDepthSecurity | 30 |
| MaxObjectKeys | 5,000 |
| MaxArrayElements | 5,000 |
| FullSecurityScan | true |
| StrictMode | true |

---

### PrettyConfig

```go
func PrettyConfig() Config
```

Returns a configuration for pretty-printed JSON output.

---

### Config Methods

```go
func (c *Config) Clone() *Config
func (c *Config) Validate() error
func (c *Config) GetSecurityLimits() map[string]any
```

---

## Processor Type

### New

```go
func New(config ...*Config) *Processor
```

Creates a new Processor with optional configuration.

**Example:**
```go
processor := json.New()
defer processor.Close()

// Or with configuration
processor := json.New(json.DefaultConfig())
defer processor.Close()
```

---

### Processor Methods

```go
// Core operations
func (p *Processor) Get(jsonStr, path string, opts ...Config) (any, error)
func (p *Processor) Set(jsonStr, path string, value any, opts ...Config) (string, error)
func (p *Processor) Delete(jsonStr, path string, opts ...Config) (string, error)

// Encoding/Decoding
func (p *Processor) Marshal(v any) ([]byte, error)
func (p *Processor) Unmarshal(data []byte, v any, opts ...Config) error
func (p *Processor) EncodeWithConfig(value any, cfg Config) (string, error)

// File operations
func (p *Processor) LoadFromFile(filePath string, opts ...Config) (string, error)
func (p *Processor) SaveToFile(filePath string, data any, cfg ...Config) error

// Schema validation
func (p *Processor) ValidateSchema(jsonStr string, schema *Schema, opts ...Config) ([]ValidationError, error)

// Cache operations
func (p *Processor) ClearCache()
func (p *Processor) GetStats() Stats
func (p *Processor) GetHealthStatus() HealthStatus

// Lifecycle
func (p *Processor) Close() error
```

---

## Error Types

### Standard Errors (encoding/json compatible)

```go
type SyntaxError struct {
    Offset int64
}
type UnmarshalTypeError struct { ... }
type InvalidUnmarshalError struct { ... }
type UnsupportedTypeError struct { ... }
type UnsupportedValueError struct { ... }
type MarshalerError struct { ... }
```

---

### Extended Error Types

```go
type JsonsError struct {
    Op      string  // Operation name
    Path    string  // JSON path
    Message string  // Error message
    Err     error   // Underlying error
}
```

---

### Error Variables

```go
var ErrInvalidJSON       = errors.New("invalid JSON")
var ErrPathNotFound      = errors.New("path not found")
var ErrTypeMismatch      = errors.New("type mismatch")
var ErrSizeLimit         = errors.New("size limit exceeded")
var ErrDepthLimit        = errors.New("depth limit exceeded")
var ErrSecurityViolation = errors.New("security violation")
var ErrOperationFailed   = errors.New("operation failed")
var ErrInvalidPath       = errors.New("invalid path")
```

---

### Error Handling Example

```go
result, err := json.Get(data, "user.name")
if err != nil {
    // Check specific error types
    var jsonsErr *json.JsonsError
    if errors.As(err, &jsonsErr) {
        fmt.Printf("Operation: %s, Path: %s\n", jsonsErr.Op, jsonsErr.Path)
    }

    // Check error codes
    if errors.Is(err, json.ErrPathNotFound) {
        // Handle missing path
    }
}
```

---

## IterableValue Type

```go
type IterableValue struct {
    // Contains methods for safe data access during iteration
}

func (iv *IterableValue) Get(path string) any
func (iv *IterableValue) GetString(key string) string
func (iv *IterableValue) GetInt(key string) int
func (iv *IterableValue) GetFloat64(key string) float64
func (iv *IterableValue) GetBool(key string) bool
func (iv *IterableValue) GetArray(key string) []any
func (iv *IterableValue) GetObject(key string) map[string]any
func (iv *IterableValue) GetWithDefault(key string, defaultValue any) any
func (iv *IterableValue) Exists(key string) bool
func (iv *IterableValue) IsNull(key string) bool
func (iv *IterableValue) IsEmpty(key string) bool
```

---

## Result[T] Type

```go
type Result[T any] struct {
    Value  T
    Exists bool
    Error  error
}

func (r Result[T]) Ok() bool
func (r Result[T]) Unwrap() T
func (r Result[T]) UnwrapOr(defaultValue T) T
func (r Result[T]) Must() T
```

---

## Schema Type

```go
type Schema struct {
    Type                 string             `json:"type,omitempty"`
    Properties           map[string]*Schema `json:"properties,omitempty"`
    Items                *Schema            `json:"items,omitempty"`
    Required             []string           `json:"required,omitempty"`
    MinLength            int                `json:"minLength,omitempty"`
    MaxLength            int                `json:"maxLength,omitempty"`
    Minimum              float64            `json:"minimum,omitempty"`
    Maximum              float64            `json:"maximum,omitempty"`
    Pattern              string             `json:"pattern,omitempty"`
    Format               string             `json:"format,omitempty"`
    AdditionalProperties bool               `json:"additionalProperties,omitempty"`
    MinItems             int                `json:"minItems,omitempty"`
    MaxItems             int                `json:"maxItems,omitempty"`
    UniqueItems          bool               `json:"uniqueItems,omitempty"`
    Enum                 []any              `json:"enum,omitempty"`
}

type ValidationError struct {
    Path    string `json:"path"`
    Message string `json:"message"`
}
```

---

## Iterator Control

The `IteratorControl` type is used internally for iteration flow control.

For user-facing iteration with early termination, use `ForeachFile`, `ForeachFileChunked`, or `ForeachFileObject` with `IterableValue.Break()`:

```go
processor, _ := json.New()
defer processor.Close()

err := processor.ForeachFile("data.json", func(key any, item *json.IterableValue) error {
    if item.GetInt("id") == targetId {
        return item.Break() // stop iteration
    }
    return nil // continue
})
```

**Note:** The `IteratorControl` constants (`IteratorContinue`, `IteratorBreak`) are internal. Use the `IterableValue.Break()` method for user-facing iteration control.

---

## Path Expression Syntax

| Syntax | Description | Example |
|--------|-------------|---------|
| `.` | Property access | `user.name` |
| `[n]` | Array index | `items[0]` |
| `[-n]` | Negative index | `items[-1]` |
| `[start:end]` | Array slice | `items[1:3]` |
| `[::step]` | Step slice | `numbers[::2]` |
| `{field}` | Batch extract | `users{name}` |
| `{flat:field}` | Flatten extract | `users{flat:skills}` |
| `$` | Root reference | `$` |

---

**For more information, see:**
- [Compatibility Guide](./COMPATIBILITY.md)
- [Security Guide](./SECURITY.md)
- [Quick Reference](./QUICK_REFERENCE.md)
