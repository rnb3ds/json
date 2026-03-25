//go:build example

package main

import (
	"fmt"

	"github.com/cybergodev/json"
)

// Type Conversion Example
//
// This example demonstrates comprehensive type conversion capabilities in the cybergodev/json library.
// Learn about safe type conversion, generic type operations, and automatic conversions.
//
// Topics covered:
// - Safe type conversion functions
// - Type-safe generic operations
// - Automatic type conversion
// - JSON Number handling
// - Custom type conversions
//
// Run: go run examples/7_type_conversion.go

func main() {
	fmt.Println("🔄 JSON Library - Type Conversion")
	fmt.Println("=================================\n ")

	// 1. SAFE TYPE CONVERSION
	demonstrateSafeConversion()

	// 2. AUTOMATIC CONVERSION
	demonstrateAutomaticConversion()

	// 3. NUMBER HANDLING
	demonstrateNumberHandling()

	// 4. STRING CONVERSION
	demonstrateStringConversion()

	// 5. BOOL CONVERSION
	demonstrateBoolConversion()

	// 6. TYPE-SAFE GENERICS
	demonstrateGenerics()

	// 7. DEEP COPY
	demonstrateDeepCopy()

	fmt.Println("\n✅ Type conversion examples complete!")
}

func demonstrateSafeConversion() {
	fmt.Println("1️⃣  Safe Type Conversion")
	fmt.Println("─────────────────────────")

	values := []interface{}{
		42,                        // int
		3.14,                      // float64
		"123",                     // string
		true,                      // bool
		json.Number("9876543210"), // json.Number
		int8(10),                  // int8
		int16(20),                 // int16
		int32(30),                 // int32
		int64(40),                 // int64
		uint(50),                  // uint
		uint8(60),                 // uint8
		uint16(70),                // uint16
		uint32(80),                // uint32
		uint64(90),                // uint64
		float32(2.5),              // float32
	}

	fmt.Println("   Converting various types to int:")
	for _, v := range values {
		if result, ok := json.ConvertToInt(v); ok {
			fmt.Printf("   %25v -> %d\n", v, result)
		}
	}

	fmt.Println("\n   Converting various types to float64:")
	for _, v := range values {
		if result, ok := json.ConvertToFloat64(v); ok {
			fmt.Printf("   %25v -> %.2f\n", v, result)
		}
	}

	fmt.Println("\n   Converting various types to bool:")
	boolValues := []interface{}{
		true, false, 1, 0, "true", "false", "TRUE", "FALSE", "1", "0", "T", "F", "yes", "no", "on", "off", 2, 0.0, 0.1,
	}
	for _, v := range boolValues {
		if result, ok := json.ConvertToBool(v); ok {
			fmt.Printf("   %25v -> %t\n", v, result)
		}
	}
}

func demonstrateAutomaticConversion() {
	fmt.Println("\n2️⃣  Automatic Type Conversion")
	fmt.Println("───────────────────────────")

	testJSON := `{
		"intString": "42",
		"floatString": "3.14",
		"boolString": "true",
		"actualInt": 100,
		"actualFloat": 2.718,
		"actualBool": false
	}`

	// Automatic conversion from JSON GetTyped functions
	fmt.Println("   Automatic type conversion with GetTyped:")

	// String to int
	intVal, _ := json.GetInt(testJSON, "intString")
	fmt.Printf("   String '42' -> int: %d\n", intVal)

	// String to float64
	floatVal, _ := json.GetFloat64(testJSON, "floatString")
	fmt.Printf("   String '3.14' -> float64: %.2f\n", floatVal)

	// String to bool
	boolVal, _ := json.GetBool(testJSON, "boolString")
	fmt.Printf("   String 'true' -> bool: %t\n", boolVal)

	// Type-preserving retrieval
	fmt.Println("\n   Type-preserving retrieval:")
	intVal2, _ := json.GetInt(testJSON, "actualInt")
	fmt.Printf("   int 100 -> int: %d (type: %T)\n", intVal2, intVal2)

	floatVal2, _ := json.GetFloat64(testJSON, "actualFloat")
	fmt.Printf("   float64 2.718 -> float64: %.3f (type: %T)\n", floatVal2, floatVal2)
}

