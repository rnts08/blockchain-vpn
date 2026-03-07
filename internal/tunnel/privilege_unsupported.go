//go:build !linux && !darwin && !windows

package tunnel

import "fmt"

func EnsureElevatedPrivileges() error {
	return fmt.Errorf("automatic networking setup is not supported on this platform")
}
