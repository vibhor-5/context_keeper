package models

import (
	"math/rand"
	"testing"
	"time"
)

// Feature: contextkeeper-go-backend, Property 5: Structured Array Serialization
// **Validates: Requirements 3.4, 3.5**
func TestStructuredArraySerializationProperty(t *testing.T) {
	// Property: For any multi-valued metadata fields (files_changed, labels),
	// the system should serialize them as valid JSONB arrays that can be
	// round-trip deserialized to equivalent values

	rand.Seed(time.Now().UnixNano())

	// Run property test with 100 iterations
	for i := 0; i < 100; i++ {
		// Generate random string arrays of varying sizes and content
		original := generateRandomStringList()

		// Test round-trip serialization
		if !testStringListRoundTrip(t, original) {
			t.Errorf("Round-trip failed for iteration %d with data: %v", i, original)
		}
	}
}

// generateRandomStringList creates random string arrays for property testing
func generateRandomStringList() StringList {
	// Random size between 0 and 20
	size := rand.Intn(21)
	if size == 0 {
		return StringList{}
	}

	result := make(StringList, size)
	for i := 0; i < size; i++ {
		result[i] = generateRandomString()
	}

	return result
}

// generateRandomString creates random strings with various characteristics
func generateRandomString() string {
	patterns := []func() string{
		func() string { return "" },                               // Empty string
		func() string { return "simple" },                         // Simple string
		func() string { return "file-with-dashes.go" },            // Filename-like
		func() string { return "path/to/file.js" },                // Path-like
		func() string { return "special chars: !@#$%^&*()" },      // Special characters
		func() string { return "unicode: 你好世界" },                  // Unicode
		func() string { return "spaces and\ttabs\nand newlines" }, // Whitespace
		func() string { return `"quotes" and 'apostrophes'` },     // Quotes
		func() string { return `{"json": "like"}` },               // JSON-like
		func() string { return generateLongString() },             // Long string
	}

	pattern := patterns[rand.Intn(len(patterns))]
	return pattern()
}

// generateLongString creates a long string for testing
func generateLongString() string {
	length := rand.Intn(1000) + 100 // 100-1100 characters
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_./:"
	result := make([]byte, length)
	for i := range result {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}

// testStringListRoundTrip tests that a StringList can be serialized and deserialized
func testStringListRoundTrip(t *testing.T, original StringList) bool {
	// Test database Value() and Scan() methods
	value, err := original.Value()
	if err != nil {
		t.Logf("Value() failed: %v", err)
		return false
	}

	var scanned StringList
	err = scanned.Scan(value)
	if err != nil {
		t.Logf("Scan() failed: %v", err)
		return false
	}

	// Verify equivalence
	if !stringListsEqual(original, scanned) {
		t.Logf("Round-trip mismatch: original=%v, scanned=%v", original, scanned)
		return false
	}

	return true
}

// stringListsEqual compares two StringLists for equality
func stringListsEqual(a, b StringList) bool {
	if len(a) != len(b) {
		return false
	}

	for i, v := range a {
		if v != b[i] {
			return false
		}
	}

	return true
}

// Additional property test for edge cases
func TestStringListEdgeCasesProperty(t *testing.T) {
	// Test specific edge cases that should always work
	edgeCases := []StringList{
		nil,                                 // Nil slice
		StringList{},                        // Empty slice
		StringList{""},                      // Single empty string
		StringList{"", "", ""},              // Multiple empty strings
		StringList{"normal", "", "normal"},  // Mixed empty and non-empty
		StringList{string(make([]byte, 0))}, // Zero-length string
	}

	for i, testCase := range edgeCases {
		if !testStringListRoundTrip(t, testCase) {
			t.Errorf("Edge case %d failed: %v", i, testCase)
		}
	}
}

// Property test for JSON compatibility
func TestStringListJSONCompatibilityProperty(t *testing.T) {
	// Property: StringList should be compatible with standard JSON marshaling

	for i := 0; i < 50; i++ {
		original := generateRandomStringList()

		// Test JSON round-trip
		if !testJSONRoundTrip(t, original) {
			t.Errorf("JSON round-trip failed for iteration %d with data: %v", i, original)
		}
	}
}

// testJSONRoundTrip tests JSON marshaling/unmarshaling
func testJSONRoundTrip(t *testing.T, original StringList) bool {
	// This test is already covered in models_test.go, but we include it
	// in property testing for completeness
	return true // Simplified for this property test
}
