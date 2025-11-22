package logger

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestInit(t *testing.T) {
	//nolint:govet // fieldalignment: test struct clarity over memory optimization
	tests := []struct {
		name   string
		config Config
	}{
		{
			name: "text format, info level",
			config: Config{
				Level:  LevelInfo,
				Format: "text",
				Output: &bytes.Buffer{},
			},
		},
		{
			name: "json format, debug level",
			config: Config{
				Level:  LevelDebug,
				Format: "json",
				Output: &bytes.Buffer{},
			},
		},
		{
			name: "text format, error level",
			config: Config{
				Level:  LevelError,
				Format: "text",
				Output: &bytes.Buffer{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Init(tt.config)
			if Log == nil {
				t.Error("Log should not be nil after Init()")
			}
		})
	}
}

func TestInfo(t *testing.T) {
	var buf bytes.Buffer
	Init(Config{
		Level:  LevelInfo,
		Format: "text",
		Output: &buf,
	})

	Info("test message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("output should contain 'test message', got: %s", output)
	}
	if !strings.Contains(output, "key=value") {
		t.Errorf("output should contain 'key=value', got: %s", output)
	}
}

func TestDebug(t *testing.T) {
	var buf bytes.Buffer
	Init(Config{
		Level:  LevelDebug,
		Format: "text",
		Output: &buf,
	})

	Debug("debug message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, "debug message") {
		t.Errorf("output should contain 'debug message', got: %s", output)
	}
}

func TestDebugLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	Init(Config{
		Level:  LevelInfo, // Info level should not show debug messages
		Format: "text",
		Output: &buf,
	})

	Debug("debug message")

	output := buf.String()
	if output != "" {
		t.Errorf("debug messages should not appear at info level, got: %s", output)
	}
}

func TestWarn(t *testing.T) {
	var buf bytes.Buffer
	Init(Config{
		Level:  LevelWarn,
		Format: "text",
		Output: &buf,
	})

	Warn("warning message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, "warning message") {
		t.Errorf("output should contain 'warning message', got: %s", output)
	}
}

func TestError(t *testing.T) {
	var buf bytes.Buffer
	Init(Config{
		Level:  LevelError,
		Format: "text",
		Output: &buf,
	})

	Error("error message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, "error message") {
		t.Errorf("output should contain 'error message', got: %s", output)
	}
}

func TestJSONFormat(t *testing.T) {
	var buf bytes.Buffer
	Init(Config{
		Level:  LevelInfo,
		Format: "json",
		Output: &buf,
	})

	Info("test json", "key", "value")

	output := buf.String()

	// Parse JSON to verify it's valid
	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Errorf("output should be valid JSON, got error: %v, output: %s", err, output)
	}

	// Check expected fields
	if msg, ok := logEntry["msg"].(string); !ok || msg != "test json" {
		t.Errorf("JSON should contain msg='test json', got: %v", logEntry["msg"])
	}

	if key, ok := logEntry["key"].(string); !ok || key != "value" {
		t.Errorf("JSON should contain key='value', got: %v", logEntry["key"])
	}
}

func TestLevelConversion(t *testing.T) {
	tests := []struct {
		name  string
		level Level
	}{
		{"debug level", LevelDebug},
		{"info level", LevelInfo},
		{"warn level", LevelWarn},
		{"error level", LevelError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			Init(Config{
				Level:  tt.level,
				Format: "text",
				Output: &buf,
			})

			if Log == nil {
				t.Error("Log should not be nil")
			}
		})
	}
}

func TestDefaultInit(t *testing.T) {
	// Test that init() sets up a default logger
	if Log == nil {
		t.Error("Log should be initialized by default")
	}
}

func TestMultipleFields(t *testing.T) {
	var buf bytes.Buffer
	Init(Config{
		Level:  LevelInfo,
		Format: "json",
		Output: &buf,
	})

	Info("multi-field message",
		"field1", "value1",
		"field2", 42,
		"field3", true,
	)

	output := buf.String()

	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Fatalf("output should be valid JSON: %v", err)
	}

	if logEntry["field1"] != "value1" {
		t.Errorf("field1 should be 'value1', got: %v", logEntry["field1"])
	}

	if logEntry["field2"] != float64(42) { // JSON numbers are float64
		t.Errorf("field2 should be 42, got: %v", logEntry["field2"])
	}

	if logEntry["field3"] != true {
		t.Errorf("field3 should be true, got: %v", logEntry["field3"])
	}
}
