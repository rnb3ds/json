# API Reference - cybergodev/json

> Unified API Design for github.com/cybergodev/json library

---

## Table of Contents

1. [API Design Principles](#api-design-principles)
2. [New vs Old API Comparison](#new-vs-old-api-comparison)
3. [Migration Guide](#migration-guide)
4. [Configuration Reference](#configuration-reference)
5. [Phased Execution Plan](#phased-execution-plan)

---

## API Design Principles

### Core Principles

1. **Config Struct Pattern** - Use `Config` struct instead of functional options
2. **DefaultConfig() Entry Point** - Always start from `DefaultConfig()` and modify
3. **Reasonable Defaults** - All configuration items have sensible defaults
4. **Minimal Cognitive Load** - Simple use cases should be simple

### Two-Layer Design

| Layer | Use Case | When to Use |
|-------|----------|-------------|
| **Package Level** | Quick operations | Simple, one-off operations |
| **Processor Level** | Advanced control | Repeated operations, custom config, resource management |

```go
// Package Level - Simple, quick operations
value, err := json.Get(data, "user.name")

// Processor Level - Advanced control
processor := json.New(json.DefaultConfig())
defer processor.Close()
value, err := processor.Get(data, "user.name")
```

---

## New vs Old API Comparison

### 1. Configuration Creation

| Old API | New API | Notes |
|---------|---------|-------|
| `json.New()` | `json.New()` | No change - uses DefaultConfig |
| `json.New(nil)` | `json.New()` | Remove nil parameter |
| `&json.Config{...}` | `json.DefaultConfig()` then modify | Always start from DefaultConfig |
| `json.SecurityConfig()` | `json.SecurityConfig()` | No change - unified security config |
| Compact output | `cfg := json.DefaultConfig(); cfg.IncludeNulls = false` | Use Config |
| Strict validation | `cfg := json.DefaultConfig(); cfg.StrictMode = true` | Use Config |
| Fast processing | `cfg := json.DefaultConfig(); cfg.SkipValidation = true` | Use Config |

### 2. Get Operations

| Old API | New API | Notes |
|---------|---------|-------|
| `json.Get(data, path)` | `json.Get(data, path)` | No change |
| `json.GetString(data, path)` | `json.GetString(data, path)` | No change |
| `json.GetInt(data, path)` | `json.GetInt(data, path)` | No change |
| `json.GetTyped[T](data, path)` | `json.GetTyped[T](data, path)` | No change |
| ~~`json.GetStringWithDefault(data, path, default)`~~ | `json.GetDefault[string](data, path, default)` | Use generic |
| ~~`json.GetIntWithDefault(data, path, default)`~~ | `json.GetDefault[int](data, path, default)` | Use generic |
| ~~`json.GetBoolWithDefault(data, path, default)`~~ | `json.GetDefault[bool](data, path, default)` | Use generic |
| ~~`json.GetTypedWithDefault[T](data, path, default)`~~ | `json.GetDefault[T](data, path, default)` | Renamed |

### 3. Set Operations

| Old API | New API | Notes |
|---------|---------|-------|
| `json.Set(data, path, value)` | `json.Set(data, path, value)` | No change |
| `json.Set(data, path, value, cfg)` | `json.Set(data, path, value, cfg)` | No change |
| ~~`json.SetCreate(data, path, value)`~~ | `cfg := json.DefaultConfig(); cfg.CreatePaths = true; json.Set(data, path, value, cfg)` | Use Config |
| ~~`json.SetCreate(data, path, value, cfg)`~~ | `cfg.CreatePaths = true; json.Set(data, path, value, cfg)` | Set flag in Config |

### 4. Batch Operations

| Old API | New API | Notes |
|---------|---------|-------|
| `json.SetMultiple(data, updates)` | `json.SetMultiple(data, updates)` | No change |
| ~~`json.SetMultipleCreate(data, updates)`~~ | `cfg := json.DefaultConfig(); cfg.CreatePaths = true; json.SetMultiple(data, updates, cfg)` | Use Config |

### 5. Delete Operations

| Old API | New API | Notes |
|---------|---------|-------|
| `json.Delete(data, path)` | `json.Delete(data, path)` | No change |
| ~~`json.DeleteClean(data, path)`~~ | `cfg := json.DefaultConfig(); cfg.CleanupNulls = true; json.Delete(data, path, cfg)` | Use Config |

### 6. Encoding Operations

| Old API | New API | Notes |
|---------|---------|-------|
| `json.Encode(value)` | `json.Encode(value)` | No change |
| `json.Encode(value, cfg)` | `json.Encode(value, cfg)` | No change |
| `json.EncodeWithConfig(value, cfg)` | `json.Encode(value, cfg)` | Consolidated into Encode |
| ~~`json.EncodeWithOpts(value, opts...)`~~ | `json.Encode(value, cfg)` | Use Config |

### 7. File Operations

| Old API | New API | Notes |
|---------|---------|-------|
| `json.LoadFromFile(path)` | `json.LoadFromFile(path)` | No change |
| `json.SaveToFile(path, data)` | `json.SaveToFile(path, data)` | No change |
| `json.SaveToFile(path, data, cfg)` | `json.SaveToFile(path, data, cfg)` | No change |
| ~~`json.SaveToFileWithOpts(path, data, opts...)`~~ | `json.SaveToFile(path, data, cfg)` | Use Config |
| `json.MarshalToFile(path, data, cfg)` | `json.MarshalToFile(path, data, cfg)` | No change |

---

## Migration Guide

### Phase 1: Quick Migration (Minimal Changes)

If you just want to remove deprecation warnings:

```go
// BEFORE (deprecated)
name := json.GetStringWithDefault(data, "user.name", "Anonymous")
result, _ := json.SetCreate(data, "user.profile.name", "Alice")
result, _ := json.DeleteClean(data, "user.temp")

// AFTER (quick fix)
name := json.GetDefault[string](data, "user.name", "Anonymous")

cfg := json.DefaultConfig()
cfg.CreatePaths = true
result, _ := json.Set(data, "user.profile.name", "Alice", cfg)

cfg2 := json.DefaultConfig()
cfg2.CleanupNulls = true
result, _ = json.Delete(data, "user.temp", cfg2)
```

### Phase 2: Recommended Migration (Clean Pattern)

Create reusable config patterns:

```go
// BEFORE (deprecated patterns)
result, _ := json.SetCreate(data, "path.to.value", "new")

// AFTER (recommended pattern)
var createPathConfig = func() *json.Config {
    cfg := json.DefaultConfig()
    cfg.CreatePaths = true
    return cfg
}()

result, _ := json.Set(data, "path.to.value", "new", createPathConfig)
```

### Phase 3: Full Migration (Processor Pattern)

For production code with repeated operations:

```go
// BEFORE (package-level with repeated config)
cfg := json.DefaultConfig()
cfg.CreatePaths = true
cfg.EnableCache = true
result1, _ := json.Set(data1, "path", value1, cfg)
result2, _ := json.Set(data2, "path", value2, cfg)
result3, _ := json.Set(data3, "path", value3, cfg)

// AFTER (processor pattern - recommended for production)
processorCfg := json.DefaultConfig()
processorCfg.CreatePaths = true
processorCfg.EnableCache = true
processor := json.New(processorCfg)
defer processor.Close()

result1, _ := processor.Set(data1, "path", value1)
result2, _ := processor.Set(data2, "path", value2)
result3, _ := processor.Set(data3, "path", value3)
```

### Migration Examples by Use Case

#### 1. Simple Script/Tool

```go
// Old approach - still works but shows deprecation warnings
func processData(data string) string {
    name := json.GetStringWithDefault(data, "user.name", "Unknown")
    age := json.GetIntWithDefault(data, "user.age", 0)
    result, _ := json.SetCreate(data, "processed", true)
    return result
}

// New approach - clean and future-proof
func processData(data string) string {
    name := json.GetDefault[string](data, "user.name", "Unknown")
    age := json.GetDefault[int](data, "user.age", 0)

    cfg := json.DefaultConfig()
    cfg.CreatePaths = true
    result, _ := json.Set(data, "processed", true, cfg)
    return result
}
```

#### 2. Web API Handler

```go
// Old approach
func handleRequest(w http.ResponseWriter, r *http.Request) {
    body, _ := io.ReadAll(r.Body)
    data := string(body)

    // Using deprecated security config
    processor := json.New(json.HighSecurityConfig())
    defer processor.Close()

    email, _ := processor.GetString(data, "user.email")
    // ...
}

// New approach
var secureProcessor *json.Processor

func init() {
    secureProcessor = json.New(json.SecurityConfig())
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
    body, _ := io.ReadAll(r.Body)
    data := string(body)

    email, _ := secureProcessor.GetString(data, "user.email")
    // ...
}
```

#### 3. Batch Processing Service

```go
// Old approach with deprecated methods
func processBatch(items []string) []string {
    results := make([]string, len(items))
    for i, item := range items {
        result, _ := json.SetCreate(item, "processedAt", time.Now().Format(time.RFC3339))
        result, _ = json.DeleteClean(result, "temporary")
        results[i] = result
    }
    return results
}

// New approach with processor pattern
type BatchProcessor struct {
    processor *json.Processor
}

func NewBatchProcessor() *BatchProcessor {
    cfg := json.DefaultConfig()
    cfg.CreatePaths = true
    cfg.CleanupNulls = true
    cfg.EnableCache = true
    return &BatchProcessor{processor: json.New(cfg)}
}

func (bp *BatchProcessor) Close() {
    bp.processor.Close()
}

func (bp *BatchProcessor) Process(items []string) []string {
    results := make([]string, len(items))
    for i, item := range items {
        result, _ := bp.processor.Set(item, "processedAt", time.Now().Format(time.RFC3339))
        result, _ = bp.processor.Delete(result, "temporary")
        results[i] = result
    }
    return results
}
```

---

## Configuration Reference

### Config Struct Overview

```go
type Config struct {
    // ===== Cache Settings =====
    MaxCacheSize int           // Maximum cache entries (default: 128)
    CacheTTL     time.Duration // Cache time-to-live (default: 5 minutes)
    EnableCache  bool          // Enable caching (default: true)
    CacheResults bool          // Cache operation results (default: true)

    // ===== Size Limits =====
    MaxJSONSize  int64 // Maximum JSON size in bytes (default: 100MB)
    MaxPathDepth int   // Maximum path depth (default: 50)
    MaxBatchSize int   // Maximum batch size (default: 2000)

    // ===== Security Limits =====
    MaxNestingDepthSecurity   int   // Maximum nesting depth (default: 200)
    MaxSecurityValidationSize int64 // Max size for security validation (default: 10MB)
    MaxObjectKeys             int   // Maximum object keys (default: 100000)
    MaxArrayElements          int   // Maximum array elements (default: 100000)
    FullSecurityScan          bool  // Full security scan for all input (default: false)

    // ===== Concurrency =====
    MaxConcurrency    int // Maximum concurrent operations (default: 50)
    ParallelThreshold int // Threshold for parallel processing (default: 10)

    // ===== Processing Options =====
    EnableValidation bool // Enable validation (default: true)
    StrictMode       bool // Strict parsing mode (default: false)
    CreatePaths      bool // Auto-create paths in Set operations (default: false)
    CleanupNulls     bool // Clean up null values in Delete operations (default: false)
    CompactArrays    bool // Compact arrays after deletion (default: false)
    ContinueOnError  bool // Continue batch on error (default: false)

    // ===== Input/Output Options =====
    AllowComments    bool // Allow JSON comments (default: false)
    PreserveNumbers  bool // Preserve number precision (default: false)
    ValidateInput    bool // Validate input JSON (default: true)
    ValidateFilePath bool // Validate file paths (default: true)
    SkipValidation   bool // Skip validation for trusted input (default: false)

    // ===== Encoding Options =====
    Pretty          bool            // Pretty print output (default: false)
    Indent          string          // Indentation string (default: "  ")
    Prefix          string          // Line prefix (default: "")
    EscapeHTML      bool            // Escape HTML characters (default: true)
    SortKeys        bool            // Sort object keys (default: false)
    ValidateUTF8    bool            // Validate UTF-8 (default: true)
    MaxDepth        int             // Maximum encoding depth (default: 100)
    DisallowUnknown bool            // Disallow unknown fields (default: false)
    FloatPrecision  int             // Float precision (-1 = default) (default: -1)
    FloatTruncate   bool            // Truncate float precision (default: false)
    DisableEscaping bool            // Disable all escaping (default: false)
    EscapeUnicode   bool            // Escape Unicode characters (default: false)
    EscapeSlash     bool            // Escape forward slash (default: false)
    EscapeNewlines  bool            // Escape newlines (default: true)
    EscapeTabs      bool            // Escape tabs (default: true)
    IncludeNulls    bool            // Include null values (default: true)
    CustomEscapes   map[rune]string // Custom escape mappings (default: nil)

    // ===== Observability =====
    EnableMetrics     bool // Enable metrics collection (default: false)
    EnableHealthCheck bool // Enable health checks (default: false)

    // ===== Context =====
    Context context.Context // Operation context (default: nil)
}
```

### Preset Configurations

```go
// DefaultConfig - General purpose, balanced settings
cfg := json.DefaultConfig()

// SecurityConfig - For untrusted input (public APIs, user data)
cfg := json.SecurityConfig()

// PrettyConfig - For human-readable output
cfg := json.PrettyConfig()
```

### Common Configuration Patterns

```go
// Pattern 1: Auto-create paths
cfg := json.DefaultConfig()
cfg.CreatePaths = true

// Pattern 2: High performance (trusted input only)
cfg := json.DefaultConfig()
cfg.EnableValidation = false
cfg.SkipValidation = true
cfg.EnableCache = true
cfg.MaxCacheSize = 1000

// Pattern 3: Large files
cfg := json.DefaultConfig()
cfg.MaxJSONSize = 500 * 1024 * 1024 // 500MB
cfg.MaxNestingDepthSecurity = 500

// Pattern 4: Streaming/batch processing
cfg := json.DefaultConfig()
cfg.EnableCache = true
cfg.CacheResults = true
cfg.MaxConcurrency = 100
cfg.ParallelThreshold = 5

// Pattern 5: Compact output
cfg := json.DefaultConfig()
cfg.IncludeNulls = false
cfg.Pretty = false

// Pattern 6: Strict validation
cfg := json.DefaultConfig()
cfg.StrictMode = true
cfg.EnableValidation = true
cfg.FullSecurityScan = true
```

---

## Phased Execution Plan

### Phase 1: Documentation & Deprecation Warnings (Current)

**Timeline: v1.x**

**Goal:** Inform users of upcoming changes without breaking existing code

**Actions:**
- [x] Add `Deprecated:` comments to all functions scheduled for removal
- [x] Update documentation with migration examples
- [x] Create this API reference document
- [ ] Add runtime warnings (optional, controlled by environment variable)

**Impact:** No breaking changes, users see deprecation warnings

### Phase 2: Remove Deprecated Functions

**Timeline: v2.0.0**

**Goal:** Clean up API by removing deprecated functions

**Functions to Remove:**
```go
// Get operations
GetStringWithDefault      // Use GetDefault[string]
GetIntWithDefault         // Use GetDefault[int]
GetFloat64WithDefault     // Use GetDefault[float64]
GetBoolWithDefault        // Use GetDefault[bool]
GetArrayWithDefault       // Use GetDefault[[]any]
GetObjectWithDefault      // Use GetDefault[map[string]any]
GetTypedWithDefault       // Use GetDefault[T]

// Set operations
SetCreate                 // Use Set with Config.CreatePaths = true
SetMultipleCreate         // Use SetMultiple with Config.CreatePaths = true

// Delete operations
DeleteClean               // Use Delete with Config.CleanupNulls = true

// Config presets
CompactConfig             // Use DefaultConfig() with modifications
StrictConfig              // Use DefaultConfig() with modifications
FastConfig                // Use DefaultConfig() with modifications

// Encoding operations
EncodeWithOpts            // Use Encode with Config
SaveToFileWithOpts        // Use SaveToFile with Config
MarshalToFileWithOpts     // Use MarshalToFile with Config
SaveToWriterWithOpts      // Use SaveToWriter with Config
EncodeBatchWithOpts       // Use EncodeBatch with Config
EncodeFieldsWithOpts      // Use EncodeFields with Config
EncodeStreamWithOpts      // Use EncodeStream with Config
```

**Migration Steps:**
1. Update all imports to v2
2. Run `go fix` to apply automated migrations
3. Manually update remaining deprecated calls
4. Test thoroughly

### Phase 3: API Consolidation

**Timeline: v2.1.0**

**Goal:** Further simplify the API surface

**Potential Changes:**
- Consolidate `EncodeWithConfig` into `Encode`
- Review and potentially remove underutilized functions
- Simplify configuration patterns

**Example:**
```go
// Simple struct-based configuration
cfg := json.DefaultConfig()
cfg.CreatePaths = true
cfg.Pretty = true
cfg.MaxCacheSize = 1000
```

### Phase 4: Documentation & Examples Update

**Timeline: Ongoing**

**Goal:** Ensure all documentation reflects new API

**Actions:**
- Update all example files
- Update QUICK_REFERENCE.md
- Update README.md
- Add more migration examples
- Create interactive playground/examples

---

## Summary

### What Stays the Same
- Core functions: `Get`, `Set`, `Delete`, `Marshal`, `Unmarshal`
- Type-safe getters: `GetString`, `GetInt`, `GetBool`, `GetFloat64`, `GetArray`, `GetObject`
- Generic getter: `GetTyped[T]`
- Batch operations: `GetMultiple`, `SetMultiple`
- Config struct pattern with `DefaultConfig()`
- Processor pattern for advanced use cases

### What Changes
- Use `GetDefault[T]` instead of type-specific `GetXxxWithDefault`
- Use `Config.CreatePaths = true` instead of `SetCreate`
- Use `Config.CleanupNulls = true` instead of `DeleteClean`
- Use `SecurityConfig()` instead of `HighSecurityConfig()`/`WebAPIConfig()`

### Key Benefits
1. **Reduced API Surface** - Fewer functions to learn
2. **Consistent Patterns** - Same config approach everywhere
3. **Better Discoverability** - Clear entry points (DefaultConfig, SecurityConfig, PrettyConfig)
4. **Future-Proof** - Easier to add new features via Config fields
