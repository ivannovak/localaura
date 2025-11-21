//go:build integration
// +build integration

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallCommand(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	oldAuraDir := auraDir
	auraDir = filepath.Join(tmpDir, ".aura")
	defer func() { auraDir = oldAuraDir }()

	// Run install command (excluding actual script execution)
	// First, just test directory creation and file copying
	if err := os.MkdirAll(auraDir, 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	// Test copyConfigs
	if err := copyConfigs(); err != nil {
		t.Fatalf("copyConfigs failed: %v", err)
	}

	// Verify directory created
	if _, err := os.Stat(auraDir); os.IsNotExist(err) {
		t.Error("aura directory not created")
	}

	// Verify files copied
	expectedFiles := []string{
		"docker-compose.yml",
		"docker-compose.example.yml",
		"setup.sh",
		"setup-loopback.sh",
		"setup-resolver.sh",
		"setup-mkcert.sh",
		"add-cert.sh",
		"uninstall-resolver.sh",
		"uninstall-loopback.sh",
	}

	for _, file := range expectedFiles {
		path := filepath.Join(auraDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file not found: %s", file)
		}

		// Verify files are not empty
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("failed to stat file %s: %v", file, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("file %s is empty", file)
		}
	}

	// Verify Corefile
	corefilePath := filepath.Join(auraDir, "coredns", "Corefile")
	if _, err := os.Stat(corefilePath); os.IsNotExist(err) {
		t.Error("Corefile not found")
	}
}

func TestCopyConfigs(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	oldAuraDir := auraDir
	auraDir = tmpDir
	defer func() { auraDir = oldAuraDir }()

	// Run copyConfigs
	err := copyConfigs()
	if err != nil {
		t.Fatalf("copyConfigs failed: %v", err)
	}

	// Verify structure
	dirs := []string{
		filepath.Join(auraDir, "certs", "domains"),
		filepath.Join(auraDir, "coredns"),
	}

	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("expected directory not created: %s", dir)
		}

		// Verify permissions
		info, err := os.Stat(dir)
		if err != nil {
			t.Errorf("failed to stat directory %s: %v", dir, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%s is not a directory", dir)
		}
	}

	// Verify file permissions (should be 0600 for security)
	testFile := filepath.Join(auraDir, "setup.sh")
	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("failed to stat test file: %v", err)
	}
	mode := info.Mode().Perm()
	expected := os.FileMode(0600)
	if mode != expected {
		t.Errorf("file permission = %#o, want %#o", mode, expected)
	}
}

func TestCopyConfigsReadsEmbeddedFiles(t *testing.T) {
	// Verify embeddedFS contains expected files
	expectedEmbedFiles := []string{
		"embed/docker-compose.yml",
		"embed/docker-compose.example.yml",
		"embed/setup.sh",
		"embed/setup-loopback.sh",
		"embed/setup-resolver.sh",
		"embed/setup-mkcert.sh",
		"embed/add-cert.sh",
		"embed/uninstall-resolver.sh",
		"embed/uninstall-loopback.sh",
		"embed/coredns/Corefile",
	}

	for _, file := range expectedEmbedFiles {
		data, err := embeddedFS.ReadFile(file)
		if err != nil {
			t.Errorf("failed to read embedded file %s: %v", file, err)
			continue
		}
		if len(data) == 0 {
			t.Errorf("embedded file %s is empty", file)
		}
	}
}

func TestValidateDomainErrorMessages(t *testing.T) {
	tests := []struct {
		name        string
		domain      string
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "path traversal",
			domain:      "../../../etc/passwd.aura",
			shouldError: true,
			errorMsg:    "path traversal",
		},
		{
			name:        "empty domain",
			domain:      ".aura",
			shouldError: true,
			errorMsg:    "empty",
		},
		{
			name:        "too long",
			domain:      string(make([]byte, 250)) + ".aura",
			shouldError: true,
			errorMsg:    "too long",
		},
		{
			name:        "invalid characters semicolon",
			domain:      "app;rm -rf /.aura",
			shouldError: true,
			errorMsg:    "invalid",
		},
		{
			name:        "invalid characters ampersand",
			domain:      "app&whoami.aura",
			shouldError: true,
			errorMsg:    "invalid",
		},
		{
			name:        "null byte",
			domain:      "app\x00.aura",
			shouldError: true,
			errorMsg:    "null byte",
		},
		{
			name:        "valid simple",
			domain:      "app.aura",
			shouldError: false,
		},
		{
			name:        "valid subdomain",
			domain:      "api.app.aura",
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDomain(tt.domain)

			if tt.shouldError {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errorMsg)
				} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

func TestRunCommandErrors(t *testing.T) {
	tests := []struct {
		name    string
		command string
		args    []string
		wantErr bool
	}{
		{
			name:    "nonexistent command",
			command: "this-command-does-not-exist-12345",
			args:    []string{},
			wantErr: true,
		},
		{
			name:    "command with error exit",
			command: "sh",
			args:    []string{"-c", "exit 1"},
			wantErr: true,
		},
		{
			name:    "valid command",
			command: "echo",
			args:    []string{"test"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runCommand(tt.command, tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("runCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRunCommandInDirErrors(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		dir     string
		command string
		args    []string
		wantErr bool
	}{
		{
			name:    "nonexistent directory",
			dir:     "/this/path/does/not/exist/12345",
			command: "echo",
			args:    []string{"test"},
			wantErr: true,
		},
		{
			name:    "valid directory and command",
			dir:     tmpDir,
			command: "pwd",
			args:    []string{},
			wantErr: false,
		},
		{
			name:    "valid directory, failing command",
			dir:     tmpDir,
			command: "sh",
			args:    []string{"-c", "exit 42"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runCommandInDir(tt.dir, tt.command, tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("runCommandInDir() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
