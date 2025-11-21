package logger

import (
	"io"
	"log/slog"
	"os"
)

var (
	// Log is the global logger instance
	Log *slog.Logger
)

// Level represents log level
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

// Config for logger initialization
type Config struct {
	Level  Level
	Format string // "text" or "json"
	Output io.Writer
}

// Init initializes the global logger
func Init(cfg Config) {
	var level slog.Level
	switch cfg.Level {
	case LevelDebug:
		level = slog.LevelDebug
	case LevelInfo:
		level = slog.LevelInfo
	case LevelWarn:
		level = slog.LevelWarn
	case LevelError:
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(cfg.Output, opts)
	} else {
		handler = slog.NewTextHandler(cfg.Output, opts)
	}

	Log = slog.New(handler)
}

// Initialize with defaults if not already initialized
func init() {
	if Log == nil {
		Init(Config{
			Level:  LevelInfo,
			Format: "text",
			Output: os.Stdout,
		})
	}
}

// Helper functions for common patterns
func Info(msg string, args ...any) {
	Log.Info(msg, args...)
}

func Debug(msg string, args ...any) {
	Log.Debug(msg, args...)
}

func Warn(msg string, args ...any) {
	Log.Warn(msg, args...)
}

func Error(msg string, args ...any) {
	Log.Error(msg, args...)
}

func Fatal(msg string, args ...any) {
	Log.Error(msg, args...)
	os.Exit(1)
}
