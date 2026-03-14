//go:build darwin

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

	var parts []string
	parts = append(parts, shellQuote(exe))
	for _, a := range os.Args[1:] {
		parts = append(parts, shellQuote(a))
	}
	commandLine := strings.Join(parts, " ")
	script := fmt.Sprintf(`do shell script %q with administrator privileges`, commandLine)

	cmd := exec.Command("osascript", "-e", script)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start elevated process via osascript: %w", err)
	}
	return nil
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

func elevationHint() string {
	return "Use Relaunch Elevated to authorize administrator privileges through macOS."
}
