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
// - GetOr[T] - recommended generic method for defaults
// - Practical use cases
//
// Run: go run -tags=example examples/11_with_defaults.go

func main() {
	fmt.Println("🔧 JSON Library - With Defaults")
	fmt.Println("================================\n ")

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

	// 2. GETTYPEDOR GENERIC
	demonstrateGetTypedOrGeneric(partialData, completeData)

	// 3. PRACTICAL USE CASES
	demonstratePracticalCases()

	fmt.Println("\nWith defaults examples complete!")
}

func demonstrateGetDefault(partialData, completeData string) {
	fmt.Println("1. GetOr[T] (Recommended)")
	fmt.Println("------------------------")

	// String with default - RECOMMENDED approach
	email := json.GetTypedOr(partialData, "user.email", "no-email@example.com")
	fmt.Printf("   user.email: %s\n", email)

	missingPhone := json.GetTypedOr(partialData, "user.phone", "N/A")
	fmt.Printf("   user.phone (missing): %s\n", missingPhone)

	// Int with default
	age := json.GetTypedOr(partialData, "user.age", 0)
	fmt.Printf("   user.age (missing): %d\n", age)

	completeAge := json.GetTypedOr(completeData, "user.age", 0)
	fmt.Printf("   user.age (from complete): %d\n", completeAge)

	// Bool with default
	notifications := json.GetTypedOr(partialData, "settings.notifications", false)
	fmt.Printf("   settings.notifications (missing): %t\n", notifications)

	// Float with default
	score := json.GetTypedOr(partialData, "user.score", 100.0)
	fmt.Printf("   user.score (missing): %.1f\n", score)

	// Array with default
	tags := json.GetTypedOr[[]any](partialData, "user.tags", []any{})
	fmt.Printf("   user.tags (missing): %v (length: %d)\n", tags, len(tags))
}

func demonstrateGetTypedOrGeneric(partialData, completeData string) {
	fmt.Println("\n2. GetTypedOr[T] (generic, type-safe)")
	fmt.Println("-------------------------------------")

	// Missing field with default
	missingPath := "user.age"
	defaultAge := 18

	age := json.GetTypedOr(partialData, missingPath, defaultAge)
	fmt.Printf("   Missing field '%s': %v (default: %d)\n", missingPath, age, defaultAge)

	// Existing field (returns actual value, not default)
	existingPath := "user.name"
	defaultName := "Unknown"

	name := json.GetTypedOr(partialData, existingPath, defaultName)
	fmt.Printf("   Existing field '%s': %v (default ignored)\n", existingPath, name)

	// Nested path with default
	missingNested := "settings.notifications"
	defaultNotif := false

	notifications := json.GetTypedOr(partialData, missingNested, defaultNotif)
	fmt.Printf("   Missing nested '%s': %v (default: %t)\n", missingNested, notifications, defaultNotif)

	// Show difference with complete data
	fmt.Println("\n   With complete data:")
	completeAge := json.GetTypedOr(completeData, missingPath, defaultAge)
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

	// Extract with defaults using GetOr[T] (recommended)
	config := Config{
		Host:         json.GetTypedOr(configJSON, "server.host", "0.0.0.0"),
		Port:         json.GetTypedOr(configJSON, "server.port", 8080),
		Debug:        json.GetTypedOr(configJSON, "debug", false),
		MaxConn:      json.GetTypedOr(configJSON, "max_connections", 100),
		ReadTimeout:  json.GetTypedOr(configJSON, "read_timeout", 30),
		WriteTimeout: json.GetTypedOr(configJSON, "write_timeout", 30),
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

	// Extract with defaults for optional fields using GetOr[T]
	status := json.GetTypedOr(apiResponse, "status", "unknown")
	productID := json.GetTypedOr(apiResponse, "data.id", 0)
	name := json.GetTypedOr(apiResponse, "data.name", "Unnamed Product")
	description := json.GetTypedOr(apiResponse, "data.description", "No description available")
	price := json.GetTypedOr(apiResponse, "data.price", 0.0)

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
		"new_ui":        json.GetTypedOr(featuresJSON, "new_ui", false),
		"beta_features": json.GetTypedOr(featuresJSON, "beta_features", false),
		"experimental":  json.GetTypedOr(featuresJSON, "experimental", false),
		"analytics":     json.GetTypedOr(featuresJSON, "analytics", true),
		"notifications": json.GetTypedOr(featuresJSON, "notifications", true),
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
