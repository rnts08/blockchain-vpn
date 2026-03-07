//go:build linux

package tunnel

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/vishvananda/netlink"
)

const resolvConfPath = "/etc/resolv.conf"
const resolvBackupPath = "/etc/resolv.conf.bcvpn-backup"
const secureDNS = "nameserver 1.1.1.1\nnameserver 8.8.8.8\n"

var (
	linuxGetDefaultGateway = getDefaultGateway
	linuxSetupRouting      = setupRouting
	linuxRestoreRouting    = restoreRouting
	linuxSetupDNS          = setupDNS
	linuxRestoreDNS        = restoreDNS
	linuxReadFile          = os.ReadFile
	linuxWriteFile         = os.WriteFile
	linuxRemoveFile        = os.Remove
)

func configureTunInterface(ifaceName, ip, subnetMask string) error {
	link, err := netlink.LinkByName(ifaceName)
	if err != nil {
		return fmt.Errorf("failed to get link for %s: %w", ifaceName, err)
	}
	addr, err := netlink.ParseAddr(ip + "/" + subnetMask)
	if err != nil {
		return fmt.Errorf("failed to parse address: %w", err)
	}
	if err := netlink.AddrAdd(link, addr); err != nil {
		return fmt.Errorf("failed to assign IP to TUN interface: %w", err)
	}
	if err := netlink.LinkSetUp(link); err != nil {
		return fmt.Errorf("failed to bring up TUN interface: %w", err)
	}
	return nil
}

func configureClientNetwork(ifaceName, providerHost string) (func(), error) {
	defaultGW, err := linuxGetDefaultGateway()
	if err != nil {
		return nil, err
	}

	if err := linuxSetupRouting(ifaceName, providerHost, defaultGW); err != nil {
		return nil, err
	}

	dnsConfigured := false
	if err := linuxSetupDNS(); err != nil {
		log.Printf("Warning: failed to set DNS automatically: %v", err)
	} else {
		dnsConfigured = true
	}

	cleanup := func() {
		linuxRestoreRouting(ifaceName, providerHost)
		if dnsConfigured {
			linuxRestoreDNS()
		}
	}
	return cleanup, nil
}

func setupRouting(ifaceName, providerHost, defaultGW string) error {
	link, err := netlink.LinkByName(ifaceName)
	if err != nil {
		return fmt.Errorf("could not get interface %s: %w", ifaceName, err)
	}

	gw := net.ParseIP(defaultGW)
	providerIP := net.ParseIP(providerHost)
	if gw == nil || providerIP == nil {
		return fmt.Errorf("invalid IP address for gateway or provider")
	}

	routeToProvider := &netlink.Route{
		Dst: &net.IPNet{IP: providerIP, Mask: net.CIDRMask(32, 32)},
		Gw:  gw,
	}
	if err := netlink.RouteAdd(routeToProvider); err != nil {
		log.Printf("Warning: failed to add route for provider endpoint: %v", err)
	}

	_, dst1, _ := net.ParseCIDR("0.0.0.0/1")
	route1 := &netlink.Route{LinkIndex: link.Attrs().Index, Dst: dst1}
	if err := netlink.RouteAdd(route1); err != nil {
		return fmt.Errorf("failed to set default route part 1: %w", err)
	}

	_, dst2, _ := net.ParseCIDR("128.0.0.0/1")
	route2 := &netlink.Route{LinkIndex: link.Attrs().Index, Dst: dst2}
	if err := netlink.RouteAdd(route2); err != nil {
		_ = netlink.RouteDel(route1)
		return fmt.Errorf("failed to set default route part 2: %w", err)
	}

	return nil
}

func restoreRouting(ifaceName, providerHost string) {
	providerIP := net.ParseIP(providerHost)
	_ = netlink.RouteDel(&netlink.Route{Dst: &net.IPNet{IP: providerIP, Mask: net.CIDRMask(32, 32)}})
	_, dst1, _ := net.ParseCIDR("0.0.0.0/1")
	_, dst2, _ := net.ParseCIDR("128.0.0.0/1")
	link, err := netlink.LinkByName(ifaceName)
	if err == nil {
		_ = netlink.RouteDel(&netlink.Route{LinkIndex: link.Attrs().Index, Dst: dst1})
		_ = netlink.RouteDel(&netlink.Route{LinkIndex: link.Attrs().Index, Dst: dst2})
	}
}

func setupDNS() error {
	content, err := linuxReadFile(resolvConfPath)
	if err != nil {
		return fmt.Errorf("failed to read resolv.conf: %w", err)
	}
	if err := linuxWriteFile(resolvBackupPath, content, 0644); err != nil {
		return fmt.Errorf("failed to backup resolv.conf: %w", err)
	}
	if err := linuxWriteFile(resolvConfPath, []byte(secureDNS), 0644); err != nil {
		return fmt.Errorf("failed to write new resolv.conf: %w", err)
	}
	return nil
}

func restoreDNS() {
	content, err := linuxReadFile(resolvBackupPath)
	if err != nil {
		return
	}
	if err := linuxWriteFile(resolvConfPath, content, 0644); err != nil {
		return
	}
	_ = linuxRemoveFile(resolvBackupPath)
}
