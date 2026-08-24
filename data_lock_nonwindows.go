//go:build !windows

package main

import "os"

func acquireExclusiveDataFile(path string) (*os.File, error) {
	return nil, nil
}

func writeExclusiveDataFile(f *os.File, path string, raw []byte) error {
	return writeBytesAtomic(path, raw, true)
}
