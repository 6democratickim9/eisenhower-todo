//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

func showAlreadyRunningWarning(message string) {
	user32 := syscall.NewLazyDLL("user32.dll")
	messageBox := user32.NewProc("MessageBoxW")
	text, _ := syscall.UTF16PtrFromString(message)
	title, _ := syscall.UTF16PtrFromString("Eisenhower Todo v1.4.6")
	const MB_OK = 0x00000000
	const MB_ICONWARNING = 0x00000030
	messageBox.Call(
		0,
		uintptr(unsafe.Pointer(text)),
		uintptr(unsafe.Pointer(title)),
		uintptr(MB_OK|MB_ICONWARNING),
	)
}
