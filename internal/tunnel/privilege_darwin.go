//go:build darwin

package tunnel

import (
	"fmt"
	"os"
)

func EnsureElevatedPrivileges() error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("elevated privileges are required for networking setup on macOS; run with sudo or as an administrator")
	}
	return nil
}
