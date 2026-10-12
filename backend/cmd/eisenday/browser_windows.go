//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// launchBrowser opens Edge in --app mode pointing directly at the server URL.
// The Go server is already confirmed ready by waitForServer() in main.go
// before this function is called.
func launchBrowser(url string) (<-chan struct{}, error) {
	candidates := []string{
		filepath.Join(os.Getenv("ProgramFiles(x86)"), "Microsoft", "Edge", "Application", "msedge.exe"),
		filepath.Join(os.Getenv("ProgramFiles"), "Microsoft", "Edge", "Application", "msedge.exe"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Microsoft", "Edge", "Application", "msedge.exe"),
	}
	for _, edge := range candidates {
		if edge == "" {
			continue
		}
		if info, err := os.Stat(edge); err != nil || info.IsDir() {
			continue
		}
		cmd := exec.Command(edge,
			"--app="+url,
			"--no-first-run",
			"--no-default-browser-check",
		)
		if err := cmd.Start(); err != nil {
			continue
		}
		done := make(chan struct{})
		go func() {
			_ = cmd.Wait()
			close(done)
		}()
		return done, nil
	}

	// Fallback: default browser
	cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("open browser: %w", err)
	}
	return nil, nil
}
