// Package logger provides a structured logging interface wrapping zerolog.
//
// This package offers a simplified logging API with support for structured
// fields, multiple log levels, and integration with external systems like Splunk.
// It wraps the zerolog library to provide JSON-formatted logs with timestamps.
//
// Example usage:
//
//	log := logger.New(os.Stdout, true) // debug mode enabled
//	log.Info("Server started", map[string]interface{}{
//	    "port": 8080,
//	    "host": "0.0.0.0",
//	})
//
//	log.Error("Failed to connect", map[string]interface{}{
//	    "error": err.Error(),
//	    "retry_count": 3,
//	})
//
// The package also provides global logging functions that use a default logger:
//
//	logger.Info("Application initialized", nil)
//	logger.Debug("Cache hit", map[string]interface{}{"key": "user:123"})
package logger

import (
	"io"
	"os"

	"github.com/rs/zerolog"
)

// Logger wraps zerolog.Logger to provide a simplified logging interface.
//
// Logger supports structured logging with arbitrary fields and automatic
// timestamp injection. It provides five log levels: Debug, Info, Warn,
// Error, and Fatal.
//
// Log entries are output in JSON format for easy parsing by log aggregation
// systems. The debug level can be controlled at logger creation time.
type Logger struct {
	zlog zerolog.Logger
}

// New creates a new zerolog-based logger with the specified output writer and debug mode.
//
// Parameters:
//   - writer: The io.Writer where log entries are written (e.g., os.Stdout, a file, or SplunkWriter)
//   - debug: If true, enables debug-level logging; if false, only info-level and above are logged
//
// The logger automatically adds timestamps to all log entries and formats them as JSON.
//
// If writer is nil, os.Stdout is used as the default output destination.
//
// Example:
//
//	// Log to stdout with debug enabled
//	log := logger.New(os.Stdout, true)
//
//	// Log to file without debug
//	file, _ := os.OpenFile("app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
//	log := logger.New(file, false)
//
//	// Log to multiple destinations
//	multi := zerolog.MultiLevelWriter(os.Stdout, file)
//	log := logger.New(multi, true)
func New(writer io.Writer, debug bool) *Logger {
	if writer == nil {
		writer = os.Stdout
	}

	// Set log level based on debug flag
	level := zerolog.WarnLevel
	if debug {
		level = zerolog.DebugLevel
	}

	zlog := zerolog.New(writer).
		Level(level).
		With().
		Timestamp().
		Logger()

	return &Logger{
		zlog: zlog,
	}
}

// NewWithLevel creates a new zerolog-based logger with an explicit log level.
//
// Use this when the caller needs direct control over the minimum log level
// (e.g. services that should always log at Info level).
func NewWithLevel(writer io.Writer, level zerolog.Level) *Logger {
	if writer == nil {
		writer = os.Stdout
	}

	zlog := zerolog.New(writer).
		Level(level).
		With().
		Timestamp().
		Logger()

	return &Logger{
		zlog: zlog,
	}
}

// defaultLogger is the global logger instance used by package-level logging functions.
//
// It logs to stdout at info level by default. Use SetDefault to replace it with a
// custom logger configuration.
var defaultLogger = New(os.Stdout, false)

