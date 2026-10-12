package planner

import (
	"fmt"
	"os"
	"path/filepath"
)

type DataFileInUseError struct{ Target string }

func (e DataFileInUseError) Error() string {
	return fmt.Sprintf("Data file is already in use: %s", e.Target)
}

type FileLock struct {
	Target, Path string
	File         *os.File
}

func NewFileLock(target string) *FileLock { return &FileLock{Target: target, Path: target + ".lock"} }
func (l *FileLock) Acquire() error {
	if err := os.MkdirAll(filepath.Dir(l.Path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(l.Path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	if st, _ := f.Stat(); st != nil && st.Size() == 0 {
		_, _ = f.WriteAt([]byte{'0'}, 0)
		_ = f.Sync()
	}
	if err = lockFile(f); err != nil {
		_ = f.Close()
		return DataFileInUseError{Target: l.Target}
	}
	l.File = f
	return nil
}
func (l *FileLock) Release() {
	if l.File != nil {
		_ = unlockFile(l.File)
		_ = l.File.Close()
		l.File = nil
	}
}
