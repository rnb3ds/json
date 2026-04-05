//go:build example

package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cybergodev/json"
)

// Streaming and NDJSON Processing Example
//
// This example demonstrates streaming processing for large JSON data
// and NDJSON (Newline-Delimited JSON) file processing.
//
// Topics covered:
// - StreamingProcessor for large JSON arrays
// - StreamArray and StreamObject for element-by-element processing
// - StreamArrayChunked for batch processing
// - Custom filtering, mapping, and reducing with StreamingProcessor
// - NDJSONProcessor for line-delimited JSON files
// - LargeFileProcessor for memory-efficient file handling
//
// Run: go run -tags=example examples/13_streaming_ndjson.go

func main() {
	fmt.Println("Streaming & NDJSON Processing - JSON Library")
	fmt.Println("=============================================\n ")

	// 1. STREAMING PROCESSOR BASICS
	demonstrateStreamingProcessor()

	// 2. STREAMING TRANSFORMATIONS
	demonstrateStreamingTransformations()

	// 3. CHUNKED PROCESSING
	demonstrateChunkedProcessing()

	// 4. NDJSON PROCESSING
	demonstrateNDJSON()

	// 5. LARGE FILE PROCESSING
	demonstrateLargeFileProcessing()

	fmt.Println("\nStreaming & NDJSON processing examples complete!")
}

func demonstrateStreamingProcessor() {
	fmt.Println("1. Streaming Processor Basics")
	fmt.Println("------------------------------")

	// Large JSON array - simulating a large data stream
	largeArray := `[
		{"id": 1, "name": "Alice", "active": true},
		{"id": 2, "name": "Bob", "active": false},
		{"id": 3, "name": "Charlie", "active": true},
		{"id": 4, "name": "Diana", "active": true},
		{"id": 5, "name": "Eve", "active": false}
	]`

	reader := strings.NewReader(largeArray)
	processor := json.NewStreamingProcessor(reader, 0)

	fmt.Println("   Streaming array elements:")

	err := processor.StreamArray(func(index int, item any) bool {
		// Process each element one at a time
		if obj, ok := item.(map[string]any); ok {
			name := obj["name"]
			active := obj["active"]
			fmt.Printf("   [%d] %v (active: %v)\n", index, name, active)
		}
		return true // Continue processing
	})

	if err != nil {
		fmt.Printf("   Error: %v\n", err)
	}

	// Get statistics
	stats := processor.GetStats()
	fmt.Printf("\n   Stats: %d items processed\n", stats.ItemsProcessed)
}

