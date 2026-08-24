//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

func acquireExclusiveDataFile(path string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}

	// Share READ only. Other processes may inspect the planner JSON, but they
	// cannot open it for writing or rename/delete it while this app is running.
	h, err := syscall.CreateFile(
		p,
		syscall.GENERIC_READ|syscall.GENERIC_WRITE,
		syscall.FILE_SHARE_READ,
		nil,
		syscall.OPEN_ALWAYS,
		syscall.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errDataFileInUse, err)
	}
	return os.NewFile(uintptr(h), path), nil
}

func writeExclusiveDataFile(f *os.File, path string, raw []byte) error {
	if f == nil {
		return fmt.Errorf("planner data file lock is missing")
	}

	// Snapshot previous valid contents. The main JSON is updated directly; no .tmp/.old file is created.
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return err
	}
	old, readErr := io.ReadAll(f)
	if readErr != nil {
		return readErr
	}
	if len(old) > 0 && json.Valid(old) {
		_ = os.WriteFile(path+".bak", old, 0o600)
	}

	rollback := func() {
		_ = f.Truncate(0)
		_, _ = f.Seek(0, io.SeekStart)
		if len(old) > 0 {
			_, _ = f.Write(old)
		}
		_ = f.Sync()
	}

	if err := f.Truncate(0); err != nil {
		return err
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		rollback()
		return err
	}
	if _, err := f.Write(raw); err != nil {
		rollback()
		return err
	}
	if err := f.Sync(); err != nil {
		rollback()
		return err
	}
	return nil
}
