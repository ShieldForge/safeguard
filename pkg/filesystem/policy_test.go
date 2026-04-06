package filesystem

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPolicyEvaluator(t *testing.T) {
	// Create a temporary policy file
	tmpDir := t.TempDir()
	policyPath := filepath.Join(tmpDir, "test-policy.rego")

	// Write a simple test policy
	policyContent := `
package vault

default result = {"allow": false, "reason": ""}

# Allow PowerShell
result = {"allow": true, "reason": "PowerShell access"} {
	input.process_name == "powershell.exe"
}

# Allow admin users
result = {"allow": true, "reason": "Admin access"} {
	input.username == "admin"
}
`

	if err := os.WriteFile(policyPath, []byte(policyContent), 0644); err != nil {
		t.Fatalf("Failed to write test policy: %v", err)
	}

	// Create policy evaluator
	evaluator, err := NewPolicyEvaluator(policyPath)
	if err != nil {
		t.Fatalf("Failed to create policy evaluator: %v", err)
	}

	tests := []struct {
		name     string
		request  *AccessRequest
		expected bool
	}{
		{
			name: "Allow PowerShell",
			request: &AccessRequest{
				Path:        "secret/test",
				Operation:   "READ",
				ProcessName: "powershell.exe",
				Username:    "user",
			},
			expected: true,
		},
		{
			name: "Allow admin user",
			request: &AccessRequest{
				Path:        "secret/test",
				Operation:   "READ",
				ProcessName: "cmd.exe",
				Username:    "admin",
			},
			expected: true,
		},
		{
			name: "Deny other process",
			request: &AccessRequest{
				Path:        "secret/test",
				Operation:   "READ",
				ProcessName: "notepad.exe",
				Username:    "user",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, _, err := evaluator.Evaluate(context.Background(), tt.request)
			if err != nil {
				t.Fatalf("Policy evaluation failed: %v", err)
			}

			if allowed != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, allowed)
			}
		})
	}
}

func TestValidatePolicy(t *testing.T) {
	// Create a temporary valid policy file
	tmpDir := t.TempDir()
	validPolicyPath := filepath.Join(tmpDir, "valid-policy.rego")

	validPolicy := `
package vault

default result = {"allow": false, "reason": ""}

result = {"allow": true, "reason": "Admin access"} {
	input.username == "admin"
}
`

	if err := os.WriteFile(validPolicyPath, []byte(validPolicy), 0644); err != nil {
		t.Fatalf("Failed to write valid policy: %v", err)
	}

	// Test valid policy
	if err := ValidatePolicy(validPolicyPath); err != nil {
		t.Errorf("Valid policy should not produce error: %v", err)
	}

	// Test invalid policy
	invalidPolicyPath := filepath.Join(tmpDir, "invalid-policy.rego")
	invalidPolicy := `
package vault

this is not valid rego
`

	if err := os.WriteFile(invalidPolicyPath, []byte(invalidPolicy), 0644); err != nil {
		t.Fatalf("Failed to write invalid policy: %v", err)
	}

	if err := ValidatePolicy(invalidPolicyPath); err == nil {
		t.Error("Invalid policy should produce error")
	}

	// Test non-existent policy
	if err := ValidatePolicy("/nonexistent/policy.rego"); err == nil {
		t.Error("Non-existent policy should produce error")
	}
}

