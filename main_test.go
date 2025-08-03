package main

import (
	"os"
	"testing"
)

func TestGetEnvBool(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue bool
		expected     bool
	}{
		{"true string", "true", false, true},
		{"TRUE string", "TRUE", false, true},
		{"True string", "True", false, true},
		{"1 string", "1", false, true},
		{"false string", "false", true, false},
		{"FALSE string", "FALSE", true, false},
		{"False string", "False", true, false},
		{"0 string", "0", true, false},
		{"empty string", "", true, true},
		{"invalid string", "invalid", true, true},
		{"invalid string with false default", "invalid", false, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Set environment variable
			if test.envValue != "" {
				os.Setenv("TEST_BOOL", test.envValue)
				defer os.Unsetenv("TEST_BOOL")
			}

			result := getEnvBool("TEST_BOOL", test.defaultValue)
			if result != test.expected {
				t.Errorf("Expected getEnvBool('TEST_BOOL', %t) to return %t, got %t", test.defaultValue, test.expected, result)
			}
		})
	}
}

func TestGetEnvInt(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue int
		expected     int
	}{
		{"valid integer", "42", 0, 42},
		{"zero", "0", 10, 0},
		{"negative integer", "-5", 0, -5},
		{"large integer", "999999", 0, 999999},
		{"empty string", "", 100, 100},
		{"invalid string", "not-a-number", 50, 50},
		{"float string", "3.14", 10, 10},   // Should return default for non-integer
		{"mixed string", "123abc", 20, 20}, // Should return default for non-integer
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Set environment variable
			if test.envValue != "" {
				os.Setenv("TEST_INT", test.envValue)
				defer os.Unsetenv("TEST_INT")
			}

			result := getEnvInt("TEST_INT", test.defaultValue)
			if result != test.expected {
				t.Errorf("Expected getEnvInt('TEST_INT', %d) to return %d, got %d", test.defaultValue, test.expected, result)
			}
		})
	}
}

func TestGetEnvBoolWithUnsetEnv(t *testing.T) {
	// Test with unset environment variable
	result := getEnvBool("UNSET_BOOL_VAR", true)
	if result != true {
		t.Errorf("Expected getEnvBool('UNSET_BOOL_VAR', true) to return true, got %t", result)
	}

	result = getEnvBool("UNSET_BOOL_VAR", false)
	if result != false {
		t.Errorf("Expected getEnvBool('UNSET_BOOL_VAR', false) to return false, got %t", result)
	}
}

func TestGetEnvIntWithUnsetEnv(t *testing.T) {
	// Test with unset environment variable
	result := getEnvInt("UNSET_INT_VAR", 42)
	if result != 42 {
		t.Errorf("Expected getEnvInt('UNSET_INT_VAR', 42) to return 42, got %d", result)
	}

	result = getEnvInt("UNSET_INT_VAR", -10)
	if result != -10 {
		t.Errorf("Expected getEnvInt('UNSET_INT_VAR', -10) to return -10, got %d", result)
	}
}

func TestGetEnvBoolEdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue bool
		expected     bool
	}{
		{"space before true", " true", false, true},
		{"space after true", "true ", false, true},
		{"space before and after true", " true ", false, true},
		{"space before false", " false", true, false},
		{"space after false", "false ", true, false},
		{"space before and after false", " false ", true, false},
		{"space before 1", " 1", false, true},
		{"space after 1", "1 ", false, true},
		{"space before 0", " 0", true, false},
		{"space after 0", "0 ", true, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			os.Setenv("TEST_BOOL_EDGE", test.envValue)
			defer os.Unsetenv("TEST_BOOL_EDGE")

			result := getEnvBool("TEST_BOOL_EDGE", test.defaultValue)
			if result != test.expected {
				t.Errorf("Expected getEnvBool('TEST_BOOL_EDGE', %t) to return %t, got %t", test.defaultValue, test.expected, result)
			}
		})
	}
}

func TestGetEnvIntEdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue int
		expected     int
	}{
		{"space before number", " 42", 0, 42},
		{"space after number", "42 ", 0, 42},
		{"space before and after number", " 42 ", 0, 42},
		{"tab before number", "\t42", 0, 42},
		{"tab after number", "42\t", 0, 42},
		{"newline before number", "\n42", 0, 42},
		{"newline after number", "42\n", 0, 42},
		{"space before invalid", " not-a-number", 50, 50},
		{"space after invalid", "not-a-number ", 50, 50},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			os.Setenv("TEST_INT_EDGE", test.envValue)
			defer os.Unsetenv("TEST_INT_EDGE")

			result := getEnvInt("TEST_INT_EDGE", test.defaultValue)
			if result != test.expected {
				t.Errorf("Expected getEnvInt('TEST_INT_EDGE', %d) to return %d, got %d", test.defaultValue, test.expected, result)
			}
		})
	}
}
