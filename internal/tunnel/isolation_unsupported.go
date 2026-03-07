//go:build !linux

package tunnel

import (
	"fmt"
	"strings"
)

func applyProviderIsolation(mode string) error {
	mode = strings.ToLower(strings.TrimSpace(mode))
	switch mode {
	case "", "none":
		return nil
	case "sandbox":
		return fmt.Errorf("sandbox isolation mode is currently supported only on Linux")
	default:
		return fmt.Errorf("unknown isolation mode %q (supported: none)", mode)
	}
}
