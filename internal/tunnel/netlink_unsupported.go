//go:build !linux

package tunnel

import "fmt"

// getDefaultGateway is a stub for non-Linux platforms. It returns an error
// as this functionality is not supported.
func getDefaultGateway() (string, error) {
	return "", fmt.Errorf("automatic default gateway detection is not supported on this platform")
}
