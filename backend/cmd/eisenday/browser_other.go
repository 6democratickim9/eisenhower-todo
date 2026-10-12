//go:build !windows

package main

import (
	"fmt"
	"os/exec"
	"runtime"
)

func launchBrowser(url string) (<-chan struct{}, error) {
	var cmd *exec.Cmd
	if runtime.GOOS == "darwin" {
		cmd = exec.Command("open", url)
	} else {
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("open browser: %w", err)
	}
	return nil, nil
}