// SetDefault sets the global default logger used by package-level logging functions.
//
// addFields is a helper function that adds structured fields to a zerolog event.
//
// This internal function iterates over the fields map and adds each key-value pair
// to the zerolog event using the Interface method, which automatically handles
// type serialization.
//
// If fields is nil, the event is returned unchanged. at application startup that will be
// used by all package-level logging calls (Debug, Info, Warn, Error, Fatal).
//
// Example:
//
//	log := logger.New(-level message with optional structured fields.
//
// Debug messages are typically used for detailed diagnostic information useful
// during development and troubleshooting. They are only output if the logger
// was created with debug mode enabled.
//
// Parameters:
//   - message: The log message
//   - fields: Optional key-value pairs for structured data (can be nil)
//
// Example:
//
//	log.Debug("Cache lookup", map[string]interface{}{
//	    "key": "user:123",
//	    "cache_hit": true,
//	    "latency_ms": 2,
//	}) debugMode)
//	logger.SetDefault(log)
//
//	// Now all packagrmational message with optional structured fields.
//
// Info messages are used for general operational information about application
// execution, such as startup, shutdown, major state changes, or significant events.
//
// Parameters: with optional structured fields.
//
// Warn messages indicate potentially problematic situations that don't prevent
// the application from functioning but may require attention. Examples include
// deprecated API usage, non-optimal conditions, or recoverable errors.
// with optional structured fields.
//
// Error messages indicate serious problems that prevented a specific operation
// from completing successfully but don't necessarily prevent the application
// from continuing. Always include error details in the fields map.
//
// Parameters:
//   - message: The log message
//   - fields: Optional key-value pairs for structured data (can be nil)
//
// Example:
//error message and terminates the application.
//
// Fatal messages indicate critical errors that prevent the application from
// continuing. After logging the message, this function calls os.Exit(1).
//
// Package-level logging functions that use the default logger.
// These are convenient for simple logging without managing a logger instance.

// Debug logs a debug-level message using the default logger.
// See Logger.Debug for details.itialization or when the
// application state is corrupted beyond repair.
//
// Parameters:
//   - message: The log message
//   - fields: Optional key-value pairs for structured data (can be nil)
//
// Example:
//
//	log.Fatal("Configuration file not found", map[string]interface{}{
//	    "path": "/etc/app/config.json",
//	    "error": err.Error(),
//	})
//	// Application terminates here
//	log.Error("Failed to connect to database", map[string]interface{}{
//	    "error": err.Error(),
//	    "host": "db.example.com",
//
// Info logs an informational message using the default logger.
// See Logger.Info for details.
//
//	    "retry_count": 3,
//	})
//
// Warn logs a warning message using the default logger.
// See Logger.Warn for details.
// Parameters:
//   - message: The log message
//
// Error logs an error message using the default logger.
// See Logger.Error for details.
//   - fields: Optional key-value pairs for structured data (can be nil)
//
// Fatal logs a fatal error message and terminates the application using the default logger.
// See Logger.Fatal for details.
// Example:
//
//		log.Warn("High memory usage", map[string]interface{}{
//		    "memory_mb": 1024,
//		    "threshold_mb": 800,
//		})
//	  - message: The log message
//	  - fields: Optional key-value pairs for structured data (can be nil)
//
// Example:
//
//	log.Info("Server started", map[string]interface{}{
//	    "port": 8080,
//	    "environment": "production",
//	})calls use this logger
//	logger.Info("Application started", nil)
func SetDefault(l *Logger) {
	defaultLogger = l
}

// Helper function to add fields to zerolog event
func addFields(event *zerolog.Event, fields map[string]interface{}) *zerolog.Event {
	if fields == nil {
		return event
	}
	for key, value := range fields {
		event = event.Interface(key, value)
	}
	return event
}

// Debug logs a debug message
func (l *Logger) Debug(message string, fields map[string]interface{}) {
	addFields(l.zlog.Debug(), fields).Msg(message)
}

// Info logs an info message
func (l *Logger) Info(message string, fields map[string]interface{}) {
	addFields(l.zlog.Info(), fields).Msg(message)
}

// Warn logs a warning message
func (l *Logger) Warn(message string, fields map[string]interface{}) {
	addFields(l.zlog.Warn(), fields).Msg(message)
}

// Error logs an error message
func (l *Logger) Error(message string, fields map[string]interface{}) {
	addFields(l.zlog.Error(), fields).Msg(message)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(message string, fields map[string]interface{}) {
	addFields(l.zlog.Fatal(), fields).Msg(message)
}

// Global logging functions using the default logger
func Debug(message string, fields map[string]interface{}) {
	defaultLogger.Debug(message, fields)
}

func Info(message string, fields map[string]interface{}) {
	defaultLogger.Info(message, fields)
}

func Warn(message string, fields map[string]interface{}) {
	defaultLogger.Warn(message, fields)
}

func Error(message string, fields map[string]interface{}) {
	defaultLogger.Error(message, fields)
}

func Fatal(message string, fields map[string]interface{}) {
	defaultLogger.Fatal(message, fields)
}