func demonstrateNumberHandling() {
	fmt.Println("\n3️⃣  JSON Number Handling")
	fmt.Println("────────────────────────")

	// JSON with number in various formats
	numberJSON := `{
		"integer": 42,
		"largeInteger": 9007199254740992,
		"float": 3.14159,
		"scientific": 1.23e10,
		"negative": -123.45,
		"zero": 0
	}`

	fmt.Println("   Number handling:")

	// Get as int64 for large integers
	largeInt, _ := json.GetInt(numberJSON, "largeInteger")
	fmt.Printf("   Large integer: %d\n", largeInt)

	// Get as float64 for decimals
	floatVal, _ := json.GetFloat64(numberJSON, "float")
	fmt.Printf("   Float value: %.5f\n", floatVal)

	// Get as uint64 for unsigned numbers
	uintVal, _ := json.Get(numberJSON, "integer")
	fmt.Printf("   As any: %v (type: %T)\n", uintVal, uintVal)

	// Convert to string representation
	fmt.Println("\n   Number to string conversion:")
	numbers := map[string]interface{}{
		"int":      42,
		"float":    3.14159,
		"negative": -123,
		"jsonNum":  json.Number("12345"),
	}
	for name, val := range numbers {
		str := json.FormatNumber(val)
		fmt.Printf("   %10s: %v -> '%s'\n", name, val, str)
	}
}

func demonstrateStringConversion() {
	fmt.Println("\n4️⃣  String Conversion")
	fmt.Println("──────────────────────")

	values := []interface{}{
		42,
		3.14,
		true,
		false,
		json.Number("12345"),
	}

	fmt.Println("   Convert to string:")
	for _, v := range values {
		str := json.ConvertToString(v)
		fmt.Printf("   %15v -> '%s'\n", v, str)
	}
}

func demonstrateBoolConversion() {
	fmt.Println("\n5️⃣  Boolean Conversion")
	fmt.Println("───────────────────────")

	fmt.Println("   Boolean conversion truth table:")

	testCases := []struct {
		value interface{}
	}{
		{true},
		{false},
		{1},
		{0},
		{-1},
		{1.0},
		{0.0},
		{"true"},
		{"false"},
		{"TRUE"},
		{"FALSE"},
		{"1"},
		{"0"},
		{"T"},
		{"F"},
		{"yes"},
		{"no"},
		{"on"},
		{"off"},
	}

	for _, tc := range testCases {
		if result, ok := json.ConvertToBool(tc.value); ok {
			fmt.Printf("   %15v -> %t\n", tc.value, result)
		}
	}
}

func demonstrateGenerics() {
	fmt.Println("\n6️⃣  Type-Safe Generic Operations")
	fmt.Println("────────────────────────────────")

	testJSON := `{
		"name": "Alice",
		"age": 30,
		"score": 95.5,
		"active": true,
		"tags": ["go", "json", "developer"]
	}`

	// Type-safe generic operations
	fmt.Println("   Type-safe generic retrieval with GetTyped[T]:")

	// Get as string
	name, err := json.GetTyped[string](testJSON, "name")
	if err == nil {
		fmt.Printf("   Name (string): %s\n", name)
	}

	// Get as int
	age, err := json.GetTyped[int](testJSON, "age")
	if err == nil {
		fmt.Printf("   Age (int): %d\n", age)
	}

	// Get as float64
	score, err := json.GetTyped[float64](testJSON, "score")
	if err == nil {
		fmt.Printf("   Score (float64): %.1f\n", score)
	}

	// Get as bool
	active, err := json.GetTyped[bool](testJSON, "active")
	if err == nil {
		fmt.Printf("   Active (bool): %t\n", active)
	}

	// Get as array
	tags, err := json.GetTyped[[]interface{}](testJSON, "tags")
	if err == nil {
		fmt.Printf("   Tags ([]interface{}): %v\n", tags)
	}

	// Get as object
	user, err := json.GetTyped[map[string]interface{}](testJSON, "")
	if err == nil {
		fmt.Printf("   Full user (map): %v\n", user)
	}
}

func demonstrateDeepCopy() {
	fmt.Println("\n7️⃣  Deep Copy with Type Preservation")
	fmt.Println("──────────────────────────────────")

	original := map[string]interface{}{
		"name": "Bob",
		"age":  25,
		"address": map[string]interface{}{
			"street": "123 Main St",
			"city":   "Springfield",
		},
		"hobbies": []interface{}{"reading", "coding"},
	}

	fmt.Println("   Deep copy demonstration:")

	// Create deep copy
	copied, err := json.DeepCopy(original)
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
		return
	}

	fmt.Printf("   Original: %v\n", original)
	fmt.Printf("   Copy:     %v\n", copied)

	// Modify copy to prove it's deep
	if copiedMap, ok := copied.(map[string]interface{}); ok {
		if addr, ok := copiedMap["address"].(map[string]interface{}); ok {
			addr["city"] = "New City"
		}
		if hobbies, ok := copiedMap["hobbies"].([]interface{}); ok {
			hobbies[0] = "writing"
		}
	}

	fmt.Println("\n   After modifying copy:")
	fmt.Printf("   Original address city: %v\n", original["address"].(map[string]interface{})["city"])
	fmt.Printf("   Copy address city:     %v\n", copied.(map[string]interface{})["address"].(map[string]interface{})["city"])

	fmt.Println("\n   ✓ Original is unchanged (deep copy successful)")
}
