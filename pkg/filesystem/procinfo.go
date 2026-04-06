package filesystem

import (
	"fmt"
)

// ProcessDetails contains comprehensive information about a process, including
// resolved names and paths that are platform-specific.
//
// This structure extends the basic ProcessInfo with human-readable details:
//   - Process name (executable filename)
//   - Full executable path
//   - Username resolved from UID
//
// These details are used for logging, monitoring, and policy evaluation.
type ProcessDetails struct {
	PID            int    // Process ID
	ProcessName    string // Executable name (e.g., "notepad.exe")
	ExecutablePath string // Full path to executable
	UID            uint32 // User ID
	Username       string // Resolved username
	GID            uint32 // Group ID
}

// resolveProcessDetails enriches ProcessInfo with resolved names and paths.
//
// This function performs platform-specific lookups to convert raw process
// information (PIDs, UIDs) into human-readable details like process names
// and usernames.
//
// Returns nil if procInfo is nil. The returned ProcessDetails may have
// empty string fields if resolution fails for any particular detail.
func resolveProcessDetails(procInfo *ProcessInfo) *ProcessDetails {
	if procInfo == nil {
		return nil
	}

	details := &ProcessDetails{
		PID: procInfo.PID,
		UID: procInfo.UID,
		GID: procInfo.GID,
	}

	// Resolve process name and path
	details.ProcessName, details.ExecutablePath = getProcessNameAndPath(procInfo.PID)

	// Resolve username (platform-specific)
	details.Username = getUsernameFromProcess(procInfo)

	return details
}

// getUsernameFromProcess resolves a username from process information.
//
// This is a wrapper function that delegates to platform-specific implementations
// (getUsernameFromPlatform) which handle the different username resolution
// mechanisms on Windows vs Unix systems.
//
// On Unix: Looks up username from UID using system user database
// On Windows: Retrieves username from process token via Windows API
//
// Returns an empty string if procInfdetails into a human-readable string for logging.
//
// The format includes:
//   - PID and process name
//   - Executable path (if different from name)
//   - UID and username
//   - GID
//
// Example output:
//
//	[PID: 1234 (notepad.exe) - C:\Windows\notepad.exe, UID: 1000 (john), GID: 100]
//
// Returns "[NO CONTEXT]" if details is nil. cannot be resolved.
func getUsernameFromProcess(procInfo *ProcessInfo) string {
	if procInfo == nil {
		return ""
	}
	return getUsernameFromPlatform(procInfo)
}

// formatProcessInfo formats process information for logging
func formatProcessInfo(details *ProcessDetails) string {
	if details == nil {
		return "[NO CONTEXT]"
	}

	result := fmt.Sprintf("[PID: %d", details.PID)

	if details.ProcessName != "" {
		result += fmt.Sprintf(" (%s)", details.ProcessName)
		if details.ExecutablePath != "" && details.ExecutablePath != details.ProcessName {
			result += fmt.Sprintf(" - %s", details.ExecutablePath)
		}
	}

	result += fmt.Sprintf(", UID: %d", details.UID)
	if details.Username != "" {
		result += fmt.Sprintf(" (%s)", details.Username)
	}

	result += fmt.Sprintf(", GID: %d]", details.GID)

	return result
}
