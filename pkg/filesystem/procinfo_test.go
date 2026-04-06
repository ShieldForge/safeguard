package filesystem

import (
	"strings"
	"testing"
)

func TestResolveProcessDetails_Nil(t *testing.T) {
	result := resolveProcessDetails(nil)
	if result != nil {
		t.Error("resolveProcessDetails(nil) should return nil")
	}
}

func TestResolveProcessDetails_Basic(t *testing.T) {
	info := &ProcessInfo{
		PID: 1234,
		UID: 1000,
		GID: 100,
	}

	details := resolveProcessDetails(info)
	if details == nil {
		t.Fatal("resolveProcessDetails() returned nil")
	}
	if details.PID != 1234 {
		t.Errorf("PID = %d, want 1234", details.PID)
	}
	if details.UID != 1000 {
		t.Errorf("UID = %d, want 1000", details.UID)
	}
	if details.GID != 100 {
		t.Errorf("GID = %d, want 100", details.GID)
	}
}

func TestGetUsernameFromProcess_Nil(t *testing.T) {
	result := getUsernameFromProcess(nil)
	if result != "" {
		t.Errorf("getUsernameFromProcess(nil) = %v, want empty", result)
	}
}

func TestFormatProcessInfo_Nil(t *testing.T) {
	result := formatProcessInfo(nil)
	if result != "[NO CONTEXT]" {
		t.Errorf("formatProcessInfo(nil) = %v, want [NO CONTEXT]", result)
	}
}

func TestFormatProcessInfo_BasicPID(t *testing.T) {
	details := &ProcessDetails{
		PID: 42,
		UID: 0,
		GID: 0,
	}
	result := formatProcessInfo(details)
	if !strings.Contains(result, "PID: 42") {
		t.Errorf("Expected PID: 42 in result: %s", result)
	}
	if !strings.Contains(result, "UID: 0") {
		t.Errorf("Expected UID: 0 in result: %s", result)
	}
	if !strings.Contains(result, "GID: 0") {
		t.Errorf("Expected GID: 0 in result: %s", result)
	}
}

func TestFormatProcessInfo_WithProcessName(t *testing.T) {
	details := &ProcessDetails{
		PID:         1234,
		ProcessName: "notepad.exe",
		UID:         1000,
		GID:         100,
	}
	result := formatProcessInfo(details)
	if !strings.Contains(result, "(notepad.exe)") {
		t.Errorf("Expected (notepad.exe) in result: %s", result)
	}
}

func TestFormatProcessInfo_WithExecutablePath(t *testing.T) {
	details := &ProcessDetails{
		PID:            1234,
		ProcessName:    "notepad.exe",
		ExecutablePath: `C:\Windows\notepad.exe`,
		UID:            1000,
		GID:            100,
	}
	result := formatProcessInfo(details)
	if !strings.Contains(result, `C:\Windows\notepad.exe`) {
		t.Errorf("Expected executable path in result: %s", result)
	}
}

func TestFormatProcessInfo_PathSameAsName(t *testing.T) {
	// When path equals name, path should not appear twice
	details := &ProcessDetails{
		PID:            1234,
		ProcessName:    "notepad.exe",
		ExecutablePath: "notepad.exe",
		UID:            1000,
		GID:            100,
	}
	result := formatProcessInfo(details)
	// Should contain name but not duplicate it with " - notepad.exe"
	count := strings.Count(result, "notepad.exe")
	if count != 1 {
		t.Errorf("Expected notepad.exe to appear once, appeared %d times in: %s", count, result)
	}
}

func TestFormatProcessInfo_WithUsername(t *testing.T) {
	details := &ProcessDetails{
		PID:      1234,
		UID:      1000,
		Username: "john",
		GID:      100,
	}
	result := formatProcessInfo(details)
	if !strings.Contains(result, "(john)") {
		t.Errorf("Expected (john) in result: %s", result)
	}
}

func TestFormatProcessInfo_FullDetails(t *testing.T) {
	details := &ProcessDetails{
		PID:            5678,
		ProcessName:    "app.exe",
		ExecutablePath: `C:\Program Files\app.exe`,
		UID:            500,
		Username:       "admin",
		GID:            50,
	}
	result := formatProcessInfo(details)

	expected := []string{
		"PID: 5678",
		"(app.exe)",
		`C:\Program Files\app.exe`,
		"UID: 500",
		"(admin)",
		"GID: 50",
	}
	for _, want := range expected {
		if !strings.Contains(result, want) {
			t.Errorf("Expected %q in result: %s", want, result)
		}
	}

	// Should be wrapped in brackets
	if !strings.HasPrefix(result, "[") || !strings.HasSuffix(result, "]") {
		t.Errorf("Result should be wrapped in brackets: %s", result)
	}
}
