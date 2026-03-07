//go:build linux

package tunnel

import (
	"fmt"
	"strings"
	"testing"
)

func TestProviderEgressNATMock(t *testing.T) {
	// Only run this test as a "soft" integration test if on Linux and we can override NAT command.
	// We'll use our runNATCommand mock.
	origRun := runNATCommand
	defer func() { runNATCommand = origRun }()

	var capturedCommands []string
	runNATCommand = func(name string, args ...string) (string, error) {
		cmd := name + " " + strings.Join(args, " ")
		capturedCommands = append(capturedCommands, cmd)

		// Mock responses
		if name == "ip" && args[0] == "route" {
			return "default via 192.168.1.1 dev eth0 proto dhcp metric 100", nil
		}
		return "success", nil
	}

	cleanup, err := setupProviderEgressNAT("bcvpn0", "10.10.0.1", "24", "")
	if err != nil {
		t.Fatalf("setupProviderEgressNAT failed: %v", err)
	}

	expectedCmds := []string{
		"ip route show default",
		"sysctl -w net.ipv4.ip_forward=1",
		"iptables -A FORWARD -i bcvpn0 -o eth0 -j ACCEPT",
		"iptables -A FORWARD -i eth0 -o bcvpn0 -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT",
		"iptables -t nat -A POSTROUTING -s 10.10.0.0/24 -o eth0 -j MASQUERADE",
	}

	for i, cmd := range expectedCmds {
		if i >= len(capturedCommands) {
			t.Errorf("expected command %q not found", cmd)
			continue
		}
		if capturedCommands[i] != cmd {
			t.Errorf("command mismatch at index %d: got %q, want %q", i, capturedCommands[i], cmd)
		}
	}

	if cleanup != nil {
		capturedCommands = nil // Reset for cleanup check
		cleanup()

		expectedCleanup := []string{
			"iptables -t nat -D POSTROUTING -s 10.10.0.0/24 -o eth0 -j MASQUERADE",
			"iptables -D FORWARD -i eth0 -o bcvpn0 -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT",
			"iptables -D FORWARD -i bcvpn0 -o eth0 -j ACCEPT",
			// sysctl restoration depends on host state, so we just check prefixes in the look below
		}

		for i, cmdPrefix := range expectedCleanup {
			if i >= len(capturedCommands) {
				t.Errorf("expected cleanup command starting with %q not found", cmdPrefix)
				continue
			}
			if !strings.HasPrefix(capturedCommands[i], cmdPrefix) {
				t.Errorf("cleanup command mismatch at index %d: got %q, want prefix %q", i, capturedCommands[i], cmdPrefix)
			}
		}
		// Special check for sysctl restoration
		lastCmd := capturedCommands[len(capturedCommands)-1]
		if !strings.HasPrefix(lastCmd, "sysctl -w net.ipv4.ip_forward=") {
			t.Errorf("last cleanup command should be sysctl restoration, got %q", lastCmd)
		}
	}
}

func TestProviderEgressNAT_Errors(t *testing.T) {
	origRun := runNATCommand
	defer func() { runNATCommand = origRun }()

	tests := []struct {
		name       string
		tunIfName  string
		outboundIf string
		mockErr    error
		wantErr    string
	}{
		{"EmptyInterface", "", "eth0", nil, "interface name is required"},
		{"CommandFailure", "bcvpn0", "eth0", fmt.Errorf("iptables error"), "failed to enable ipv4 forwarding"}, // Failure at sysctl
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runNATCommand = func(name string, args ...string) (string, error) {
				if tt.mockErr != nil {
					return "", tt.mockErr
				}
				return "ok", nil
			}
			_, err := setupProviderEgressNAT(tt.tunIfName, "10.0.0.1", "24", tt.outboundIf)
			if err == nil {
				t.Errorf("expected error containing %q, got nil", tt.wantErr)
			} else if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}
