//go:build linux

package tunnel

import (
	"fmt"
	"os"
)

var osGeteuid = os.Geteuid

func EnsureElevatedPrivileges() error {
	if osGeteuid() != 0 {
		return fmt.Errorf("elevated privileges are required for networking setup on Linux; run as root")
	}
	return nil
}
