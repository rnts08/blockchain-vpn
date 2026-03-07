//go:build windows

package tunnel

import (
	"fmt"

	"golang.org/x/sys/windows"
)

func EnsureElevatedPrivileges() error {
	token := windows.GetCurrentProcessToken()
	elevated := token.IsElevated()
	if !elevated {
		return fmt.Errorf("administrator privileges are required for networking setup on Windows; run in an elevated terminal")
	}
	return nil
}
