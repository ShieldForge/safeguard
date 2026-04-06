//go:build linux || darwin
// +build linux darwin

package filesystem

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

// getProcessNameAndPath returns the process name and full executable path for a given PID on Unix systems.
//
// This function uses the /proc filesystem (available on Linux and some Unix systems):
//  1. Reads /proc/[pid]/exe symlink to get the full executable path
//  2. Falls back to /proc/[pid]/cmdline if exe symlink fails
//  3. Extracts the process name from the path
//
// On macOS, /proc may not be available, so this may return empty strings.
// For better macOS support, additional methods (like using sysctl or ps) would be needed.
//
// Returns empty strings if:
//   - PID is <= 0
//   - /proc filesystem is not available
//   - Process doesn't exist or is not accessible
//
// Example: For PID 1234 running /usr/bin/python3, returns:
//
//	name: "python3"
//	path: "/usr/bin/python3"
func getProcessNameAndPath(pid int) (name string, path string) {
	if pid <= 0 {
		return "", ""
	}

	// Try to read the symbolic link /proc/[pid]/exe (Linux)
	exePath := filepath.Join("/proc", fmt.Sprintf("%d", pid), "exe")

	if target, err := os.Readlink(exePath); err == nil {
		path = target
		// Extract name from path
		name = filepath.Base(path)
		return name, path
	}

	// Fallback: try reading /proc/[pid]/cmdline for the command
	cmdlinePath := filepath.Join("/proc", fmt.Sprintf("%d", pid), "cmdline")
	if data, err := os.ReadFile(cmdlinePath); err == nil && len(data) > 0 {
		// cmdline is null-separated, get first component
		cmdline := string(data)
		parts := strings.Split(cmdline, "\x00")
		if len(parts) > 0 && parts[0] != "" {
			path = parts[0]
			name = filepath.Base(path)
			return name, path
		}
	}

	// On macOS or if /proc doesn't work, we could try using ps command
	// but for now just return empty
	return "", ""
}

// getUsernameFromPlatform retrieves the username on Unix systems from the UID.
//
// On Unix systems, the UID from FUSE context is reliable and can be used directly
// to look up the username from the system user database.
//
// This function:
//  1. Looks up the user by UID using user.LookupId
//  2. Returns the username if found
//  3. Returns "uid:N" format if lookup fails (user may not exist in database)
//
// Returns an empty string if procInfo is nil.
//
// Example: For UID 1000, might return "john" or "uid:1000" if lookup fails.
func getUsernameFromPlatform(procInfo *ProcessInfo) string {
	if procInfo == nil {
		return ""
	}

	// On Unix, UID is reliable, so use it
	u, err := user.LookupId(fmt.Sprintf("%d", procInfo.UID))
	if err != nil {
		// If lookup fails, return UID as string
		return fmt.Sprintf("uid:%d", procInfo.UID)
	}
	return u.Username
}
