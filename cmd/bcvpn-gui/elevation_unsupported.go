//go:build !linux && !darwin && !windows

package main

import "fmt"

func canRelaunchElevated() bool {
	return false
}

func relaunchElevated() error {
	return fmt.Errorf("automatic elevation relaunch is not supported on this platform")
}

func elevationHint() string {
	return "Automatic elevation is not supported on this platform."
}
