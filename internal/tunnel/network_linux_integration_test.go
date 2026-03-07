//go:build linux

package tunnel

import (
	"errors"
	"io/fs"
	"testing"
)

func TestLinuxDNSSetupRestore(t *testing.T) {
	origRead := linuxReadFile
	origWrite := linuxWriteFile
	origRemove := linuxRemoveFile
	defer func() {
		linuxReadFile = origRead
		linuxWriteFile = origWrite
		linuxRemoveFile = origRemove
	}()

	files := map[string][]byte{
		resolvConfPath: []byte("nameserver 9.9.9.9\n"),
	}
	linuxReadFile = func(path string) ([]byte, error) {
		v, ok := files[path]
		if !ok {
			return nil, errors.New("not found")
		}
		return v, nil
	}
	linuxWriteFile = func(path string, data []byte, perm fs.FileMode) error {
		_ = perm
		files[path] = append([]byte(nil), data...)
		return nil
	}
	linuxRemoveFile = func(path string) error {
		delete(files, path)
		return nil
	}

	if err := setupDNS(); err != nil {
		t.Fatalf("setupDNS failed: %v", err)
	}
	if got := string(files[resolvConfPath]); got != secureDNS {
		t.Fatalf("unexpected resolv.conf after setup: %q", got)
	}

	restoreDNS()
	if got := string(files[resolvConfPath]); got != "nameserver 9.9.9.9\n" {
		t.Fatalf("unexpected resolv.conf after restore: %q", got)
	}
	if _, ok := files[resolvBackupPath]; ok {
		t.Fatal("expected backup file to be removed after restore")
	}
}

func TestLinuxConfigureClientNetworkCleanupCallsRestore(t *testing.T) {
	origGetGW := linuxGetDefaultGateway
	origSetupRouting := linuxSetupRouting
	origRestoreRouting := linuxRestoreRouting
	origSetupDNS := linuxSetupDNS
	origRestoreDNS := linuxRestoreDNS
	defer func() {
		linuxGetDefaultGateway = origGetGW
		linuxSetupRouting = origSetupRouting
		linuxRestoreRouting = origRestoreRouting
		linuxSetupDNS = origSetupDNS
		linuxRestoreDNS = origRestoreDNS
	}()

	linuxGetDefaultGateway = func() (string, error) { return "192.168.1.1", nil }
	linuxSetupRouting = func(ifaceName, providerHost, defaultGW string) error {
		if ifaceName != "bcvpn1" || providerHost != "1.2.3.4" || defaultGW != "192.168.1.1" {
			t.Fatalf("unexpected routing setup args: %s %s %s", ifaceName, providerHost, defaultGW)
		}
		return nil
	}
	restoreRoutingCalled := false
	linuxRestoreRouting = func(ifaceName, providerHost string) {
		restoreRoutingCalled = true
		if ifaceName != "bcvpn1" || providerHost != "1.2.3.4" {
			t.Fatalf("unexpected restore routing args: %s %s", ifaceName, providerHost)
		}
	}
	linuxSetupDNS = func() error { return nil }
	restoreDNSCalled := false
	linuxRestoreDNS = func() {
		restoreDNSCalled = true
	}

	cleanup, err := configureClientNetwork("bcvpn1", "1.2.3.4")
	if err != nil {
		t.Fatalf("configureClientNetwork failed: %v", err)
	}
	cleanup()

	if !restoreRoutingCalled {
		t.Fatal("expected restoreRouting to be called")
	}
	if !restoreDNSCalled {
		t.Fatal("expected restoreDNS to be called")
	}
}

func TestLinuxConfigureClientNetworkRepeatedSetupCleanup(t *testing.T) {
	origGetGW := linuxGetDefaultGateway
	origSetupRouting := linuxSetupRouting
	origRestoreRouting := linuxRestoreRouting
	origSetupDNS := linuxSetupDNS
	origRestoreDNS := linuxRestoreDNS
	defer func() {
		linuxGetDefaultGateway = origGetGW
		linuxSetupRouting = origSetupRouting
		linuxRestoreRouting = origRestoreRouting
		linuxSetupDNS = origSetupDNS
		linuxRestoreDNS = origRestoreDNS
	}()

	linuxGetDefaultGateway = func() (string, error) { return "192.168.1.1", nil }
	linuxSetupRouting = func(ifaceName, providerHost, defaultGW string) error { return nil }
	var routeRestores int
	linuxRestoreRouting = func(ifaceName, providerHost string) { routeRestores++ }
	linuxSetupDNS = func() error { return nil }
	var dnsRestores int
	linuxRestoreDNS = func() { dnsRestores++ }

	const rounds = 5
	for i := 0; i < rounds; i++ {
		cleanup, err := configureClientNetwork("bcvpn1", "1.2.3.4")
		if err != nil {
			t.Fatalf("round %d configureClientNetwork failed: %v", i, err)
		}
		cleanup()
	}

	if routeRestores != rounds {
		t.Fatalf("expected %d route restore calls, got %d", rounds, routeRestores)
	}
	if dnsRestores != rounds {
		t.Fatalf("expected %d dns restore calls, got %d", rounds, dnsRestores)
	}
}
