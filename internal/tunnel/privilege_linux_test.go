//go:build linux

package tunnel

import (
	"testing"
)

func TestEnsureElevatedPrivileges(t *testing.T) {
	t.Parallel()

	t.Run("root user passes", func(t *testing.T) {
		t.Skip("requires non-root test environment")
	})

	t.Run("non-root user fails", func(t *testing.T) {
		t.Skip("requires non-root test environment")
	})
}

func TestEnsureElevatedPrivilegesMock(t *testing.T) {
	origGeteuid := osGeteuid
	t.Cleanup(func() { osGeteuid = origGeteuid })

	osGeteuid = func() int { return 0 }
	err := EnsureElevatedPrivileges()
	if err != nil {
		t.Errorf("expected root to pass, got error: %v", err)
	}

	osGeteuid = func() int { return 1000 }
	err = EnsureElevatedPrivileges()
	if err == nil {
		t.Error("expected non-root to fail, got nil error")
	}
}
