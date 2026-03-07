//go:build !linux && !darwin && !windows

package tunnel

func recoverPendingNetworkStateFromMarker(m *networkCleanupMarker) error {
	_ = m
	return nil
}
