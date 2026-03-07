//go:build linux

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func canRelaunchElevated() bool {
	return true
}

func relaunchElevated() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}

	args := append([]string{exe}, os.Args[1:]...)
	cmd := exec.Command("pkexec", args...)
	cmd.Env = append(os.Environ(),
		"DISPLAY="+os.Getenv("DISPLAY"),
		"XAUTHORITY="+os.Getenv("XAUTHORITY"),
	)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start elevated process via pkexec: %w", err)
	}
	return nil
}

func elevationHint() string {
	return strings.TrimSpace("Use the Relaunch Elevated button to request administrator privileges.")
}
