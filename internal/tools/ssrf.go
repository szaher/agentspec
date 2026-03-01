package tools

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

// IsPrivateIP checks if an IP address is in a private/internal range.
// Blocks RFC 1918, RFC 3927 (link-local), loopback, and IPv6 equivalents.
func IsPrivateIP(ip net.IP) bool {
	privateRanges := []struct {
		network *net.IPNet
	}{
		{mustParseCIDR("10.0.0.0/8")},
		{mustParseCIDR("172.16.0.0/12")},
		{mustParseCIDR("192.168.0.0/16")},
		{mustParseCIDR("169.254.0.0/16")}, // link-local
		{mustParseCIDR("127.0.0.0/8")},    // loopback
		{mustParseCIDR("::1/128")},        // IPv6 loopback
		{mustParseCIDR("fc00::/7")},       // IPv6 unique local
		{mustParseCIDR("fe80::/10")},      // IPv6 link-local
	}

	for _, r := range privateRanges {
		if r.network.Contains(ip) {
			return true
		}
	}

	return ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast()
}

func mustParseCIDR(s string) *net.IPNet {
	_, n, err := net.ParseCIDR(s)
	if err != nil {
		panic(fmt.Sprintf("invalid CIDR: %s", s))
	}
	return n
}

// NewSafeTransport returns an http.Transport that validates resolved IPs
// against private/internal ranges before connecting.
// Checks at dial time (not URL parse time) to prevent DNS rebinding.
func NewSafeTransport() *http.Transport {
	return &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, fmt.Errorf("SSRF: invalid address %q: %w", addr, err)
			}

			// Resolve DNS
			ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil {
				return nil, fmt.Errorf("SSRF: DNS resolution failed for %q: %w", host, err)
			}

			// Check all resolved IPs
			for _, ip := range ips {
				if IsPrivateIP(ip.IP) {
					return nil, fmt.Errorf("SSRF: private network access denied for %s (%s)", host, ip.IP.String())
				}
			}

			// Connect to the first resolved IP
			dialer := &net.Dialer{Timeout: 10 * time.Second}
			return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
		},
	}
}
