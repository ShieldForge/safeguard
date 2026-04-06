package filesystem

import (
	"os"
	"testing"
)

func TestCurrentProcessAlwaysAllowed(t *testing.T) {
	// Create a mock client
	client := &MockVaultClient{}

	// Create filesystem with policy that denies everything
	opts := VaultFSOptions{
		Debug:         false,
		AccessControl: true,
		AllowedPIDs:   []int{99999}, // Some PID that's not the current process
	}

	fs := NewVaultFSWithOptions(client, opts)

	// Get current process PID
	currentPID := os.Getpid()

	// Create ProcessInfo for current process
	procInfo := &ProcessInfo{
		PID: currentPID,
		UID: 1000,
		GID: 1000,
	}

	// Current process should always be allowed, even with restricted PIDs
	if !fs.checkAccess(procInfo, "secret/test", "READ") {
		t.Error("Current process should always be allowed access")
	}

	// Different process should be denied (not in allowedPIDs)
	otherProcInfo := &ProcessInfo{
		PID: currentPID + 1,
		UID: 1000,
		GID: 1000,
	}

	if fs.checkAccess(otherProcInfo, "secret/test", "READ") {
		t.Error("Other process should be denied when not in allowedPIDs")
	}

	// System/kernel operations (PID -1) should always be allowed
	systemProcInfo := &ProcessInfo{
		PID: -1,
		UID: 0,
		GID: 0,
	}

	if !fs.checkAccess(systemProcInfo, "secret/test", "READ") {
		t.Error("System/kernel operations (PID -1) should always be allowed")
	}
}
