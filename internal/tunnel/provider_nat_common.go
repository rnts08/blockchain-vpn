package tunnel

import (
	"fmt"
	"net"
	"strings"
)

func cidrFromIPPrefix(ip, prefix string) (string, error) {
	prefix = strings.TrimSpace(prefix)
	cidr := fmt.Sprintf("%s/%s", ip, prefix)
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", fmt.Errorf("invalid TUN CIDR %s: %w", cidr, err)
	}
	return ipNet.String(), nil
}
