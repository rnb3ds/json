package json

import (
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"time"
)

// testHelper provides utilities for testing JSON operations
type testHelper struct {
	t *testing.T
}

// testProcessorResourcePools tests processor resource pool functionality
func testProcessorResourcePools(processor *Processor) bool {
	// Test string builder pool
	sb := processor.getStringBuilder()
	if sb == nil {
		return false
	}
	processor.putStringBuilder(sb)

	// Test path segments pool
	segments := processor.getPathSegments()
	if segments == nil {
		return false
	}
	processor.putPathSegments(segments)

	return true
}

// newTestHelper creates a new test helper
func newTestHelper(t *testing.T) *testHelper {
	return &testHelper{t: t}
}

// AssertEqual checks if two values are equal
func (h *testHelper) AssertEqual(expected, actual any, msgAndArgs ...any) {
	h.t.Helper()
	if !reflect.DeepEqual(expected, actual) {
		msg := "Values are not equal"
		if len(msgAndArgs) > 0 {
			msg = fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...)
		}
		h.t.Errorf("%s\nExpected: %v (%T)\nActual: %v (%T)", msg, expected, expected, actual, actual)
	}
}

// AssertNotEqual checks if two values are not equal
func (h *testHelper) AssertNotEqual(expected, actual any, msgAndArgs ...any) {
	h.t.Helper()
	if reflect.DeepEqual(expected, actual) {
		msg := "Values should not be equal"
		if len(msgAndArgs) > 0 {
			msg = fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...)
		}
		h.t.Errorf("%s\nBoth values: %v (%T)", msg, expected, expected)
	}
}

// AssertNoError checks that error is nil
func (h *testHelper) AssertNoError(err error, msgAndArgs ...any) {
	h.t.Helper()
	if err != nil {
		msg := "Expected no error"
		if len(msgAndArgs) > 0 {
			msg = fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...)
		}
		h.t.Errorf("%s, but got: %v", msg, err)
	}
}

// AssertError checks that error is not nil
func (h *testHelper) AssertError(err error, msgAndArgs ...any) {
	h.t.Helper()
	if err == nil {
		msg := "Expected an error"
		if len(msgAndArgs) > 0 {
			msg = fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...)
		}
		h.t.Error(msg + ", but got nil")
	}
}

// AssertErrorContains checks that error contains specific text
func (h *testHelper) AssertErrorContains(err error, contains string, msgAndArgs ...any) {
	h.t.Helper()
	if err == nil {
		msg := "Expected an error"
		if len(msgAndArgs) > 0 {
			msg = fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...)
		}
		h.t.Error(msg + ", but got nil")
		return
	}
	if !strings.Contains(err.Error(), contains) {
		msg := fmt.Sprintf("Expected error to contain '%s'", contains)
		if len(msgAndArgs) > 0 {
			msg = fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...)
		}
		h.t.Errorf("%s, but got: %v", msg, err)
	}
}

// AssertPanic checks that function panics
func (h *testHelper) AssertPanic(fn func(), msgAndArgs ...any) {
	h.t.Helper()
	defer func() {
		if r := recover(); r == nil {
			msg := "Expected function to panic"
			if len(msgAndArgs) > 0 {
				msg = fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...)
			}
			h.t.Error(msg + ", but it didn't")
		}
	}()
	fn()
}

// AssertNoPanic checks that function doesn't panic
func (h *testHelper) AssertNoPanic(fn func(), msgAndArgs ...any) {
	h.t.Helper()
	defer func() {
		if r := recover(); r != nil {
			msg := "Expected function not to panic"
			if len(msgAndArgs) > 0 {
				msg = fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...)
			}
			h.t.Errorf("%s, but it panicked with: %v", msg, r)
		}
	}()
	fn()
}

// AssertTrue checks that condition is true
func (h *testHelper) AssertTrue(condition bool, msgAndArgs ...any) {
	h.t.Helper()
	if !condition {
		msg := "Expected condition to be true"
		if len(msgAndArgs) > 0 {
			msg = fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...)
		}
		h.t.Error(msg)
	}
}

