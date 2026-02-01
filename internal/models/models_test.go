package models

import (
	"encoding/json"
	"testing"
)

func TestStringListSerialization(t *testing.T) {
	// Test data
	original := StringList{"file1.go", "file2.go", "file3.go"}

	// Test Value() method
	value, err := original.Value()
	if err != nil {
		t.Fatalf("Value() failed: %v", err)
	}

	// Test Scan() method
	var scanned StringList
	err = scanned.Scan(value)
	if err != nil {
		t.Fatalf("Scan() failed: %v", err)
	}

	// Verify round-trip
	if len(scanned) != len(original) {
		t.Fatalf("Length mismatch: got %d, want %d", len(scanned), len(original))
	}

	for i, v := range scanned {
		if v != original[i] {
			t.Errorf("Value mismatch at index %d: got %s, want %s", i, v, original[i])
		}
	}
}

func TestStringListJSON(t *testing.T) {
	// Test JSON marshaling/unmarshaling
	original := StringList{"test1", "test2"}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	var unmarshaled StringList
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if len(unmarshaled) != len(original) {
		t.Fatalf("Length mismatch: got %d, want %d", len(unmarshaled), len(original))
	}

	for i, v := range unmarshaled {
		if v != original[i] {
			t.Errorf("Value mismatch at index %d: got %s, want %s", i, v, original[i])
		}
	}
}
