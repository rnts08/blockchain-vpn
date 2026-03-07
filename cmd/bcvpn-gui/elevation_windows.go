//go:build windows

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

	argList := make([]string, 0, len(os.Args)-1)
	for _, a := range os.Args[1:] {
		argList = append(argList, psSingleQuote(a))
	}
	joinedArgs := strings.Join(argList, ",")
	ps := fmt.Sprintf("Start-Process -FilePath %s -Verb RunAs", psSingleQuote(exe))
	if joinedArgs != "" {
		ps += fmt.Sprintf(" -ArgumentList @(%s)", joinedArgs)
	}

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", ps)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start elevated process via Start-Process RunAs: %w", err)
	}
	return nil
}

func psSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func elevationHint() string {
	return "Use Relaunch Elevated to reopen the app with Administrator privileges."
}
