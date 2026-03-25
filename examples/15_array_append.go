//go:build example

// Package main demonstrates the [+] append syntax for array operations
package main

import (
	"fmt"

	"github.com/cybergodev/json"
)

func main() {
	fmt.Println("=== Array Append with [+] Syntax ===")
	fmt.Println()

	// Example 1: Simple array append
	simpleArrayAppend()

	// Example 2: Append to nested arrays
	nestedArrayAppend()

	// Example 3: Append multiple values
	multiValueAppend()

	// Example 4: Append to empty array
	emptyArrayAppend()

	// Example 5: Comparison with old approach
	comparisonExample()
}

func simpleArrayAppend() {
	fmt.Println("--- Simple Array Append ---")

	data := `{"items": ["apple", "banana", "cherry"]}`

	// Append a single value
	result, err := json.Set(data, "items[+]", "date")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	items, _ := json.GetArray(result, "items")
	fmt.Printf("After append: %v\n\n", items)
}

func nestedArrayAppend() {
	fmt.Println("--- Nested Array Append ---")

	data := `{
		"company": {
			"departments": [
				{
					"name": "Engineering",
					"employees": [
						{"name": "Alice", "role": "Lead"},
						{"name": "Bob", "role": "Developer"}
					]
				}
			]
		}
	}`

	newEmployee := map[string]any{
		"name": "Charlie",
		"role": "Senior Developer",
	}

	// Append to nested array in one operation
	result, err := json.Set(data, "company.departments[0].employees[+]", newEmployee)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	names, _ := json.Get(result, "company.departments[0].employees{name}")
	fmt.Printf("Employees after append: %v\n\n", names)
}

func multiValueAppend() {
	fmt.Println("--- Append Multiple Values ---")

	data := `{"numbers": [1, 2, 3]}`

	// Append a slice - values are automatically expanded
	newNumbers := []any{4, 5, 6}
	result, err := json.Set(data, "numbers[+]", newNumbers)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	numbers, _ := json.GetArray(result, "numbers")
	fmt.Printf("After multi-append: %v\n\n", numbers)
}

func emptyArrayAppend() {
	fmt.Println("--- Append to Empty Array ---")

	data := `{"queue": []}`

	// Append to empty array
	result, err := json.Set(data, "queue[+]", "first-item")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	queue, _ := json.GetArray(result, "queue")
	fmt.Printf("Queue after append: %v\n\n", queue)
}

func comparisonExample() {
	fmt.Println("--- Old vs New Approach ---")

	data := `{"users": [{"name": "Alice"}]}`
	newUser := map[string]any{"name": "Bob"}

	// OLD WAY: 3 steps
	fmt.Println("Old way (3 operations):")
	fmt.Println("  1. GetArray(data, \"users\")")
	fmt.Println("  2. append(users, newUser)")
	fmt.Println("  3. Set(data, \"users\", users)")

	// NEW WAY: 1 step
	fmt.Println("\nNew way (1 operation):")
	fmt.Println("  Set(data, \"users[+]\", newUser)")

	result, _ := json.Set(data, "users[+]", newUser)
	names, _ := json.Get(result, "users{name}")
	fmt.Printf("\nResult: %v\n", names)
}
