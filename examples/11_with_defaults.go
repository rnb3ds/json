//go:build example

package main

import (
	"fmt"

	"github.com/cybergodev/json"
)

// With Defaults Example
//
// This example demonstrates using default values with JSON operations
// to handle missing or null values gracefully.
//
// Topics covered:
// - GetDefault[T] - recommended generic method
// - GetWithDefault for any type
// - Practical use cases
//
// Run: go run -tags=example examples/11_with_defaults.go

func main() {
	fmt.Println("JSON Library - With Defaults")
	fmt.Println("============================\n ")

	// Sample data with some missing/optional fields
	partialData := `{
		"user": {
			"name": "Alice",
			"email": "alice@example.com"
		},
		"settings": {
			"theme": "dark"
		}
	}`

	completeData := `{
		"user": {
			"name": "Bob",
			"email": "bob@example.com",
			"age": 30
		},
		"settings": {
			"theme": "light",
			"notifications": true,
			"language": "en"
		}
	}`

	// 1. GETDEFAULT[T] - RECOMMENDED
	demonstrateGetDefault(partialData, completeData)

	// 2. GETWITHDEFAULT
	demonstrateGetWithDefault(partialData, completeData)

	// 3. PRACTICAL USE CASES
	demonstratePracticalCases()

	fmt.Println("\nWith defaults examples complete!")
}

func demonstrateGetDefault(partialData, completeData string) {
	fmt.Println("1. GetDefault[T] (Recommended)")
	fmt.Println("-----------------------------")

	// String with default - RECOMMENDED approach
	email := json.GetDefault(partialData, "user.email", "no-email@example.com")
	fmt.Printf("   user.email: %s\n", email)

	missingPhone := json.GetDefault(partialData, "user.phone", "N/A")
	fmt.Printf("   user.phone (missing): %s\n", missingPhone)

	// Int with default
	age := json.GetDefault(partialData, "user.age", 0)
	fmt.Printf("   user.age (missing): %d\n", age)

	completeAge := json.GetDefault(completeData, "user.age", 0)
	fmt.Printf("   user.age (from complete): %d\n", completeAge)

	// Bool with default
	notifications := json.GetDefault(partialData, "settings.notifications", false)
	fmt.Printf("   settings.notifications (missing): %t\n", notifications)

	// Float with default
	score := json.GetDefault(partialData, "user.score", 100.0)
	fmt.Printf("   user.score (missing): %.1f\n", score)

	// Array with default
	tags := json.GetDefault[[]any](partialData, "user.tags", []any{})
	fmt.Printf("   user.tags (missing): %v (length: %d)\n", tags, len(tags))
}

func demonstrateGetWithDefault(partialData, completeData string) {
	fmt.Println("\n2. GetWithDefault (any type)")
	fmt.Println("----------------------------")

	// Missing field with default
	missingPath := "user.age"
	defaultAge := 18

	age := json.GetWithDefault(partialData, missingPath, defaultAge)
	fmt.Printf("   Missing field '%s': %v (default: %d)\n", missingPath, age, defaultAge)

	// Existing field (returns actual value, not default)
	existingPath := "user.name"
	defaultName := "Unknown"

	name := json.GetWithDefault(partialData, existingPath, defaultName)
	fmt.Printf("   Existing field '%s': %v (default ignored)\n", existingPath, name)

	// Nested path with default
	missingNested := "settings.notifications"
	defaultNotif := false

	notifications := json.GetWithDefault(partialData, missingNested, defaultNotif)
	fmt.Printf("   Missing nested '%s': %v (default: %t)\n", missingNested, notifications, defaultNotif)

	// Show difference with complete data
	fmt.Println("\n   With complete data:")
	completeAge := json.GetWithDefault(completeData, missingPath, defaultAge)
	fmt.Printf("   Field '%s': %v (actual value, default ignored)\n", missingPath, completeAge)
}

func demonstratePracticalCases() {
	fmt.Println("\n3. Practical Use Cases")
	fmt.Println("-----------------------")

	// Use case 1: Configuration with sensible defaults
	configJSON := `{
		"server": {
			"host": "localhost"
		}
	}`

	fmt.Println("   Use Case 1: Configuration defaults")

	type Config struct {
		Host         string
		Port         int
		Debug        bool
		MaxConn      int
		ReadTimeout  int
		WriteTimeout int
	}

	// Extract with defaults using GetDefault[T] (recommended)
	config := Config{
		Host:         json.GetDefault(configJSON, "server.host", "0.0.0.0"),
		Port:         json.GetDefault(configJSON, "server.port", 8080),
		Debug:        json.GetDefault(configJSON, "debug", false),
		MaxConn:      json.GetDefault(configJSON, "max_connections", 100),
		ReadTimeout:  json.GetDefault(configJSON, "read_timeout", 30),
		WriteTimeout: json.GetDefault(configJSON, "write_timeout", 30),
	}

	fmt.Printf("   Config: %+v\n", config)

	// Use case 2: API response handling
	fmt.Println("\n   Use Case 2: API response with optional fields")

	apiResponse := `{
		"status": "success",
		"data": {
			"id": 123,
			"name": "Product Name"
		}
	}`

	// Extract with defaults for optional fields using GetDefault[T]
	status := json.GetDefault(apiResponse, "status", "unknown")
	productID := json.GetDefault(apiResponse, "data.id", 0)
	name := json.GetDefault(apiResponse, "data.name", "Unnamed Product")
	description := json.GetDefault(apiResponse, "data.description", "No description available")
	price := json.GetDefault(apiResponse, "data.price", 0.0)

	fmt.Printf("   Status: %s\n", status)
	fmt.Printf("   Product: %s (ID: %d)\n", name, productID)
	fmt.Printf("   Description: %s\n", description)
	fmt.Printf("   Price: $%.2f\n", price)

	// Use case 3: Feature flags
	fmt.Println("\n   Use Case 3: Feature flags with defaults")

	featuresJSON := `{
		"new_ui": true
	}`

	features := map[string]bool{
		"new_ui":        json.GetDefault(featuresJSON, "new_ui", false),
		"beta_features": json.GetDefault(featuresJSON, "beta_features", false),
		"experimental":  json.GetDefault(featuresJSON, "experimental", false),
		"analytics":     json.GetDefault(featuresJSON, "analytics", true),
		"notifications": json.GetDefault(featuresJSON, "notifications", true),
	}

	fmt.Println("   Feature flags:")
	for name, enabled := range features {
		status := "disabled"
		if enabled {
			status = "enabled"
		}
		fmt.Printf("   - %s: %s\n", name, status)
	}
}
