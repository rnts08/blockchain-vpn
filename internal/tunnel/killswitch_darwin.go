//go:build darwin

package tunnel

import (
	"fmt"
	"strings"
)

const darwinKillSwitchAnchor = "com.apple/blockchainvpn-killswitch"

func setupKillSwitch(tunIfName, providerHost string) (func(), error) {
	if strings.TrimSpace(tunIfName) == "" {
		return nil, fmt.Errorf("kill switch requires client TUN interface name")
	}
	providerIP, err := resolveProviderIPv4(providerHost)
	if err != nil {
		return nil, err
	}

	if _, err := runCmd("pfctl", "-E"); err != nil {
		return nil, fmt.Errorf("failed to enable pf for kill switch: %w", err)
	}

	rules := fmt.Sprintf(
		"pass out quick on lo0 all\npass out quick on %s all\npass out quick to %s/32 all\nblock drop out quick all\n",
		tunIfName,
		providerIP,
	)
	if _, err := runCmd("sh", "-c", fmt.Sprintf("printf %%s %q | pfctl -a %s -f -", rules, darwinKillSwitchAnchor)); err != nil {
		return nil, fmt.Errorf("failed to load kill switch PF rules: %w", err)
	}

	return func() {
		_, _ = runCmd("pfctl", "-a", darwinKillSwitchAnchor, "-F", "all")
	}, nil
}
