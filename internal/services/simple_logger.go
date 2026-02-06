package services

import (
	"fmt"
	"log"
)

// SimpleLogger implements the Logger interface with basic logging
type SimpleLogger struct{}

// Info logs an info message
func (l *SimpleLogger) Info(message string, fields map[string]interface{}) {
	log.Printf("[INFO] %s %s", message, formatFields(fields))
}

// Error logs an error message
func (l *SimpleLogger) Error(message string, err error, fields map[string]interface{}) {
	log.Printf("[ERROR] %s: %v %s", message, err, formatFields(fields))
}

// Debug logs a debug message
func (l *SimpleLogger) Debug(message string, fields map[string]interface{}) {
	log.Printf("[DEBUG] %s %s", message, formatFields(fields))
}

// Warn logs a warning message
func (l *SimpleLogger) Warn(message string, fields map[string]interface{}) {
	log.Printf("[WARN] %s %s", message, formatFields(fields))
}

// formatFields formats the fields map for logging
func formatFields(fields map[string]interface{}) string {
	if len(fields) == 0 {
		return ""
	}
	
	result := "{"
	first := true
	for key, value := range fields {
		if !first {
			result += ", "
		}
		result += fmt.Sprintf("%s: %v", key, value)
		first = false
	}
	result += "}"
	return result
}