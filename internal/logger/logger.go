package logger

import (
	"log/slog"
	"os"
)

// Init configures the global slog logger.
// When verbose is true the level is set to Debug; otherwise Info.
func Init(verbose bool) {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	slog.SetDefault(slog.New(handler))
}

// Debug logs a debug-level message (only visible with --verbose).
func Debug(msg string, args ...any) { slog.Debug(msg, args...) }

// Info logs an info-level message.
func Info(msg string, args ...any) { slog.Info(msg, args...) }

// Warn logs a warning-level message.
func Warn(msg string, args ...any) { slog.Warn(msg, args...) }

// Error logs an error-level message.
func Error(msg string, args ...any) { slog.Error(msg, args...) }
