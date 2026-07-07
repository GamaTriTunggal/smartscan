package utils

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"syscall"
	"time"
)

// isDisallowedIP reports whether an IP must never be dialed by a server-side
// request to a user-supplied URL (SSRF protection). It blocks loopback, the
// unspecified address, link-local (incl. the 169.254.169.254 cloud-metadata
// endpoint and IPv6 fe80::/10), private ranges (RFC1918 + RFC4193 ULA via
// IsPrivate), and multicast.
func isDisallowedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() || ip.IsMulticast() ||
		ip.IsPrivate() {
		return true
	}
	// Block IPv4 carrier-grade NAT (100.64.0.0/10), which IsPrivate does not cover.
	if ip4 := ip.To4(); ip4 != nil {
		if ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127 {
			return true
		}
	}
	return false
}

// ssrfSafeControl is a net.Dialer Control hook. It is invoked with the CONCRETE
// resolved address (ip:port) right before the socket connects — including for
// each hop of an HTTP redirect — so it closes the DNS-rebinding gap that a
// resolve-then-check approach would leave open.
func ssrfSafeControl(network, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	ip := net.ParseIP(host)
	if ip == nil || isDisallowedIP(ip) {
		return fmt.Errorf("ssrf: refusing to connect to disallowed address %q", address)
	}
	return nil
}

// SafeHTTPClient returns an *http.Client for outbound requests to user-supplied
// URLs. It refuses to connect to internal/loopback/link-local/private addresses
// at dial time and caps redirects. Use this for any request whose destination is
// influenced by tenant/user input (e.g. outbound webhooks).
func SafeHTTPClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 5 * time.Second,
		Control:   ssrfSafeControl,
	}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.DialContext(ctx, network, addr)
		},
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: timeout,
		ExpectContinueTimeout: 1 * time.Second,
		DisableKeepAlives:     true,
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(_ *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return fmt.Errorf("stopped after 3 redirects")
			}
			return nil
		},
	}
}