// AssertFalse checks that condition is false
func (h *testHelper) AssertFalse(condition bool, msgAndArgs ...any) {
	h.t.Helper()
	if condition {
		msg := "Expected condition to be false"
		if len(msgAndArgs) > 0 {
			msg = fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...)
		}
		h.t.Error(msg)
	}
}

// AssertNotNil checks that value is not nil
func (h *testHelper) AssertNotNil(value any, msgAndArgs ...any) {
	h.t.Helper()
	if value == nil {
		msg := "Expected value to be not nil"
		if len(msgAndArgs) > 0 {
			msg = fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...)
		}
		h.t.Error(msg)
	}
}

// AssertNil checks that value is nil
func (h *testHelper) AssertNil(value any, msgAndArgs ...any) {
	h.t.Helper()
	if value != nil {
		msg := "Expected value to be nil"
		if len(msgAndArgs) > 0 {
			msg = fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...)
		}
		h.t.Errorf("%s, but got: %v (%T)", msg, value, value)
	}
}

// testDataGenerator generates test data for various scenarios
type testDataGenerator struct {
	rand *rand.Rand
}

// newTestDataGenerator creates a new test data generator
func newTestDataGenerator() *testDataGenerator {
	return &testDataGenerator{
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}


// generateComplexJSON generates complex nested JSON structures
func (g *testDataGenerator) generateComplexJSON() string {
	return `{
		"users": [
			{
				"id": 1,
				"name": "Alice Johnson",
				"email": "alice@example.com",
				"profile": {
					"age": 28,
					"location": "New York",
					"preferences": {
						"theme": "dark",
						"notifications": true,
						"languages": ["en", "es", "fr"]
					}
				},
				"roles": ["admin", "user"],
				"metadata": {
					"created": "2023-01-15T10:30:00Z",
					"lastLogin": "2024-01-20T14:45:00Z",
					"loginCount": 156
				}
			},
			{
				"id": 2,
				"name": "Bob Smith",
				"email": "bob@example.com",
				"profile": {
					"age": 35,
					"location": "San Francisco",
					"preferences": {
						"theme": "light",
						"notifications": false,
						"languages": ["en"]
					}
				},
				"roles": ["user"],
				"metadata": {
					"created": "2023-03-22T09:15:00Z",
					"lastLogin": "2024-01-19T16:20:00Z",
					"loginCount": 89
				}
			}
		],
		"settings": {
			"appName": "TestApp",
			"version": "1.2.3",
			"features": {
				"authentication": true,
				"logging": true,
				"caching": false
			},
			"limits": {
				"maxUsers": 1000,
				"maxRequests": 10000,
				"timeout": 30.5
			}
		},
		"statistics": {
			"totalUsers": 2,
			"activeUsers": 1,
			"metrics": [
				{"name": "cpu", "value": 45.2, "unit": "%"},
				{"name": "memory", "value": 78.9, "unit": "%"},
				{"name": "disk", "value": 23.1, "unit": "%"}
			]
		}
	}`
}


// concurrencyTester helps test concurrent operations
type concurrencyTester struct {
	t           *testing.T
	concurrency int
	iterations  int
}

// newConcurrencyTester creates a new concurrency tester
func newConcurrencyTester(t *testing.T, concurrency, iterations int) *concurrencyTester {
	return &concurrencyTester{
		t:           t,
		concurrency: concurrency,
		iterations:  iterations,
	}
}

// run runs concurrent test operations
func (ct *concurrencyTester) run(operation func(workerID, iteration int) error) {
	ct.t.Helper()

	done := make(chan error, ct.concurrency)

	for i := 0; i < ct.concurrency; i++ {
		go func(workerID int) {
			for j := 0; j < ct.iterations; j++ {
				if err := operation(workerID, j); err != nil {
					done <- fmt.Errorf("worker %d, iteration %d: %w", workerID, j, err)
					return
				}
			}
			done <- nil
		}(i)
	}

	for i := 0; i < ct.concurrency; i++ {
		if err := <-done; err != nil {
			ct.t.Errorf("Concurrent operation failed: %v", err)
		}
	}
}

// ============================================================================
// Shared Test Data Generators
// ============================================================================

// genJSONArraySize generates a large JSON array wrapped in an object with n items.
// Produces: {"items": [{"id": 0}, {"id": 1}, ...]}
func genJSONArraySize(n int) string {
	var sb strings.Builder
	sb.WriteString(`{"items": [`)
	for i := 0; i < n; i++ {
		if i > 0 {
			sb.WriteByte(',')
		}
		fmt.Fprintf(&sb, `{"id": %d}`, i)
	}
	sb.WriteString("]}")
	return sb.String()
}

// genJSONArrayRaw generates a raw JSON array with n richly-typed objects.
// Produces: [{"id":0,"name":"user0",...}, ...]
func genJSONArrayRaw(n int) string {
	var sb strings.Builder
	sb.WriteByte('[')
	for i := 0; i < n; i++ {
		if i > 0 {
			sb.WriteByte(',')
		}
		fmt.Fprintf(&sb, `{"id":%d,"name":"user%d","email":"user%d@example.com","active":true,"score":%.2f}`,
			i, i, i, float64(i)*1.5)
	}
	sb.WriteByte(']')
	return sb.String()
}

// genJSONObject generates a flat JSON object with n keys.
// Produces: {"key0":{"value":0,...}, ...}
func genJSONObject(n int) string {
	var sb strings.Builder
	sb.WriteByte('{')
	for i := 0; i < n; i++ {
		if i > 0 {
			sb.WriteByte(',')
		}
		fmt.Fprintf(&sb, `"key%d":{"value":%d,"label":"Label %d"}`, i, i, i)
	}
	sb.WriteByte('}')
	return sb.String()
}

// genNestedJSON generates depth-level nested JSON with a fixed key.
// Produces: {"a":{"a":...:"leaf"}}
func genNestedJSON(depth int, leaf string) string {
	var sb strings.Builder
	sb.Grow(depth*6 + len(leaf) + depth)
	for i := 0; i < depth; i++ {
		sb.WriteString(`{"a":`)
	}
	sb.WriteString(`"` + leaf + `"`)
	for i := 0; i < depth; i++ {
		sb.WriteByte('}')
	}
	return sb.String()
}

// genNestedJSONDynamicKeys generates depth-level nested JSON with dynamic keys.
// Produces: {"level0":{"level1":...:"value"}}
func genNestedJSONDynamicKeys(depth int) string {
	var sb strings.Builder
	for i := 0; i < depth; i++ {
		fmt.Fprintf(&sb, `{"level%d":`, i)
	}
	sb.WriteString(`"value"`)
	for i := 0; i < depth; i++ {
		sb.WriteByte('}')
	}
	return sb.String()
}

// genLargeJSONBytes generates JSON of approximately targetSize bytes.
func genLargeJSONBytes(targetSize int) string {
	var sb strings.Builder
	sb.Grow(targetSize + 20)
	sb.WriteString(`{"data": [`)
	item := `{"value":"data"},`
	itemLen := len(item)
	remaining := targetSize - 12
	for remaining >= itemLen {
		sb.WriteString(item)
		remaining -= itemLen
	}
	str := sb.String()
	if len(str) > 0 && str[len(str)-1] == ',' {
		str = str[:len(str)-1]
	}
	return str + `]}`
}

// genUserFragments generates comma-separated user JSON objects.
func genUserFragments(count int) string {
	users := make([]string, count)
	for i := 0; i < count; i++ {
		users[i] = fmt.Sprintf(`{"id": %d, "name": "User%d"}`, i, i)
	}
	return strings.Join(users, ",")
}

// genItemFragments generates comma-separated item JSON objects.
func genItemFragments(count int) string {
	items := make([]string, count)
	for i := 0; i < count; i++ {
		items[i] = fmt.Sprintf(`{"id": %d}`, i)
	}
	return strings.Join(items, ",")
}