func demonstrateStreamingTransformations() {
	fmt.Println("\n2. Streaming Transformations")
	fmt.Println("-----------------------------")

	data := `[
		{"name": "Alice", "score": 85},
		{"name": "Bob", "score": 92},
		{"name": "Charlie", "score": 78},
		{"name": "Diana", "score": 95},
		{"name": "Eve", "score": 88}
	]`

	// Filter: Get only high scores (>= 90) using StreamingProcessor
	reader := strings.NewReader(data)
	processor := json.NewStreamingProcessor(reader, 0)
	var filtered []map[string]any

	err := processor.StreamArray(func(index int, item any) bool {
		if obj, ok := item.(map[string]any); ok {
			if score, ok := obj["score"].(float64); ok && score >= 90 {
				filtered = append(filtered, obj)
			}
		}
		return true
	})
	if err != nil {
		fmt.Printf("   Filter error: %v\n", err)
	} else {
		fmt.Printf("   Filtered (score >= 90): %d items\n", len(filtered))
		for _, obj := range filtered {
			fmt.Printf("   - %v: %v\n", obj["name"], obj["score"])
		}
	}

	// Map: Transform names to uppercase using StreamingProcessor
	reader = strings.NewReader(data)
	processor = json.NewStreamingProcessor(reader, 0)
	var transformed []map[string]any

	err = processor.StreamArray(func(index int, item any) bool {
		if obj, ok := item.(map[string]any); ok {
			newObj := make(map[string]any)
			for k, v := range obj {
				newObj[k] = v
			}
			if name, ok := newObj["name"].(string); ok {
				newObj["name"] = strings.ToUpper(name)
			}
			transformed = append(transformed, newObj)
		}
		return true
	})
	if err != nil {
		fmt.Printf("   Map error: %v\n", err)
	} else {
		fmt.Println("\n   Transformed (uppercase names):")
		for _, obj := range transformed {
			fmt.Printf("   - %v\n", obj["name"])
		}
	}

	// Reduce: Sum all scores using StreamingProcessor
	reader = strings.NewReader(data)
	processor = json.NewStreamingProcessor(reader, 0)
	var sum float64

	err = processor.StreamArray(func(index int, item any) bool {
		if obj, ok := item.(map[string]any); ok {
			if score, ok := obj["score"].(float64); ok {
				sum += score
			}
		}
		return true
	})
	if err != nil {
		fmt.Printf("   Reduce error: %v\n", err)
	} else {
		fmt.Printf("\n   Total score (reduce): %.0f\n", sum)
	}

	// Count: Get array length using StreamingProcessor
	reader = strings.NewReader(data)
	processor = json.NewStreamingProcessor(reader, 0)
	count := 0

	err = processor.StreamArray(func(index int, item any) bool {
		count++
		return true
	})
	if err != nil {
		fmt.Printf("   Count error: %v\n", err)
	} else {
		fmt.Printf("   Array count: %d\n", count)
	}

	// First: Find first matching element using StreamingProcessor
	reader = strings.NewReader(data)
	processor = json.NewStreamingProcessor(reader, 0)
	var first map[string]any
	found := false

	err = processor.StreamArray(func(index int, item any) bool {
		if obj, ok := item.(map[string]any); ok && obj["name"] == "Charlie" {
			first = obj
			found = true
			return false // Stop iteration
		}
		return true
	})
	if err != nil {
		fmt.Printf("   First error: %v\n", err)
	} else if found {
		fmt.Printf("   Found 'Charlie': %v\n", first)
	}

	// Take: Get first N elements using StreamingProcessor
	reader = strings.NewReader(data)
	processor = json.NewStreamingProcessor(reader, 0)
	var taken []map[string]any
	takeN := 3

	err = processor.StreamArray(func(index int, item any) bool {
		if len(taken) >= takeN {
			return false // Stop iteration
		}
		if obj, ok := item.(map[string]any); ok {
			taken = append(taken, obj)
		}
		return true
	})
	if err != nil {
		fmt.Printf("   Take error: %v\n", err)
	} else {
		fmt.Printf("   First 3 elements: %d items\n", len(taken))
	}
}

func demonstrateChunkedProcessing() {
	fmt.Println("\n3. Chunked Processing")
	fmt.Println("----------------------")

	data := `[1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15]`

	reader := strings.NewReader(data)
	processor := json.NewStreamingProcessor(reader, 0)

	// Process in chunks of 5
	chunkNum := 0
	err := processor.StreamArrayChunked(5, func(chunk []any) error {
		chunkNum++
		fmt.Printf("   Chunk %d: %v\n", chunkNum, chunk)
		return nil
	})

	if err != nil {
		fmt.Printf("   Error: %v\n", err)
	}

	// Object streaming
	objectData := `{"a": 1, "b": 2, "c": 3, "d": 4, "e": 5}`
	reader = strings.NewReader(objectData)
	processor = json.NewStreamingProcessor(reader, 0)

	fmt.Println("\n   Streaming object:")
	err = processor.StreamObject(func(key string, value any) bool {
		fmt.Printf("   - %s: %v\n", key, value)
		return true
	})

	if err != nil {
		fmt.Printf("   Error: %v\n", err)
	}
}

