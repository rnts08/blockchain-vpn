//go:build !linux

package tunnel

import "github.com/songgao/water"

func platformSpecificParams(_ string) water.PlatformSpecificParams {
	return water.PlatformSpecificParams{}
}
