// The instance's outbound HTTP client: the one seam through which the
// server reaches a meta service (the release feed today; the model
// download's own transport carries its separate, pre-existing
// discipline). It is deliberately narrow:
//
//   - The forward proxy comes from the environment (http_proxy,
//     https_proxy, no_proxy). A deployment that reaches the outside
//     world through a proxy needs no instance setting for it, and a
//     deployment that does not has nothing to configure.
//   - Internal targets are refused: a dial whose resolved address is
//     a loopback, private, link-local, multicast or otherwise
//     non-public range fails before a byte moves. The check happens
//     at dial time, on the resolved IP, so a hostname pointing at an
//     internal address is refused exactly like the literal address.
//   - Redirects are never followed: a 3xx answer is the target's
//     answer and the caller treats it as such.
//   - Every request is bounded by one timeout.
//
// The photo data path has no business here: this client carries no
// upload and no client-supplied body.
package updatecheck

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"syscall"
	"time"
)

// RequestTimeout bounds one outbound request end to end.
const RequestTimeout = 10 * time.Second

// ErrInternalTarget is the sentinel the dial guard answers: the
// address resolved to a range this client refuses to reach.
var ErrInternalTarget = errors.New("updatecheck: the address resolves to an internal range")

// ClientPolicy carries the client's reachability rules.
type ClientPolicy struct {
	// RefuseInternal refuses dials whose address is not a public
	// unicast range. Production pins true; tests that stand up a
	// local fixture server pin false, so the refusal itself stays a
	// pinned behavior of the same code path.
	RefuseInternal bool
}

// NewClient builds the instance's outbound client under the policy.
func NewClient(policy ClientPolicy) *http.Client {
	dialer := &net.Dialer{Timeout: RequestTimeout}
	if policy.RefuseInternal {
		dialer.Control = func(_ string, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return fmt.Errorf("updatecheck: parse the dial address %q: %w", address, err)
			}
			ip := net.ParseIP(host)
			if ip == nil {
				return fmt.Errorf("updatecheck: dial address %q is not an IP", address)
			}
			if !publicIP(ip) {
				return fmt.Errorf("%w: %s", ErrInternalTarget, address)
			}
			return nil
		}
	}
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		TLSHandshakeTimeout:   RequestTimeout,
		ResponseHeaderTimeout: RequestTimeout,
		ForceAttemptHTTP2:     true,
	}
	return &http.Client{
		Timeout:   RequestTimeout,
		Transport: transport,
		// A redirect is a different target than the one the
		// administrator configured; the probe never follows one.
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// publicIP reports whether ip is a public unicast address -- the
// complement of every range a self-hosted instance must never be
// steered at by a configured URL.
func publicIP(ip net.IP) bool {
	if ip == nil || ip.IsUnspecified() || ip.IsLoopback() ||
		ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() || ip.IsInterfaceLocalMulticast() {
		return false
	}
	if v4 := ip.To4(); v4 != nil {
		// 100.64.0.0/10: the carrier-grade NAT range, routable only
		// inside an operator's network.
		if v4[0] == 100 && v4[1] >= 64 && v4[1] <= 127 {
			return false
		}
		// 192.0.0.0/24: IETF protocol assignments.
		if v4[0] == 192 && v4[1] == 0 && v4[2] == 0 {
			return false
		}
		return true
	}
	// fc00::/7: the IPv6 unique-local range.
	if len(ip) == net.IPv6len && ip[0]&0xfe == 0xfc {
		return false
	}
	return true
}

// validEndpoint reports whether raw is an absolute http(s) URL with a
// host -- the only shape the probe accepts.
func validEndpoint(raw string) (*url.URL, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("updatecheck: parse the endpoint: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("updatecheck: the endpoint scheme %q is not http or https", parsed.Scheme)
	}
	if parsed.Host == "" {
		return nil, errors.New("updatecheck: the endpoint carries no host")
	}
	return parsed, nil
}
