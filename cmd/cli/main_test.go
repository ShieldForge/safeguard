package main

import (
	"runtime"
	"testing"
)

func TestGetDefaultMountPoint(t *testing.T) {
	tests := []struct {
		name string
		goos string
		want string
	}{
		{
			name: "windows",
			goos: "windows",
			want: "V:",
		},
		{
			name: "darwin",
			goos: "darwin",
			want: "/tmp/vault",
		},
		{
			name: "linux",
			goos: "linux",
			want: "/mnt/vault",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: This test will only pass for the current OS
			// Full testing would require build tags or mocking
			if runtime.GOOS == tt.goos {
				got := getDefaultMountPoint()
				if got != tt.want {
					t.Errorf("getDefaultMountPoint() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestGetDefaultMountPoint_CurrentOS(t *testing.T) {
	got := getDefaultMountPoint()

	if got == "" {
		t.Error("getDefaultMountPoint() returned empty string")
	}

	// Verify it returns something sensible for the current OS
	switch runtime.GOOS {
	case "windows":
		if got != "V:" {
			t.Errorf("On Windows, expected V:, got %v", got)
		}
	case "darwin":
		if got != "/tmp/vault" {
			t.Errorf("On macOS, expected /tmp/vault, got %v", got)
		}
	case "linux":
		if got != "/mnt/vault" {
			t.Errorf("On Linux, expected /mnt/vault, got %v", got)
		}
	default:
		// For other OS, just check it's not empty
		if got != "/tmp/vault" {
			t.Errorf("On unknown OS, expected /tmp/vault, got %v", got)
		}
	}
}
