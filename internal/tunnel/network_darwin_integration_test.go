//go:build darwin

package tunnel

import (
	"strings"
	"testing"
)

func TestDarwinConfigureClientNetworkSetupAndRestore(t *testing.T) {
	orig := darwinRunCommand
	defer func() { darwinRunCommand = orig }()

	var calls []string
	darwinRunCommand = func(name string, args ...string) (string, error) {
		cmd := name + " " + strings.Join(args, " ")
		calls = append(calls, cmd)

		switch cmd {
		case "route -n get default":
			return "gateway: 192.168.1.1\ninterface: en0\n", nil
		case "networksetup -listnetworkserviceorder":
			return "(1) Wi-Fi\n(Hardware Port: Wi-Fi, Device: en0)\n", nil
		case "networksetup -getdnsservers Wi-Fi":
			return "8.8.8.8\n1.1.1.1\n", nil
		default:
			return "", nil
		}
	}

	cleanup, err := configureClientNetwork("utun9", "1.2.3.4")
	if err != nil {
		t.Fatalf("configureClientNetwork failed: %v", err)
	}
	cleanup()

	expectContains(t, calls, "route -n add -net 0.0.0.0/1 -interface utun9")
	expectContains(t, calls, "route -n add -net 128.0.0.0/1 -interface utun9")
	expectContains(t, calls, "networksetup -setdnsservers Wi-Fi 1.1.1.1 8.8.8.8")
	expectContains(t, calls, "route -n delete -net 0.0.0.0/1 -interface utun9")
	expectContains(t, calls, "route -n delete -net 128.0.0.0/1 -interface utun9")
	expectContains(t, calls, "networksetup -setdnsservers Wi-Fi 8.8.8.8 1.1.1.1")
}

func TestDarwinConfigureClientNetworkRepeatedSetupCleanup(t *testing.T) {
	orig := darwinRunCommand
	defer func() { darwinRunCommand = orig }()

	darwinRunCommand = func(name string, args ...string) (string, error) {
		cmd := name + " " + strings.Join(args, " ")
		switch cmd {
		case "route -n get default":
			return "gateway: 192.168.1.1\ninterface: en0\n", nil
		case "networksetup -listnetworkserviceorder":
			return "(1) Wi-Fi\n(Hardware Port: Wi-Fi, Device: en0)\n", nil
		case "networksetup -getdnsservers Wi-Fi":
			return "8.8.8.8\n1.1.1.1\n", nil
		default:
			return "", nil
		}
	}

	const rounds = 5
	for i := 0; i < rounds; i++ {
		cleanup, err := configureClientNetwork("utun9", "1.2.3.4")
		if err != nil {
			t.Fatalf("round %d configureClientNetwork failed: %v", i, err)
		}
		cleanup()
	}
}

func expectContains(t *testing.T, calls []string, want string) {
	t.Helper()
	for _, c := range calls {
		if c == want {
			return
		}
	}
	t.Fatalf("expected call %q in %#v", want, calls)
}
