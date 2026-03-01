package tools

import (
	"net"
	"testing"
)

func TestIsPrivateIP_RFC1918(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want bool
	}{
		// 10.0.0.0/8
		{"10.0.0.1", "10.0.0.1", true},
		{"10.255.255.255", "10.255.255.255", true},
		// 172.16.0.0/12
		{"172.16.0.1", "172.16.0.1", true},
		{"172.31.255.255", "172.31.255.255", true},
		// 192.168.0.0/16
		{"192.168.0.1", "192.168.0.1", true},
		{"192.168.255.255", "192.168.255.255", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ip := net.ParseIP(tc.ip)
			if ip == nil {
				t.Fatalf("failed to parse IP %q", tc.ip)
			}
			got := IsPrivateIP(ip)
			if got != tc.want {
				t.Fatalf("IsPrivateIP(%s) = %v, want %v", tc.ip, got, tc.want)
			}
		})
	}
}

func TestIsPrivateIP_Loopback(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{"IPv4 loopback", "127.0.0.1", true},
		{"IPv4 loopback range", "127.0.0.2", true},
		{"IPv6 loopback", "::1", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ip := net.ParseIP(tc.ip)
			if ip == nil {
				t.Fatalf("failed to parse IP %q", tc.ip)
			}
			got := IsPrivateIP(ip)
			if got != tc.want {
				t.Fatalf("IsPrivateIP(%s) = %v, want %v", tc.ip, got, tc.want)
			}
		})
	}
}

func TestIsPrivateIP_LinkLocal(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{"IPv4 link-local", "169.254.1.1", true},
		{"IPv4 link-local start", "169.254.0.1", true},
		{"IPv6 link-local", "fe80::1", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ip := net.ParseIP(tc.ip)
			if ip == nil {
				t.Fatalf("failed to parse IP %q", tc.ip)
			}
			got := IsPrivateIP(ip)
			if got != tc.want {
				t.Fatalf("IsPrivateIP(%s) = %v, want %v", tc.ip, got, tc.want)
			}
		})
	}
}

func TestIsPrivateIP_IPv6UniqueLocal(t *testing.T) {
	ip := net.ParseIP("fc00::1")
	if ip == nil {
		t.Fatal("failed to parse IP fc00::1")
	}
	if !IsPrivateIP(ip) {
		t.Fatal("IsPrivateIP(fc00::1) = false, want true")
	}
}

func TestIsPrivateIP_PublicAddresses(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{"Google DNS", "8.8.8.8", false},
		{"Cloudflare DNS", "1.1.1.1", false},
		{"TEST-NET-3", "203.0.113.1", false},
		{"Public IPv6", "2001:4860:4860::8888", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ip := net.ParseIP(tc.ip)
			if ip == nil {
				t.Fatalf("failed to parse IP %q", tc.ip)
			}
			got := IsPrivateIP(ip)
			if got != tc.want {
				t.Fatalf("IsPrivateIP(%s) = %v, want %v", tc.ip, got, tc.want)
			}
		})
	}
}

func TestNewSafeTransport(t *testing.T) {
	transport := NewSafeTransport()

	if transport == nil {
		t.Fatal("NewSafeTransport() returned nil")
	}

	if transport.DialContext == nil {
		t.Fatal("expected DialContext to be set on safe transport")
	}
}

func TestIsPrivateIP_AllRangesTableDriven(t *testing.T) {
	// Comprehensive table-driven test covering all categories in a single test
	tests := []struct {
		ip   string
		want bool
		desc string
	}{
		{"10.0.0.1", true, "RFC 1918 class A"},
		{"10.255.255.255", true, "RFC 1918 class A upper"},
		{"172.16.0.1", true, "RFC 1918 class B lower"},
		{"172.31.255.255", true, "RFC 1918 class B upper"},
		{"192.168.0.1", true, "RFC 1918 class C lower"},
		{"192.168.255.255", true, "RFC 1918 class C upper"},
		{"127.0.0.1", true, "IPv4 loopback"},
		{"169.254.1.1", true, "IPv4 link-local"},
		{"::1", true, "IPv6 loopback"},
		{"fc00::1", true, "IPv6 unique local"},
		{"fe80::1", true, "IPv6 link-local"},
		{"8.8.8.8", false, "Google public DNS"},
		{"1.1.1.1", false, "Cloudflare public DNS"},
		{"203.0.113.1", false, "TEST-NET-3 public"},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			ip := net.ParseIP(tc.ip)
			if ip == nil {
				t.Fatalf("failed to parse IP %q", tc.ip)
			}
			got := IsPrivateIP(ip)
			if got != tc.want {
				t.Fatalf("IsPrivateIP(%s) = %v, want %v (%s)", tc.ip, got, tc.want, tc.desc)
			}
		})
	}
}
