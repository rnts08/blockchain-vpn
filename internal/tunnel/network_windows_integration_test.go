//go:build windows

package tunnel

import (
	"strings"
	"testing"
)

func TestWindowsConfigureClientNetworkSetupAndRestore(t *testing.T) {
	origCmd := windowsRunCommand
	origPS := windowsRunPowerShell
	defer func() {
		windowsRunCommand = origCmd
		windowsRunPowerShell = origPS
	}()

	var cmdCalls []string
	var psCalls []string

	windowsRunCommand = func(name string, args ...string) (string, error) {
		cmdCalls = append(cmdCalls, name+" "+strings.Join(args, " "))
		return "", nil
	}
	windowsRunPowerShell = func(cmd string) (string, error) {
		psCalls = append(psCalls, cmd)
		switch {
		case strings.Contains(cmd, "Get-NetRoute -AddressFamily IPv4"):
			return `{"InterfaceAlias":"Ethernet","InterfaceIndex":12,"NextHop":"192.168.1.1"}`, nil
		case strings.Contains(cmd, "Get-NetIPInterface -AddressFamily IPv4 -InterfaceAlias 'bcvpn1'"):
			return "42", nil
		case strings.Contains(cmd, "Get-DnsClientServerAddress"):
			return `["8.8.8.8","1.1.1.1"]`, nil
		default:
			return "", nil
		}
	}

	cleanup, err := configureClientNetwork("bcvpn1", "1.2.3.4")
	if err != nil {
		t.Fatalf("configureClientNetwork failed: %v", err)
	}
	cleanup()

	expectContainsWindows(t, cmdCalls, "route ADD 0.0.0.0 MASK 128.0.0.0 0.0.0.0 IF 42")
	expectContainsWindows(t, cmdCalls, "route ADD 128.0.0.0 MASK 128.0.0.0 0.0.0.0 IF 42")
	expectContainsWindows(t, cmdCalls, "route DELETE 0.0.0.0 MASK 128.0.0.0 0.0.0.0 IF 42")
	expectContainsWindows(t, cmdCalls, "route DELETE 128.0.0.0 MASK 128.0.0.0 0.0.0.0 IF 42")

	expectContainsPS(t, psCalls, "Set-DnsClientServerAddress -InterfaceAlias 'Ethernet' -ServerAddresses @('1.1.1.1','8.8.8.8')")
	expectContainsPS(t, psCalls, "Set-DnsClientServerAddress -InterfaceAlias 'Ethernet' -ServerAddresses @('8.8.8.8','1.1.1.1')")
}

func TestWindowsConfigureClientNetworkRepeatedSetupCleanup(t *testing.T) {
	origCmd := windowsRunCommand
	origPS := windowsRunPowerShell
	defer func() {
		windowsRunCommand = origCmd
		windowsRunPowerShell = origPS
	}()

	windowsRunCommand = func(name string, args ...string) (string, error) { return "", nil }
	windowsRunPowerShell = func(cmd string) (string, error) {
		switch {
		case strings.Contains(cmd, "Get-NetRoute -AddressFamily IPv4"):
			return `{"InterfaceAlias":"Ethernet","InterfaceIndex":12,"NextHop":"192.168.1.1"}`, nil
		case strings.Contains(cmd, "Get-NetIPInterface -AddressFamily IPv4 -InterfaceAlias 'bcvpn1'"):
			return "42", nil
		case strings.Contains(cmd, "Get-DnsClientServerAddress"):
			return `["8.8.8.8","1.1.1.1"]`, nil
		default:
			return "", nil
		}
	}

	const rounds = 5
	for i := 0; i < rounds; i++ {
		cleanup, err := configureClientNetwork("bcvpn1", "1.2.3.4")
		if err != nil {
			t.Fatalf("round %d configureClientNetwork failed: %v", i, err)
		}
		cleanup()
	}
}

func expectContainsWindows(t *testing.T, calls []string, want string) {
	t.Helper()
	for _, c := range calls {
		if c == want {
			return
		}
	}
	t.Fatalf("expected windows command %q in %#v", want, calls)
}

func expectContainsPS(t *testing.T, calls []string, contains string) {
	t.Helper()
	for _, c := range calls {
		if strings.Contains(c, contains) {
			return
		}
	}
	t.Fatalf("expected powershell command containing %q in %#v", contains, calls)
}
