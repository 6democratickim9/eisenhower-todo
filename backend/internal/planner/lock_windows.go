//go:build windows

package planner

import (
	"os"
	"syscall"
	"unsafe"
)

var (
	kernel32Lock     = syscall.NewLazyDLL("kernel32.dll")
	procLockFileEx   = kernel32Lock.NewProc("LockFileEx")
	procUnlockFileEx = kernel32Lock.NewProc("UnlockFileEx")
)

const (
	lockfileFailImmediately = 0x1
	lockfileExclusive       = 0x2
)

type overlapped struct {
	Internal, InternalHigh uintptr
	Offset, OffsetHigh     uint32
	HEvent                 syscall.Handle
}

func lockFile(f *os.File) error {
	var o overlapped
	r, _, e := procLockFileEx.Call(f.Fd(), lockfileExclusive|lockfileFailImmediately, 0, 1, 0, uintptr(unsafe.Pointer(&o)))
	if r == 0 {
		return e
	}
	return nil
}
func unlockFile(f *os.File) error {
	var o overlapped
	r, _, e := procUnlockFileEx.Call(f.Fd(), 0, 1, 0, uintptr(unsafe.Pointer(&o)))
	if r == 0 {
		return e
	}
	return nil
}
