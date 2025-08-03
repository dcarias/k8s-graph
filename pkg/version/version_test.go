package version

import (
	"testing"
)

func TestGetVersionInfo(t *testing.T) {
	info := GetVersionInfo()

	// Check that all expected keys are present
	expectedKeys := []string{"version", "commit", "branch", "full"}
	for _, key := range expectedKeys {
		if _, exists := info[key]; !exists {
			t.Errorf("Expected version info to contain key '%s'", key)
		}
	}

	// Check that values are not empty (except for commit/branch which might be "unknown" in tests)
	if info["version"] == "" {
		t.Error("Expected version to not be empty")
	}

	// Check that full version contains all components
	fullVersion := info["full"]
	if fullVersion == "" {
		t.Error("Expected full version to not be empty")
	}

	// The full version should contain the version, branch, and commit
	// Format: "version (branch@commit)"
	if len(fullVersion) < 5 {
		t.Errorf("Expected full version to be longer, got '%s'", fullVersion)
	}
}

func TestGetFullVersion(t *testing.T) {
	fullVersion := GetFullVersion()

	if fullVersion == "" {
		t.Error("Expected full version to not be empty")
	}

	// The full version should contain the version, branch, and commit
	// Format: "version (branch@commit)"
	if len(fullVersion) < 5 {
		t.Errorf("Expected full version to be longer, got '%s'", fullVersion)
	}
}

func TestVersionConsistency(t *testing.T) {
	info := GetVersionInfo()
	fullVersion := GetFullVersion()

	// The full version from GetFullVersion should match the "full" key in GetVersionInfo
	if info["full"] != fullVersion {
		t.Errorf("Expected GetFullVersion() to return same value as GetVersionInfo()['full'], got '%s' vs '%s'", fullVersion, info["full"])
	}
}

func TestVersionFormat(t *testing.T) {
	info := GetVersionInfo()
	fullVersion := info["full"]

	// The full version should follow the format "version (branch@commit)"
	// We can't be too strict about the exact format since the values might be "unknown" in tests,
	// but we can check that it contains the expected structure
	if len(fullVersion) < 5 {
		t.Errorf("Expected full version to be longer, got '%s'", fullVersion)
	}

	// Check that it contains parentheses (indicating the branch@commit part)
	hasParentheses := false
	for _, char := range fullVersion {
		if char == '(' || char == ')' {
			hasParentheses = true
			break
		}
	}
	if !hasParentheses {
		t.Errorf("Expected full version to contain parentheses, got '%s'", fullVersion)
	}
}

func TestVersionVariables(t *testing.T) {
	// Test that the package variables are accessible and have reasonable values
	if Version == "" {
		t.Error("Expected Version variable to not be empty")
	}

	// GitCommit and GitBranch might be "unknown" in test environment, which is acceptable
	if GitCommit == "" {
		t.Error("Expected GitCommit variable to not be empty")
	}

	if GitBranch == "" {
		t.Error("Expected GitBranch variable to not be empty")
	}
}

func TestVersionInfoStructure(t *testing.T) {
	info := GetVersionInfo()

	// Test that the info map has exactly 4 keys
	expectedKeyCount := 4
	if len(info) != expectedKeyCount {
		t.Errorf("Expected version info to have exactly %d keys, got %d", expectedKeyCount, len(info))
	}

	// Test that all values are strings (GetVersionInfo returns map[string]string)
	for key, value := range info {
		if value == "" {
			t.Errorf("Expected value for key '%s' to not be empty", key)
		}
	}
}
