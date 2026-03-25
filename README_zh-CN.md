# 🚀 cybergodev/json - 高性能 Go JSON 处理库

[![Go Version](https://img.shields.io/badge/Go-1.24+-blue.svg)](https://golang.org)
[![pkg.go.dev](https://pkg.go.dev/badge/github.com/cybergodev/json.svg)](https://pkg.go.dev/github.com/cybergodev/json)
[![MIT license](https://img.shields.io/badge/license-MIT-brightgreen.svg)](https://opensource.org/licenses/MIT)
[![Performance](https://img.shields.io/badge/performance-high%20performance-green.svg)](https://github.com/cybergodev/json)
[![Thread Safe](https://img.shields.io/badge/thread%20safe-yes-brightgreen.svg)](https://github.com/cybergodev/json)

> 一个高性能、功能丰富的 Go JSON 处理库，100% 兼容 `encoding/json`，提供强大的路径操作、类型安全和生产级性能。

**[📖 English Documentation](README.md)**

---

## 🏆 特性

| 特性 | 描述 |
|------|------|
| 🔄 **100% 兼容** | 无需修改代码，直接替换 `encoding/json` 零学习成本 |
| 🎯 **强大路径** | 简洁的路径语法，如 `users[0].name` 轻松访问嵌套数据 |
| 🚀 **高性能** | 智能缓存、内存优化、并发安全 |
| 🛡️ **类型安全** | 泛型支持，编译时类型检查 |
| 🔧 **功能丰富** | 批量操作、流式处理、文件操作、数据验证 |
| 🏗️ **生产就绪** | 线程安全、完善的错误处理、安全特性 |

---

## 📦 安装

```bash
go get github.com/cybergodev/json
```

---

## ⚡ 快速开始（5 分钟）

```go
package main

import (
    "fmt"
    "github.com/cybergodev/json"
)

func main() {
    // 示例 JSON 数据
    data := `{
        "user": {
            "name": "Alice",
            "age": 28,
            "tags": ["premium", "verified"]
        }
    }`

    // 1. 简单字段访问
    name, _ := json.GetString(data, "user.name")
    fmt.Println(name) // "Alice"

    // 2. 类型安全获取
    age, _ := json.GetInt(data, "user.age")
    fmt.Println(age) // 28

    // 3. 负索引访问数组
    lastTag, _ := json.Get(data, "user.tags[-1]")
    fmt.Println(lastTag) // "verified"

    // 4. 修改数据
    updated, _ := json.Set(data, "user.age", 29)
    newAge, _ := json.GetInt(updated, "user.age")
    fmt.Println(newAge) // 29

    // 5. 100% encoding/json 兼容
    bytes, _ := json.Marshal(map[string]any{"status": "ok"})
    fmt.Println(string(bytes)) // {"status":"ok"}
}
```

---

## 📋 路径语法参考

| 语法 | 描述 | 示例 |
|------|------|------|
| `.property` | 访问属性 | `user.name` |
| `[n]` | 数组索引 | `items[0]` |
| `[-n]` | 负索引（从末尾） | `items[-1]`（最后一个元素） |
| `[start:end]` | 数组切片 | `items[1:3]`（第1-2个元素） |
| `[start:end:step]` | 带步长的切片 | `items[::2]`（每隔一个元素） |
| `[+]` | 追加到数组 | `items[+]` |
| `{field}` | 提取数组所有元素的字段 | `users{name}` |
| `{flat:field}` | 扁平化嵌套数组 | `users{flat:tags}` |

---

## 🎯 核心 API

### 数据获取

```go
// 基础获取
json.Get(data, "user.name")           // (any, error)
json.GetString(data, "user.name")     // (string, error)
json.GetInt(data, "user.age")         // (int, error)
json.GetFloat64(data, "user.score")   // (float64, error)
json.GetBool(data, "user.active")     // (bool, error)
json.GetArray(data, "user.tags")      // ([]any, error)
json.GetObject(data, "user.profile")  // (map[string]any, error)

// 类型安全的泛型获取
json.GetTyped[string](data, "user.name")
json.GetTyped[[]int](data, "numbers")
json.GetTyped[User](data, "user")     // 自定义结构体

// 带默认值（推荐用于可能缺失的字段）
json.GetDefault(data, "user.name", "Anonymous")     // string
json.GetDefault(data, "user.age", 0)                // int
json.GetDefault(data, "user.score", 0.0)            // float64
json.GetDefault[[]any](data, "user.tags", []any{})  // 泛型

// 任意类型带默认值
json.GetWithDefault(data, "user.name", "default")

// 批量获取
results, err := json.GetMultiple(data, []string{"user.name", "user.age"})
```

### 数据修改

```go
// 基础设置 - 成功返回修改后的 JSON，失败返回原始数据
result, err := json.Set(data, "user.name", "Bob")

// 使用配置自动创建路径
cfg := json.DefaultConfig()
cfg.CreatePaths = true
result, err := json.Set(data, "user.profile.level", "gold", cfg)

// 追加到数组
result, _ := json.Set(data, "user.tags[+]", "new-tag")

// 批量设置
result, _ := json.SetMultiple(data, map[string]any{
    "user.name": "Bob",
    "user.age":  30,
})

// 删除
result, err := json.Delete(data, "user.temp")
```

### 编码与格式化

```go
// 标准编码（100% 兼容 encoding/json）
bytes, err := json.Marshal(data)
err = json.Unmarshal(bytes, &target)
bytes, err := json.MarshalIndent(data, "", "  ")

// 快速格式化
pretty, _ := json.FormatPretty(jsonStr)    // 美化输出
compact, _ := json.CompactString(jsonStr)  // 压缩

// 直接输出
json.Print(data)        // 压缩格式到 stdout
json.PrintPretty(data)  // 美化格式到 stdout

// 带配置编码
cfg := json.DefaultConfig()
cfg.Pretty = true
cfg.SortKeys = true
result, err := json.Encode(data, cfg)

// 预设配置
result, _ := json.Encode(data, json.PrettyConfig())
```

### 文件操作

```go
// 加载和保存
jsonStr, err := json.LoadFromFile("data.json")
err = json.SaveToFile("output.json", data, json.PrettyConfig())

// 结构体/Map 序列化
err = json.MarshalToFile("user.json", user)
err = json.UnmarshalFromFile("user.json", &user)
```

### 类型转换工具

```go
// 安全类型转换
intVal, ok := json.ConvertToInt(value)
floatVal, ok := json.ConvertToFloat64(value)
boolVal, ok := json.ConvertToBool(value)
strVal := json.ConvertToString(value)

// 泛型转换
result, err := json.TypeSafeConvert[string](value)

// JSON 工具
equal, err := json.CompareJson(json1, json2)
merged, err := json.MergeJson(json1, json2)
copy, err := json.DeepCopy(data)
```

---

## 🔧 配置

### 使用自定义配置创建处理器

```go
// 使用配置创建处理器
cfg := &json.Config{
    EnableCache:      true,
    MaxCacheSize:     256,
    CacheTTL:         5 * time.Minute,
    MaxJSONSize:      100 * 1024 * 1024, // 100MB
    MaxConcurrency:   50,
    EnableValidation: true,
    CreatePaths:      true,  // Set 操作自动创建路径
    CleanupNulls:     true,  // Delete 后清理 null
}

processor := json.New(cfg)
defer processor.Close()

// 使用处理器方法
result, err := processor.Get(jsonStr, "user.name")
stats := processor.GetStats()
health := processor.GetHealthStatus()
processor.ClearCache()
```

### 预设配置

```go
cfg := json.DefaultConfig()    // 平衡的默认配置
cfg := json.SecurityConfig()   // 用于不受信任的输入
cfg := json.PrettyConfig()     // 用于美化输出
```

---

## 📁 高级功能

### 数据迭代

```go
// 基础迭代
json.Foreach(data, func(key any, item *json.IterableValue) {
    name := item.GetString("name")
    fmt.Printf("键: %v, 名称: %s\n", key, name)
})

// 带路径和控制流
json.ForeachWithPathAndControl(data, "users", func(key any, value any) json.IteratorControl {
    if shouldStop {
        return json.IteratorBreak  // 提前终止
    }
    return json.IteratorContinue
})
```

### 批量操作

```go
operations := []json.BatchOperation{
    {Type: "get", Path: "user.name"},
    {Type: "set", Path: "user.age", Value: 25},
    {Type: "delete", Path: "user.temp"},
}
results, err := json.ProcessBatch(operations)
```

### 流式处理（大文件）

```go
// 流式处理数组元素
processor := json.NewStreamingProcessor(reader, 64*1024)
err := processor.StreamArray(func(index int, item any) bool {
    // 处理每个元素
    return true // 继续
})

// JSONL 处理
jsonlProcessor := json.NewJSONLProcessor(reader)
err := jsonlProcessor.ProcessReader(func(lineNum int, obj map[string]any) error {
    // 处理每一行
    return nil
})
```

### 模式验证

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

## 🎯 常见用例

### API 响应处理

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

// 快速提取
status, _ := json.GetString(apiResponse, "status")
total, _ := json.GetInt(apiResponse, "data.pagination.total")

// 提取所有用户名
names, _ := json.Get(apiResponse, "data.users{name}")
// 结果: ["Alice"]

// 扁平化所有权限
permissions, _ := json.Get(apiResponse, "data.users{flat:permissions}")
// 结果: ["read", "write"]
```

### 配置管理

```go
config := `{
    "database": {"host": "localhost", "port": 5432},
    "cache": {"enabled": true}
}`

// 类型安全带默认值
dbHost := json.GetDefault(config, "database.host", "localhost")
dbPort := json.GetDefault(config, "database.port", 5432)
cacheEnabled := json.GetDefault(config, "cache.enabled", false)

// 动态更新
updated, _ := json.SetMultiple(config, map[string]any{
    "database.host": "prod-db.example.com",
    "cache.ttl":     3600,
})
```

---

## 📊 性能监控

```go
// 包级别监控
stats := json.GetStats()
fmt.Printf("操作数: %d\n", stats.OperationCount)
fmt.Printf("缓存命中率: %.2f%%\n", stats.HitRatio*100)

health := json.GetHealthStatus()
fmt.Printf("健康状态: %v\n", health.Healthy)

// 缓存管理
json.ClearCache()

// 缓存预热
paths := []string{"user.name", "user.age", "user.profile"}
result, err := json.WarmupCache(jsonStr, paths)
```

---

## 从 encoding/json 迁移

只需修改导入：

```go
// 之前
import "encoding/json"

// 之后
import "github.com/cybergodev/json"
```

所有标准函数完全兼容：
- `json.Marshal()` / `json.Unmarshal()`
- `json.MarshalIndent()`
- `json.Valid()`
- `json.Compact()` / `json.Indent()` / `json.HTMLEscape()`

---

## 🛡️ 安全配置

```go
// 用于处理不受信任的 JSON 输入
secureConfig := json.SecurityConfig()
// 特性:
// - 启用完整安全扫描
// - 保守的大小限制
// - 严格模式验证
// - 原型污染防护

processor := json.New(secureConfig)
defer processor.Close()
```

---

## 🔄 从 encoding/json 迁移

只需更改导入：

```go
// 之前
import "encoding/json"

// 之后
import "github.com/cybergodev/json"
```

所有标准函数完全兼容：
- `json.Marshal()` / `json.Unmarshal()`
- `json.MarshalIndent()`
- `json.Valid()`
- `json.Compact()` / `json.Indent()` / `json.HTMLEscape()`

---

## 示例代码

| 文件 | 说明 |
|------|------|
| [1_basic_usage.go](examples/1_basic_usage.go) | 核心操作 |
| [2_advanced_features.go](examples/2_advanced_features.go) | 复杂路径、文件 I/O |
| [3_production_ready.go](examples/3_production_ready.go) | 线程安全模式 |
| [4_error_handling.go](examples/4_error_handling.go) | 错误处理模式 |
| [5_encoding_options.go](examples/5_encoding_options.go) | 编码配置 |
| [10_file_operations.go](examples/10_file_operations.go) | 文件 I/O 操作 |
| [11_with_defaults.go](examples/11_with_defaults.go) | 默认值处理 |
| [12_advanced_delete.go](examples/12_advanced_delete.go) | 高级删除操作 |
| [13_streaming_ndjson.go](examples/13_streaming_ndjson.go) | 流式处理 & JSONL |
| [14_batch_operations.go](examples/14_batch_operations.go) | 批量操作 |
| [15_array_append.go](examples/15_array_append.go) | 数组追加 `[+]` |

---

## 文档

- **[API 参考](docs/API_REFERENCE.md)** - 完整 API 文档
- **[安全指南](docs/SECURITY.md)** - 安全最佳实践
- **[pkg.go.dev](https://pkg.go.dev/github.com/cybergodev/json)** - GoDoc

---

## 📄 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件。

---

如果这个项目对你有帮助，请给一个 Star！ ⭐
