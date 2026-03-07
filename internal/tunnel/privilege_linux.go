//go:build linux

package tunnel

import (
	"fmt"
	"os"
)

func EnsureElevatedPrivileges() error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("elevated privileges are required for networking setup on Linux; run as root")
	}
	return nil
}
