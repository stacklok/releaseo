// Copyright 2025 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package logging provides a simple logging abstraction for the releaseo action.
package logging

import (
	"log/slog"
	"os"
)

// Logger wraps slog.Logger to provide convenience methods for the action.
type Logger struct {
	*slog.Logger
}

// New creates a new Logger with text output to stdout.
// The log level can be set via the LOG_LEVEL environment variable.
func New() *Logger {
	level := slog.LevelInfo

	// Allow log level to be configured via environment variable
	if levelStr := os.Getenv("LOG_LEVEL"); levelStr != "" {
		switch levelStr {
		case "debug", "DEBUG":
			level = slog.LevelDebug
		case "info", "INFO":
			level = slog.LevelInfo
		case "warn", "WARN", "warning", "WARNING":
			level = slog.LevelWarn
		case "error", "ERROR":
			level = slog.LevelError
		}
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	handler := slog.NewTextHandler(os.Stdout, opts)
	return &Logger{slog.New(handler)}
}

// NewWithLevel creates a new Logger with the specified log level.
func NewWithLevel(level slog.Level) *Logger {
	opts := &slog.HandlerOptions{
		Level: level,
	}
	handler := slog.NewTextHandler(os.Stdout, opts)
	return &Logger{slog.New(handler)}
}

// Infof logs an info message with printf-style formatting.
func (l *Logger) Infof(format string, args ...any) {
	//nolint:sloglint // using printf-style is intentional for migration simplicity
	l.Info(format, args...)
}

// Warnf logs a warning message with printf-style formatting.
func (l *Logger) Warnf(format string, args ...any) {
	//nolint:sloglint // using printf-style is intentional for migration simplicity
	l.Warn(format, args...)
}

// Errorf logs an error message with printf-style formatting.
func (l *Logger) Errorf(format string, args ...any) {
	//nolint:sloglint // using printf-style is intentional for migration simplicity
	l.Error(format, args...)
}

// Default returns the default logger instance for package-level logging.
var defaultLogger = New()

// Default returns the default logger instance.
func Default() *Logger {
	return defaultLogger
}
