//go:build linux

package tunnel

import "github.com/songgao/water"

func platformSpecificParams(ifaceName string) water.PlatformSpecificParams {
	return water.PlatformSpecificParams{Name: ifaceName}
}
