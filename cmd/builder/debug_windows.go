package main

import "syscall"

// isDebuggerAttached checks if a debugger is attached via Windows kernel32 IsDebuggerPresent.
func isDebuggerAttached() bool {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	proc := kernel32.NewProc("IsDebuggerPresent")
	ret, _, _ := proc.Call()
	return ret != 0
}
