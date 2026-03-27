package internal

import (
	"maps"
)

// MergeObjects merges two objects, with the second object taking precedence
func MergeObjects(obj1, obj2 map[string]any) map[string]any {
	// Pre-allocate with combined size to avoid rehashing
	result := make(map[string]any, len(obj1)+len(obj2))

	// Copy from first object
	maps.Copy(result, obj1)

	// Override with second object
	maps.Copy(result, obj2)

	return result
}