func TestPolicyDirectory(t *testing.T) {
	// Create a temporary directory with multiple policy files
	tmpDir := t.TempDir()
	policyDir := filepath.Join(tmpDir, "policies")
	if err := os.Mkdir(policyDir, 0755); err != nil {
		t.Fatalf("Failed to create policy directory: %v", err)
	}

	// Create first policy file (allows PowerShell)
	policy1 := `
package vault

default result = {"allow": false, "reason": ""}

result = {"allow": true, "reason": "PowerShell access"} {
	input.process_name == "powershell.exe"
}
`
	if err := os.WriteFile(filepath.Join(policyDir, "allow-powershell.rego"), []byte(policy1), 0644); err != nil {
		t.Fatalf("Failed to write policy1: %v", err)
	}

	// Create second policy file (allows admin users)
	policy2 := `
package vault

result = {"allow": true, "reason": "Admin access"} {
	input.username == "admin"
}
`
	if err := os.WriteFile(filepath.Join(policyDir, "allow-admin.rego"), []byte(policy2), 0644); err != nil {
		t.Fatalf("Failed to write policy2: %v", err)
	}

	// Create policy evaluator from directory
	evaluator, err := NewPolicyEvaluator(policyDir)
	if err != nil {
		t.Fatalf("Failed to create policy evaluator from directory: %v", err)
	}

	tests := []struct {
		name     string
		request  *AccessRequest
		expected bool
	}{
		{
			name: "Allow PowerShell (from policy1)",
			request: &AccessRequest{
				Path:        "secret/test",
				Operation:   "READ",
				ProcessName: "powershell.exe",
				Username:    "user",
			},
			expected: true,
		},
		{
			name: "Allow admin user (from policy2)",
			request: &AccessRequest{
				Path:        "secret/test",
				Operation:   "READ",
				ProcessName: "notepad.exe",
				Username:    "admin",
			},
			expected: true,
		},
		{
			name: "Deny other",
			request: &AccessRequest{
				Path:        "secret/test",
				Operation:   "READ",
				ProcessName: "notepad.exe",
				Username:    "user",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, _, err := evaluator.Evaluate(context.Background(), tt.request)
			if err != nil {
				t.Fatalf("Policy evaluation failed: %v", err)
			}

			if allowed != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, allowed)
			}
		})
	}

	// Test validation of directory
	if err := ValidatePolicy(policyDir); err != nil {
		t.Errorf("Valid policy directory should not produce error: %v", err)
	}

	// Test reload from directory
	if err := evaluator.Reload(); err != nil {
		t.Errorf("Reload from directory should not fail: %v", err)
	}
}

func TestEmptyPolicyDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	emptyDir := filepath.Join(tmpDir, "empty")
	if err := os.Mkdir(emptyDir, 0755); err != nil {
		t.Fatalf("Failed to create empty directory: %v", err)
	}

	// Should fail with no .rego files
	_, err := NewPolicyEvaluator(emptyDir)
	if err == nil {
		t.Error("Empty policy directory should produce error")
	}

	if !strings.Contains(err.Error(), "no .rego policy files found") {
		t.Errorf("Expected 'no .rego policy files found' error, got: %v", err)
	}
}

func TestBuildAccessRequest(t *testing.T) {
	procInfo := &ProcessInfo{
		PID: 1234,
		UID: 1000,
		GID: 100,
	}

	request := BuildAccessRequest(procInfo, "secret/myapp/config", "read")

	if request.Path != "secret/myapp/config" {
		t.Errorf("Expected path 'secret/myapp/config', got '%s'", request.Path)
	}

	if request.Operation != "READ" {
		t.Errorf("Expected operation 'READ', got '%s'", request.Operation)
	}

	if request.PID != 1234 {
		t.Errorf("Expected PID 1234, got %d", request.PID)
	}

	if request.UID != 1000 {
		t.Errorf("Expected UID 1000, got %d", request.UID)
	}

	// Test with nil procInfo
	request2 := BuildAccessRequest(nil, "secret/test", "open")
	if request2.PID != 0 {
		t.Errorf("Expected PID 0 for nil procInfo, got %d", request2.PID)
	}
}

func TestCleanPath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/secret/myapp/config", "secret/myapp/config"},
		{"\\secret\\myapp\\config", "secret/myapp/config"},
		{"secret/myapp/config/", "secret/myapp/config"},
		{"/secret/myapp/config/", "secret/myapp/config"},
		{"\\\\secret\\\\myapp\\\\config\\\\", "secret/myapp/config"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := cleanPath(tt.input)
			if result != tt.expected {
				t.Errorf("cleanPath(%q) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}
