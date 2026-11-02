package netinfo

import (
	"net"
	"testing"
)

func TestSkipAdapter(t *testing.T) {
	cases := map[string]bool{
		"Wi-Fi":                    false,
		"Ethernet":                 false,
		"eth0":                     false,
		"Tailscale":                true,
		"vEthernet (WSL)":          true,
		"Hyper-V Virtual Ethernet": true,
		"Docker Desktop":           true,
		"VPN Connection":           true,
		"VMware Network Adapter":   true,
		"bluetooth":                true,
		"WSL":                      true,
		"CloudflareWARP":           true,
		"Cloudflare WARP":          true,
	}
	for name, want := range cases {
		if got := SkipAdapter(name); got != want {
			t.Errorf("SkipAdapter(%q)=%v want %v", name, got, want)
		}
	}
}

func TestIsPrivateLAN(t *testing.T) {
	priv := []string{"10.0.0.1", "172.16.5.5", "172.31.255.1", "192.168.1.10"}
	for _, s := range priv {
		if !IsPrivateLAN(net.ParseIP(s)) {
			t.Errorf("%s should be private", s)
		}
	}
	pub := []string{"8.8.8.8", "1.1.1.1", "172.15.0.1", "172.32.0.1"}
	for _, s := range pub {
		if IsPrivateLAN(net.ParseIP(s)) {
			t.Errorf("%s should not be private", s)
		}
	}
}

func TestSelectLANIPSmoke(t *testing.T) {
	// On CI/dev machines this may succeed or fail; just ensure no panic
	// and that a returned IP is not unspecified/loopback.
	ip, name, err := SelectLANIP()
	if err != nil {
		t.Logf("no LAN IP in this environment: %v", err)
		return
	}
	if ip == nil || ip.IsUnspecified() || ip.IsLoopback() {
		t.Fatalf("bad IP %v on %s", ip, name)
	}
	if SkipAdapter(name) {
		t.Fatalf("selected skipped adapter %q", name)
	}
}
