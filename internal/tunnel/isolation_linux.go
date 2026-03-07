//go:build linux

package tunnel

import (
	"fmt"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

func applyProviderIsolation(mode string) error {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", "none":
		return nil
	case "sandbox":
		if err := unix.Prctl(unix.PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0); err != nil {
			return fmt.Errorf("failed to enable no_new_privs: %w", err)
		}
		if err := unix.Prctl(unix.PR_SET_DUMPABLE, 0, 0, 0, 0); err != nil {
			return fmt.Errorf("failed to disable core dumps: %w", err)
		}
		rl := &unix.Rlimit{Cur: 0, Max: 0}
		if err := unix.Setrlimit(unix.RLIMIT_CORE, rl); err != nil {
			return fmt.Errorf("failed to set RLIMIT_CORE: %w", err)
		}
		syscall.Umask(0o077)
		return nil
	default:
		return fmt.Errorf("unknown isolation mode %q (supported: none, sandbox)", mode)
	}
}
