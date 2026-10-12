//go:build !windows

package planner

import "os"

func replaceFile(src, dst string) error { return os.Rename(src, dst) }
func syncDir(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	_ = f.Sync()
	return nil
}
