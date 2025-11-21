package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aura/aura-proxy/pkg/version"
)

func TestAuraDirPath(t *testing.T) {
	// Test that auraDir is set correctly
	homeDir := os.Getenv("HOME")
	expectedPath := filepath.Join(homeDir, ".aura")

	if auraDir != expectedPath {
		t.Errorf("auraDir = %v, want %v", auraDir, expectedPath)
	}
}

func TestVersion(t *testing.T) {
	// Test that version is set
	if version.Version == "" {
		t.Error("version should not be empty")
	}

	// Test version format (should be semantic versioning)
	if len(version.Version) < 3 {
		t.Errorf("version = %v, should be in format X.Y.Z", version.Version)
	}
}

func TestCommandsExist(t *testing.T) {
	// Test that all expected commands are registered
	// Note: "completion" and "help" are added automatically by Cobra
	expectedCommands := []string{
		"install",
		"start",
		"stop",
		"status",
		"cert",
		"logs",
		"uninstall",
	}

	commands := rootCmd.Commands()
	commandMap := make(map[string]bool)

	for _, cmd := range commands {
		commandMap[cmd.Name()] = true
	}

	for _, expected := range expectedCommands {
		if !commandMap[expected] {
			t.Errorf("Expected command %q not found", expected)
		}
	}

	// Verify we have at least the expected number of commands
	if len(commands) < len(expectedCommands) {
		t.Errorf("Expected at least %d commands, got %d", len(expectedCommands), len(commands))
	}
}

func TestCertCommandDomainHandling(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "domain without .aura suffix",
			input:    "myapp",
			expected: "myapp.aura",
		},
		{
			name:     "domain with .aura suffix",
			input:    "myapp.aura",
			expected: "myapp.aura",
		},
		{
			name:     "subdomain without .aura",
			input:    "api.myapp",
			expected: "api.myapp.aura",
		},
		{
			name:     "subdomain with .aura",
			input:    "api.myapp.aura",
			expected: "api.myapp.aura",
		},
		{
			name:     "short domain",
			input:    "app",
			expected: "app.aura",
		},
		{
			name:     "single char",
			input:    "a",
			expected: "a.aura",
		},
		{
			name:     "empty",
			input:    "",
			expected: ".aura",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the domain handling logic from certCmd using HasSuffix
			domain := tt.input
			if !strings.HasSuffix(domain, auraTLD) {
				domain += auraTLD
			}

			if domain != tt.expected {
				t.Errorf("domain handling: got %v, want %v", domain, tt.expected)
			}
		})
	}
}

func TestCertificatePaths(t *testing.T) {
	// Test certificate path generation
	tests := []struct {
		domain       string
		expectedCert string
		expectedKey  string
	}{
		{
			domain:       "myapp.aura",
			expectedCert: "/certs/domains/myapp/cert.pem",
			expectedKey:  "/certs/domains/myapp/key.pem",
		},
		{
			domain:       "api.aura",
			expectedCert: "/certs/domains/api/cert.pem",
			expectedKey:  "/certs/domains/api/key.pem",
		},
		{
			domain:       "blog.example.aura",
			expectedCert: "/certs/domains/blog.example/cert.pem",
			expectedKey:  "/certs/domains/blog.example/key.pem",
		},
	}

	for _, tt := range tests {
		t.Run(tt.domain, func(t *testing.T) {
			// Extract domain name without .aura suffix
			domainName := tt.domain[:len(tt.domain)-5]

			const certsDomainsPath = "/certs/domains"
			certPath := filepath.Join(certsDomainsPath, domainName, "cert.pem")
			keyPath := filepath.Join(certsDomainsPath, domainName, "key.pem")

			if certPath != tt.expectedCert {
				t.Errorf("cert path = %v, want %v", certPath, tt.expectedCert)
			}

			if keyPath != tt.expectedKey {
				t.Errorf("key path = %v, want %v", keyPath, tt.expectedKey)
			}
		})
	}
}

