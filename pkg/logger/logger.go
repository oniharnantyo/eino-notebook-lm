package logger

import (
	"io"
	"log/slog"
	"os"
)

type LogLevel string

const (
	LevelDebug LogLevel = "debug"
	LevelInfo  LogLevel = "info"
	LevelWarn  LogLevel = "warn"
	LevelError LogLevel = "error"
)

// Logger wraps slog.Logger
type Logger struct {
	*slog.Logger
}

// New creates a new logger
func New(level LogLevel, format string) *Logger {
	var logLevel slog.Level
	switch level {
	case LevelDebug:
		logLevel = slog.LevelDebug
	case LevelInfo:
		logLevel = slog.LevelInfo
	case LevelWarn:
		logLevel = slog.LevelWarn
	case LevelError:
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	var handler slog.Handler
	if format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return &Logger{
		Logger: slog.New(handler),
	}
}

// NewWithWriter creates a new logger with a custom writer
func NewWithWriter(w io.Writer, level LogLevel, format string) *Logger {
	var logLevel slog.Level
	switch level {
	case LevelDebug:
		logLevel = slog.LevelDebug
	case LevelInfo:
		logLevel = slog.LevelInfo
	case LevelWarn:
		logLevel = slog.LevelWarn
	case LevelError:
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	var handler slog.Handler
	if format == "json" {
		handler = slog.NewJSONHandler(w, opts)
	} else {
		handler = slog.NewTextHandler(w, opts)
	}

	return &Logger{
		Logger: slog.New(handler),
	}
}

// With returns a logger with additional fields
func (l *Logger) With(args ...any) *Logger {
	return &Logger{
		Logger: l.Logger.With(args...),
	}
}

// WithGroup returns a logger with a group
func (l *Logger) WithGroup(name string) *Logger {
	return &Logger{
		Logger: l.Logger.WithGroup(name),
	}
}

// Default logger instance
var std = New(LevelInfo, "json")

// Default functions using the standard logger
func Debug(msg string, args ...any) {
	std.Debug(msg, args...)
}

func Info(msg string, args ...any) {
	std.Info(msg, args...)
}

func Warn(msg string, args ...any) {
	std.Warn(msg, args...)
}

func Error(msg string, args ...any) {
	std.Error(msg, args...)
}

// SetDefault sets the default logger
func SetDefault(l *Logger) {
	std = l
}