func demonstrateNDJSON() {
	fmt.Println("\n4. NDJSON Processing")
	fmt.Println("--------------------")

	// Create a temporary NDJSON file
	tempDir, err := os.MkdirTemp("", "json-ndjson-*")
	if err != nil {
		fmt.Printf("   Error creating temp dir: %v\n", err)
		return
	}
	defer os.RemoveAll(tempDir)

	// NDJSON content (each line is a valid JSON object)
	ndjsonContent := `{"type": "user", "id": 1, "name": "Alice"}
{"type": "user", "id": 2, "name": "Bob"}
{"type": "event", "id": 3, "action": "login", "userId": 1}
{"type": "event", "id": 4, "action": "purchase", "userId": 2}
{"type": "user", "id": 5, "name": "Charlie"}
`

	ndjsonPath := filepath.Join(tempDir, "data.ndjson")
	err = os.WriteFile(ndjsonPath, []byte(ndjsonContent), 0644)
	if err != nil {
		fmt.Printf("   Error writing NDJSON: %v\n", err)
		return
	}

	// Process NDJSON file
	processor := json.NewNDJSONProcessor(64 * 1024)

	userCount := 0
	eventCount := 0

	err = processor.ProcessFile(ndjsonPath, func(lineNum int, obj map[string]any) error {
		switch obj["type"] {
		case "user":
			userCount++
			fmt.Printf("   [Line %d] User: %v\n", lineNum, obj["name"])
		case "event":
			eventCount++
			fmt.Printf("   [Line %d] Event: %v\n", lineNum, obj["action"])
		}
		return nil
	})

	if err != nil {
		fmt.Printf("   Error processing NDJSON: %v\n", err)
	}

	fmt.Printf("\n   Summary: %d users, %d events\n", userCount, eventCount)

	// Process from reader
	fmt.Println("\n   Processing from reader:")
	reader := strings.NewReader(ndjsonContent)
	err = processor.ProcessReader(reader, func(lineNum int, obj map[string]any) error {
		if lineNum <= 2 { // Just show first 2 lines
			fmt.Printf("   [%d] type=%v\n", lineNum, obj["type"])
		}
		return nil
	})

	if err != nil {
		fmt.Printf("   Error: %v\n", err)
	}
}

func demonstrateLargeFileProcessing() {
	fmt.Println("\n6. Large File Processing")
	fmt.Println("------------------------")

	// Create a temporary large JSON file
	tempDir, err := os.MkdirTemp("", "json-large-*")
	if err != nil {
		fmt.Printf("   Error creating temp dir: %v\n", err)
		return
	}
	defer os.RemoveAll(tempDir)

	// Create a JSON array with many items
	var buffer bytes.Buffer
	buffer.WriteString("[")
	for i := 0; i < 100; i++ {
		if i > 0 {
			buffer.WriteString(",")
		}
		buffer.WriteString(fmt.Sprintf(`{"id":%d,"value":"item%d"}`, i, i))
	}
	buffer.WriteString("]")

	largeFilePath := filepath.Join(tempDir, "large.json")
	err = os.WriteFile(largeFilePath, buffer.Bytes(), 0644)
	if err != nil {
		fmt.Printf("   Error writing large file: %v\n", err)
		return
	}

	// Process large file using unified Processor API
	processor, err := json.New()
	if err != nil {
		fmt.Printf("   Error creating processor: %v\n", err)
		return
	}

	count := 0
	err = processor.ForeachFile(largeFilePath, func(key any, item *json.IterableValue) error {
		count++
		// Demonstrate accessing fields with IterableValue
		if count <= 3 {
			id := item.GetInt("id")
			value := item.GetString("value")
			fmt.Printf("      Item %d: id=%d, value=%s\n", count, id, value)
		}
		return nil
	})

	if err != nil {
		fmt.Printf("   Error processing large file: %v\n", err)
	} else {
		fmt.Printf("   Processed %d items from large file\n", count)
	}

	// Process in chunks with ForeachFileChunked
	fmt.Println("\n   Chunked processing:")
	chunkCount := 0
	err = processor.ForeachFileChunked(largeFilePath, 25, func(chunk []*json.IterableValue) error {
		chunkCount++
		fmt.Printf("   Processed chunk %d with %d items\n", chunkCount, len(chunk))
		return nil
	})

	if err != nil {
		fmt.Printf("   Error: %v\n", err)
	}
}