func TestCopyConfigsCreatesDirectories(t *testing.T) {
	// This is more of an integration test, but we can test the logic
	// In a real scenario, we'd use a temp directory

	// Just verify the function exists and can be called
	// Full testing would require mocking the filesystem
	t.Skip("copyConfigs requires filesystem mocking for proper testing")
}

func TestCommandDescriptions(t *testing.T) {
	// Test that all commands have descriptions
	commands := rootCmd.Commands()

	for _, cmd := range commands {
		if cmd.Short == "" {
			t.Errorf("Command %q has no short description", cmd.Name())
		}
	}
}

func TestRootCommandLong(t *testing.T) {
	// Test that root command has a long description
	if rootCmd.Long == "" {
		t.Error("Root command should have a long description")
	}

	// Test that it includes the ASCII title
	if asciiTitle == "" {
		t.Error("ASCII title should not be empty")
	}
}

func TestColorCodes(t *testing.T) {
	// Test that color codes are properly defined
	if teal == "" {
		t.Error("teal color code should not be empty")
	}

	if reset == "" {
		t.Error("reset color code should not be empty")
	}

	// Test ANSI escape sequence format
	if teal[0] != '\033' {
		t.Error("teal should start with ANSI escape sequence")
	}

	if reset[0] != '\033' {
		t.Error("reset should start with ANSI escape sequence")
	}
}

func TestAsciiTitleContainsAura(t *testing.T) {
	// The ASCII art should represent "AURA"
	// At minimum, verify it's not empty and has reasonable length
	if len(asciiTitle) < 50 {
		t.Error("ASCII title seems too short to contain proper art")
	}

	// Should contain newlines (multi-line ASCII art)
	hasNewline := false
	for _, char := range asciiTitle {
		if char == '\n' {
			hasNewline = true
			break
		}
	}

	if !hasNewline {
		t.Error("ASCII title should contain newlines (multi-line art)")
	}
}

func TestValidateDomain(t *testing.T) {
	tests := []struct {
		name    string
		domain  string
		wantErr bool
	}{
		{"valid simple", "app.aura", false},
		{"valid subdomain", "api.app.aura", false},
		{"valid with hyphens", "my-app.aura", false},
		{"valid multi-level", "api.staging.app.aura", false},
		{"path traversal", "../../../etc/passwd.aura", true},
		{"command injection semicolon", "app;rm -rf /.aura", true},
		{"command injection ampersand", "app&whoami.aura", true},
		{"null byte", "app\x00.aura", true},
		{"too long", strings.Repeat("a", 250) + ".aura", true},
		{"empty", ".aura", true},
		{"double dot", "a..b.aura", true},
		{"label too long", strings.Repeat("a", 64) + ".aura", true},
		{"empty label", "a..aura", true},
		{"uppercase letters", "MyApp.aura", true},
		{"special chars", "app@test.aura", true},
		{"underscore", "my_app.aura", true},
		{"slash", "app/test.aura", true},
		{"backslash", "app\\test.aura", true},
		{"space", "my app.aura", true},
		{"single char", "a.aura", false},
		{"numbers", "app123.aura", false},
		{"hyphen start", "-app.aura", true},
		{"hyphen end", "app-.aura", true},
		{"valid long label", strings.Repeat("a", 63) + ".aura", false},
		{"valid max length", strings.Repeat("a", 59) + "." + strings.Repeat("b", 59) + "." + strings.Repeat("c", 59) + "." + strings.Repeat("d", 59) + ".aura", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDomain(tt.domain)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateDomain(%q) error = %v, wantErr %v", tt.domain, err, tt.wantErr)
			}
		})
	}
}

func TestValidateDomainEdgeCases(t *testing.T) {
	// Test edge cases for short strings
	tests := []string{"", "a", "ab", "abc", "abcd"}
	for _, input := range tests {
		domain := input
		if !strings.HasSuffix(domain, auraTLD) {
			domain += auraTLD
		}
		// Should not panic
		err := validateDomain(domain)
		if input == "" {
			if err == nil {
				t.Errorf("expected error for empty domain, got nil")
			}
		} else {
			if err != nil {
				t.Errorf("unexpected error for %q: %v", input, err)
			}
		}
	}
}
