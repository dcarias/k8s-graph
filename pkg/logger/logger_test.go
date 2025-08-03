package logger

import (
	"os"
	"testing"
)

func TestLogLevelString(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{DEBUG, "DEBUG"},
		{INFO, "INFO"},
		{WARN, "WARN"},
		{ERROR, "ERROR"},
		{LogLevel(99), "UNKNOWN"},
	}

	for _, test := range tests {
		result := test.level.String()
		if result != test.expected {
			t.Errorf("Expected LogLevel(%d).String() to return '%s', got '%s'", test.level, test.expected, result)
		}
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected LogLevel
	}{
		{"DEBUG", DEBUG},
		{"debug", DEBUG},
		{"Debug", DEBUG},
		{"INFO", INFO},
		{"info", INFO},
		{"Info", INFO},
		{"WARN", WARN},
		{"warn", WARN},
		{"WARNING", WARN},
		{"warning", WARN},
		{"ERROR", ERROR},
		{"error", ERROR},
		{"Error", ERROR},
		{"", INFO},        // empty string defaults to INFO
		{"INVALID", INFO}, // invalid string defaults to INFO
		{"UNKNOWN", INFO}, // unknown string defaults to INFO
	}

	for _, test := range tests {
		result := ParseLogLevel(test.input)
		if result != test.expected {
			t.Errorf("Expected ParseLogLevel('%s') to return %d, got %d", test.input, test.expected, result)
		}
	}
}

func TestInit(t *testing.T) {
	// Test initialization with different levels
	testLevels := []LogLevel{DEBUG, INFO, WARN, ERROR}

	for _, level := range testLevels {
		Init(level)
		currentLevel := GetLevel()
		if currentLevel != level {
			t.Errorf("Expected GetLevel() to return %d after Init(%d), got %d", level, level, currentLevel)
		}
	}
}

func TestInitFromEnv(t *testing.T) {
	// Test with environment variable set
	os.Setenv("KUBEGRAPH_LOG_LEVEL", "DEBUG")
	InitFromEnv()
	if GetLevel() != DEBUG {
		t.Errorf("Expected GetLevel() to return DEBUG when KUBEGRAPH_LOG_LEVEL=DEBUG, got %d", GetLevel())
	}

	// Test with environment variable not set (should default to INFO)
	os.Unsetenv("KUBEGRAPH_LOG_LEVEL")
	InitFromEnv()
	if GetLevel() != INFO {
		t.Errorf("Expected GetLevel() to return INFO when KUBEGRAPH_LOG_LEVEL is not set, got %d", GetLevel())
	}

	// Test with invalid environment variable (should default to INFO)
	os.Setenv("KUBEGRAPH_LOG_LEVEL", "INVALID")
	InitFromEnv()
	if GetLevel() != INFO {
		t.Errorf("Expected GetLevel() to return INFO when KUBEGRAPH_LOG_LEVEL=INVALID, got %d", GetLevel())
	}
}

func TestIsDebugEnabled(t *testing.T) {
	// Test when DEBUG level is set
	Init(DEBUG)
	if !IsDebugEnabled() {
		t.Error("Expected IsDebugEnabled() to return true when level is DEBUG")
	}

	// Test when INFO level is set
	Init(INFO)
	if IsDebugEnabled() {
		t.Error("Expected IsDebugEnabled() to return false when level is INFO")
	}

	// Test when WARN level is set
	Init(WARN)
	if IsDebugEnabled() {
		t.Error("Expected IsDebugEnabled() to return false when level is WARN")
	}

	// Test when ERROR level is set
	Init(ERROR)
	if IsDebugEnabled() {
		t.Error("Expected IsDebugEnabled() to return false when level is ERROR")
	}
}

func TestLoggingFunctions(t *testing.T) {
	// These tests verify that the logging functions don't panic
	// In a real test environment, you might want to capture the output
	// to verify the actual log messages, but for unit tests, we'll just
	// ensure they don't crash

	Init(DEBUG)

	// Test all logging functions
	Debug("Debug message")
	Info("Info message")
	Warn("Warning message")
	Error("Error message")

	// Test with format strings
	Debug("Debug message with %s", "format")
	Info("Info message with %s", "format")
	Warn("Warning message with %s", "format")
	Error("Error message with %s", "format")

	// Test with multiple arguments
	Debug("Debug message with %s and %d", "string", 42)
	Info("Info message with %s and %d", "string", 42)
	Warn("Warning message with %s and %d", "string", 42)
	Error("Error message with %s and %d", "string", 42)
}

func TestShouldLog(t *testing.T) {
	// Test DEBUG level
	Init(DEBUG)
	if !shouldLog(DEBUG) {
		t.Error("Expected shouldLog(DEBUG) to return true when level is DEBUG")
	}
	if !shouldLog(INFO) {
		t.Error("Expected shouldLog(INFO) to return true when level is DEBUG")
	}
	if !shouldLog(WARN) {
		t.Error("Expected shouldLog(WARN) to return true when level is DEBUG")
	}
	if !shouldLog(ERROR) {
		t.Error("Expected shouldLog(ERROR) to return true when level is DEBUG")
	}

	// Test INFO level
	Init(INFO)
	if shouldLog(DEBUG) {
		t.Error("Expected shouldLog(DEBUG) to return false when level is INFO")
	}
	if !shouldLog(INFO) {
		t.Error("Expected shouldLog(INFO) to return true when level is INFO")
	}
	if !shouldLog(WARN) {
		t.Error("Expected shouldLog(WARN) to return true when level is INFO")
	}
	if !shouldLog(ERROR) {
		t.Error("Expected shouldLog(ERROR) to return true when level is INFO")
	}

	// Test WARN level
	Init(WARN)
	if shouldLog(DEBUG) {
		t.Error("Expected shouldLog(DEBUG) to return false when level is WARN")
	}
	if shouldLog(INFO) {
		t.Error("Expected shouldLog(INFO) to return false when level is WARN")
	}
	if !shouldLog(WARN) {
		t.Error("Expected shouldLog(WARN) to return true when level is WARN")
	}
	if !shouldLog(ERROR) {
		t.Error("Expected shouldLog(ERROR) to return true when level is WARN")
	}

	// Test ERROR level
	Init(ERROR)
	if shouldLog(DEBUG) {
		t.Error("Expected shouldLog(DEBUG) to return false when level is ERROR")
	}
	if shouldLog(INFO) {
		t.Error("Expected shouldLog(INFO) to return false when level is ERROR")
	}
	if shouldLog(WARN) {
		t.Error("Expected shouldLog(WARN) to return false when level is ERROR")
	}
	if !shouldLog(ERROR) {
		t.Error("Expected shouldLog(ERROR) to return true when level is ERROR")
	}
}
