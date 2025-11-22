package version

import (
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	// Version should never be empty
	if Version == "" {
		t.Error("Version should not be empty")
	}
}

func TestGitCommit(t *testing.T) {
	// GitCommit should be set (at least to "unknown")
	if GitCommit == "" {
		t.Error("GitCommit should not be empty")
	}
}

func TestBuildDate(t *testing.T) {
	// BuildDate should be set (at least to "unknown")
	if BuildDate == "" {
		t.Error("BuildDate should not be empty")
	}
}

func TestGoVersion(t *testing.T) {
	// GoVersion should be set (at least to "unknown")
	if GoVersion == "" {
		t.Error("GoVersion should not be empty")
	}
}

func TestPlatform(t *testing.T) {
	// Platform should be set (at least to "unknown")
	if Platform == "" {
		t.Error("Platform should not be empty")
	}
}

func TestFullVersion(t *testing.T) {
	result := FullVersion()

	// FullVersion should not be empty
	if result == "" {
		t.Error("FullVersion() should not return empty string")
	}

	// Should contain version
	if !strings.Contains(result, Version) {
		t.Errorf("FullVersion() = %q, should contain Version %q", result, Version)
	}

	// Should contain GitCommit
	if !strings.Contains(result, GitCommit) {
		t.Errorf("FullVersion() = %q, should contain GitCommit %q", result, GitCommit)
	}

	// Should contain BuildDate
	if !strings.Contains(result, BuildDate) {
		t.Errorf("FullVersion() = %q, should contain BuildDate %q", result, BuildDate)
	}

	// Should contain GoVersion
	if !strings.Contains(result, GoVersion) {
		t.Errorf("FullVersion() = %q, should contain GoVersion %q", result, GoVersion)
	}

	// Should contain Platform
	if !strings.Contains(result, Platform) {
		t.Errorf("FullVersion() = %q, should contain Platform %q", result, Platform)
	}

	// Should contain expected keywords
	expectedKeywords := []string{"commit:", "built:", "go:", "platform:"}
	for _, keyword := range expectedKeywords {
		if !strings.Contains(result, keyword) {
			t.Errorf("FullVersion() = %q, should contain keyword %q", result, keyword)
		}
	}
}

func TestFullVersionFormat(t *testing.T) {
	// Test with custom values
	oldVersion := Version
	oldCommit := GitCommit
	oldBuildDate := BuildDate
	oldGoVersion := GoVersion
	oldPlatform := Platform

	defer func() {
		Version = oldVersion
		GitCommit = oldCommit
		BuildDate = oldBuildDate
		GoVersion = oldGoVersion
		Platform = oldPlatform
	}()

	Version = "1.2.3"
	GitCommit = "abc123"
	BuildDate = "2024-01-15"
	GoVersion = "go1.21.5"
	Platform = "darwin/arm64"

	result := FullVersion()
	expected := "1.2.3 (commit: abc123, built: 2024-01-15, go: go1.21.5, platform: darwin/arm64)"

	if result != expected {
		t.Errorf("FullVersion() = %q, want %q", result, expected)
	}
}

func TestVersionVariablesCanBeOverridden(t *testing.T) {
	// Test that build-time variables can be set
	const testVersion = "test-version"
	originalVersion := Version

	Version = testVersion
	if Version != testVersion {
		t.Error("Version should be settable")
	}

	// Restore
	Version = originalVersion
}
