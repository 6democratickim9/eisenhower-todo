//go:build windows

package planner

import (
	"syscall"
	"unsafe"
)

var (
	kernel32Move    = syscall.NewLazyDLL("kernel32.dll")
	procMoveFileExW = kernel32Move.NewProc("MoveFileExW")
)

const (
	moveFileReplaceExisting = 0x1
	moveFileWriteThrough    = 0x8
)

func replaceFile(src, dst string) error {
	s, _ := syscall.UTF16PtrFromString(src)
	d, _ := syscall.UTF16PtrFromString(dst)
	r, _, e := procMoveFileExW.Call(uintptr(unsafe.Pointer(s)), uintptr(unsafe.Pointer(d)), moveFileReplaceExisting|moveFileWriteThrough)
	if r == 0 {
		return e
	}
	return nil
}
func syncDir(path string) error { return nil }
