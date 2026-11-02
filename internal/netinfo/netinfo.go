package netinfo

import (
	"fmt"
	"net"
	"strings"
)

// InterfaceAddr is a candidate LAN address.
type InterfaceAddr struct {
	Name string
	IP   net.IP
}

// ListIPv4 returns up, non-loopback IPv4 addresses, including ones on
// adapters we would normally skip (for --list-ifaces).
func ListIPv4() ([]InterfaceAddr, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("list interfaces: %w", err)
	}
	var out []InterfaceAddr
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			ip := ipv4FromAddr(a)
			if ip == nil || !ip.IsGlobalUnicast() {
				continue
			}
			out = append(out, InterfaceAddr{Name: iface.Name, IP: ip})
		}
	}
	return out, nil
}

// SelectLANIP picks a usable LAN IPv4, skipping VPN/Docker/Hyper-V/WSL-style
// adapters. Prefers RFC1918 addresses. Returns an error if none found.
func SelectLANIP() (net.IP, string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, "", fmt.Errorf("list interfaces: %w", err)
	}

	var (
		bestPrivate   net.IP
		bestPrivateIf string
		fallback      net.IP
		fallbackIf    string
	)
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if SkipAdapter(iface.Name) {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			ip := ipv4FromAddr(a)
			if ip == nil || !ip.IsGlobalUnicast() {
				continue
			}
			if IsPrivateLAN(ip) {
				if bestPrivate == nil {
					bestPrivate = ip
					bestPrivateIf = iface.Name
				}
				continue
			}
			if fallback == nil {
				fallback = ip
				fallbackIf = iface.Name
			}
		}
	}
	if bestPrivate != nil {
		return bestPrivate, bestPrivateIf, nil
	}
	if fallback != nil {
		return fallback, fallbackIf, nil
	}
	return nil, "", fmt.Errorf("no LAN IPv4 found (check Wi-Fi, or pass -bind)")
}

// SkipAdapter reports whether an interface name looks like VPN/virtual infra.
func SkipAdapter(name string) bool {
	n := strings.ToLower(name)
	for _, needle := range []string{
		"vpn", "docker", "vethernet", "hyper-v", "wsl", "loopback",
		"virtualbox", "vmware", "tailscale", "bluetooth", "isatap",
		"veth", "br-", "virbr", "tun", "tap", "wg", "zt",
		// Cloudflare WARP uses 172.16.0.0/24-style tunnel IPs that phones cannot reach.
		"cloudflare", "warp",
	} {
		if strings.Contains(n, needle) {
			return true
		}
	}
	return false
}

// IsPrivateLAN reports RFC1918 IPv4.
func IsPrivateLAN(ip net.IP) bool {
	ip4 := ip.To4()
	if ip4 == nil {
		return false
	}
	private := []net.IPNet{
		{IP: net.IPv4(10, 0, 0, 0), Mask: net.CIDRMask(8, 32)},
		{IP: net.IPv4(172, 16, 0, 0), Mask: net.CIDRMask(12, 32)},
		{IP: net.IPv4(192, 168, 0, 0), Mask: net.CIDRMask(16, 32)},
	}
	for i := range private {
		if private[i].Contains(ip4) {
			return true
		}
	}
	return false
}

func ipv4FromAddr(a net.Addr) net.IP {
	switch v := a.(type) {
	case *net.IPNet:
		return v.IP.To4()
	case *net.IPAddr:
		return v.IP.To4()
	default:
		return nil
	}
}
