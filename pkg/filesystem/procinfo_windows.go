//go:build windows
// +build windows

package filesystem

import (
	"syscall"
	"unsafe"
)

var (
	kernel32                       = syscall.NewLazyDLL("kernel32.dll")
	psapi                          = syscall.NewLazyDLL("psapi.dll")
	advapi32                       = syscall.NewLazyDLL("advapi32.dll")
	procOpenProcess                = kernel32.NewProc("OpenProcess")
	procCloseHandle                = kernel32.NewProc("CloseHandle")
	procQueryFullProcessImageNameW = kernel32.NewProc("QueryFullProcessImageNameW")
	procGetModuleBaseNameW         = psapi.NewProc("GetModuleBaseNameW")
	procOpenProcessToken           = advapi32.NewProc("OpenProcessToken")
	procGetTokenInformation        = advapi32.NewProc("GetTokenInformation")
	procLookupAccountSidW          = advapi32.NewProc("LookupAccountSidW")
)

const (
	PROCESS_QUERY_LIMITED_INFORMATION = 0x1000
	PROCESS_QUERY_INFORMATION         = 0x0400
	TOKEN_QUERY                       = 0x0008
	TokenUser                         = 1
	MAX_PATH                          = 260
)

// getProcessNameAndPath returns the process name and full executable path for a given PID on Windows.
//
// This function uses Windows API calls to query process information:
//   - QueryFullProcessImageNameW: Gets the full executable path
//   - GetModuleBaseNameW: Gets just the executable filename
//
// The function attempts to open the process with PROCESS_QUERY_LIMITED_INFORMATION
// rights, which works even for processes running under different user contexts.
//
// Returns empty strings if:
//   - PID is <= 0
//   - Process cannot be opened (access denied, process doesn't exist)
//   - API calls fail
//
// Example: For PID 1234 running notepad.exe, returns:
//
//	name: "notepad.exe"
//	path: "C:\\Windows\\System32\\notepad.exe"
func getProcessNameAndPath(pid int) (name string, path string) {
	if pid <= 0 {
		return "", ""
	}

	// Open the process with limited query rights
	handle, _, _ := procOpenProcess.Call(
		uintptr(PROCESS_QUERY_LIMITED_INFORMATION),
		uintptr(0),
		uintptr(pid),
	)

	if handle == 0 {
		return "", ""
	}
	defer procCloseHandle.Call(handle)

	// Get the full executable path
	pathBuf := make([]uint16, MAX_PATH*2)
	pathSize := uint32(len(pathBuf))

	ret, _, _ := procQueryFullProcessImageNameW.Call(
		handle,
		uintptr(0), // 0 = native path format
		uintptr(unsafe.Pointer(&pathBuf[0])),
		uintptr(unsafe.Pointer(&pathSize)),
	)

	if ret != 0 && pathSize > 0 {
		path = syscall.UTF16ToString(pathBuf[:pathSize])
	}

	// Get the base name (just the executable name)
	nameBuf := make([]uint16, MAX_PATH)
	ret, _, _ = procGetModuleBaseNameW.Call(
		handle,
		uintptr(0), // 0 = process itself (not a specific module)
		uintptr(unsafe.Pointer(&nameBuf[0])),
		uintptr(len(nameBuf)),
	)

	if ret != 0 {
		name = syscall.UTF16ToString(nameBuf)
	}

	// If we got a name but no path, or vice versa, use what we have
	if name == "" && path != "" {
		// Extract name from path
		for i := len(path) - 1; i >= 0; i-- {
			if path[i] == '\\' || path[i] == '/' {
				name = path[i+1:]
				break
			}
		}
	}

	return name, path
}

// getUsernameFromPID retrieves the username for a process on Windows using its PID.
//
// This function uses Windows Security API to:
//  1. Open the process with PROCESS_QUERY_INFORMATION rights
//  2. Get the process token
//  3. Query TokenUser information to get the SID
//  4. Look up the account name from the SID
//
// This is more reliable than using the UID from FUSE context on Windows, as
// WinFsp may not always provide accurate UID mapping.
//
// Returns an empty string if:
//   - PID is <= 0
//   - Process cannot be opened (insufficient permissions, doesn't exist)
//   - Token information cannot be retrieved
//   - SID lookup fails
func getUsernameFromPID(pid int) string {
	if pid <= 0 {
		return ""
	}

	// Open the process
	handle, _, _ := procOpenProcess.Call(
		uintptr(PROCESS_QUERY_INFORMATION),
		uintptr(0),
		uintptr(pid),
	)

	if handle == 0 {
		return ""
	}
	defer procCloseHandle.Call(handle)

	// Open the process token
	var token syscall.Handle
	ret, _, _ := procOpenProcessToken.Call(
		handle,
		uintptr(TOKEN_QUERY),
		uintptr(unsafe.Pointer(&token)),
	)

	if ret == 0 {
		return ""
	}
	defer syscall.CloseHandle(token)

	// Get the token user information size
	var size uint32
	procGetTokenInformation.Call(
		uintptr(token),
		uintptr(TokenUser),
		uintptr(0),
		uintptr(0),
		uintptr(unsafe.Pointer(&size)),
	)

	if size == 0 {
		return ""
	}

	// Get the token user information
	buf := make([]byte, size)
	ret, _, _ = procGetTokenInformation.Call(
		uintptr(token),
		uintptr(TokenUser),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(size),
		uintptr(unsafe.Pointer(&size)),
	)

	if ret == 0 {
		return ""
	}

	// Extract the SID from the TOKEN_USER structure
	// TOKEN_USER structure has SID_AND_ATTRIBUTES which starts with a pointer to SID
	sidPtr := *(**byte)(unsafe.Pointer(&buf[0]))

	// Lookup the account name from the SID
	var nameSize, domainSize uint32 = 256, 256
	nameBuf := make([]uint16, nameSize)
	domainBuf := make([]uint16, domainSize)
	var sidType uint32

	ret, _, _ = procLookupAccountSidW.Call(
		uintptr(0), // local computer
		uintptr(unsafe.Pointer(sidPtr)),
		uintptr(unsafe.Pointer(&nameBuf[0])),
		uintptr(unsafe.Pointer(&nameSize)),
		uintptr(unsafe.Pointer(&domainBuf[0])),
		uintptr(unsafe.Pointer(&domainSize)),
		uintptr(unsafe.Pointer(&sidType)),
	)

	if ret == 0 {
		return ""
	}

	domain := syscall.UTF16ToString(domainBuf[:domainSize])
	name := syscall.UTF16ToString(nameBuf[:nameSize])

	if domain != "" && name != "" {
		return domain + "\\" + name
	}
	return name
}

// getUsernameFromPlatform retrieves the username on Windows from process information.
//
// On Windows, this function ignores the UID field from FUSE context (which is not
// a valid Windows SID) and instead queries the process token directly using the PID.
//
// This delegates to getUsernameFromPID which performs the actual Windows API calls.
//
// Returns an empty string if the username cannot be determined.
func getUsernameFromPlatform(procInfo *ProcessInfo) string {
	// On Windows, use the PID to get the actual username
	// The UID from FUSE context is not a valid Windows SID
	return getUsernameFromPID(procInfo.PID)
}
