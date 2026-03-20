//go:build !linux && !darwin && !windows

package tunnel

func recoverPendingNetworkStateFromMarker(m *networkCleanupMarker) error {
	_ = m
	return nil
}

func cleanupStaleTunInterfaces(prefixes []string) error {
	_ = prefixes
	return nil
}
