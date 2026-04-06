package logger

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func TestNew(t *testing.T) {
	t.Run("debug mode enabled", func(t *testing.T) {
		var buf bytes.Buffer
		log := New(&buf, true)
		if log == nil {
			t.Fatal("New() returned nil")
		}
		log.Debug("hello", nil)
		if !strings.Contains(buf.String(), "hello") {
			t.Error("Debug message not written when debug=true")
		}
	})

	t.Run("debug mode disabled", func(t *testing.T) {
		var buf bytes.Buffer
		log := New(&buf, false)
		log.Debug("hidden", nil)
		if buf.Len() != 0 {
			t.Errorf("Debug message should be suppressed when debug=false, got: %s", buf.String())
		}
	})

	t.Run("nil writer defaults to stdout", func(t *testing.T) {
		log := New(nil, false)
		if log == nil {
			t.Fatal("New(nil, false) returned nil")
		}
	})
}

func TestNewWithLevel(t *testing.T) {
	var buf bytes.Buffer
	log := NewWithLevel(&buf, zerolog.InfoLevel)
	if log == nil {
		t.Fatal("NewWithLevel() returned nil")
	}

	log.Info("visible", nil)
	if !strings.Contains(buf.String(), "visible") {
		t.Error("Info message not written at InfoLevel")
	}

	buf.Reset()
	log.Debug("hidden", nil)
	if buf.Len() != 0 {
		t.Error("Debug message should be suppressed at InfoLevel")
	}
}

func TestNewWithLevel_NilWriter(t *testing.T) {
	log := NewWithLevel(nil, zerolog.InfoLevel)
	if log == nil {
		t.Fatal("NewWithLevel(nil, ...) returned nil")
	}
}

func TestLoggerLevels(t *testing.T) {
	tests := []struct {
		name  string
		logFn func(*Logger, string, map[string]interface{})
		level string
	}{
		{"Debug", (*Logger).Debug, "debug"},
		{"Info", (*Logger).Info, "info"},
		{"Warn", (*Logger).Warn, "warn"},
		{"Error", (*Logger).Error, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			log := New(&buf, true) // debug=true so all levels emit
			tt.logFn(log, "test message", nil)

			var entry map[string]interface{}
			if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
				t.Fatalf("Failed to parse log entry: %v", err)
			}
			if entry["level"] != tt.level {
				t.Errorf("level = %v, want %v", entry["level"], tt.level)
			}
			if entry["message"] != "test message" {
				t.Errorf("message = %v, want 'test message'", entry["message"])
			}
		})
	}
}

func TestLoggerFields(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, true)
	log.Info("with fields", map[string]interface{}{
		"key1": "value1",
		"key2": 42,
	})

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to parse log entry: %v", err)
	}
	if entry["key1"] != "value1" {
		t.Errorf("key1 = %v, want value1", entry["key1"])
	}
	if entry["key2"] != float64(42) {
		t.Errorf("key2 = %v, want 42", entry["key2"])
	}
}

func TestLoggerNilFields(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, true)
	log.Info("no fields", nil)

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to parse log entry: %v", err)
	}
	if entry["message"] != "no fields" {
		t.Errorf("message = %v, want 'no fields'", entry["message"])
	}
}

func TestLoggerTimestamp(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, true)
	log.Info("timestamped", nil)

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to parse log entry: %v", err)
	}
	if _, ok := entry["time"]; !ok {
		t.Error("Log entry missing timestamp")
	}
}

func TestSetDefault(t *testing.T) {
	var buf bytes.Buffer
	log := NewWithLevel(&buf, zerolog.InfoLevel)
	SetDefault(log)

	Info("package-level info", nil)
	if !strings.Contains(buf.String(), "package-level info") {
		t.Error("Package-level Info() did not use custom default logger")
	}

	// Restore default to avoid affecting other tests
	SetDefault(New(nil, false))
}

func TestPackageLevelFunctions(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, true)
	SetDefault(log)
	defer SetDefault(New(nil, false))

	tests := []struct {
		name  string
		logFn func(string, map[string]interface{})
	}{
		{"Debug", Debug},
		{"Info", Info},
		{"Warn", Warn},
		{"Error", Error},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.logFn("pkg-"+tt.name, nil)
			if !strings.Contains(buf.String(), "pkg-"+tt.name) {
				t.Errorf("Package-level %s() did not produce output", tt.name)
			}
		})
	}
}

func TestAddFields(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, true)

	// nil fields should not panic
	log.Info("safe", nil)
	if !strings.Contains(buf.String(), "safe") {
		t.Error("Log with nil fields failed")
	}

	// empty map should work
	buf.Reset()
	log.Info("empty", map[string]interface{}{})
	if !strings.Contains(buf.String(), "empty") {
		t.Error("Log with empty fields failed")
	}
}
