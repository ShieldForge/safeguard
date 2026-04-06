//go:build !windows

package main

import (
	"os"
	"strings"
)

// isDebuggerAttached checks for an attached debugger by inspecting /proc/self/status
// for a non-zero TracerPid (Linux) or falls back to false on other Unix systems.
func isDebuggerAttached() bool {
	data, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "TracerPid:") {
			pid := strings.TrimSpace(strings.TrimPrefix(line, "TracerPid:"))
			return pid != "0"
		}
	}
	return false
}
