// The instance's outbound HTTP client: the one seam through which the
// server reaches a meta service (the release feed today; the model
// download's own transport carries its separate, pre-existing
// discipline). It is deliberately narrow:
//
//   - The forward proxy comes from the environment (http_proxy,
//     https_proxy, no_proxy). A deployment that reaches the outside
//     world through a proxy needs no instance setting for it, and a
//     deployment that does not has nothing to configure.
//   - Internal targets are refused. A direct dial refuses the address
//     its name resolved to at connect time (the rebinding-proof
//     check). Behind a proxy the dial address is the PROXY's -- a
//     deployment's own loopback or LAN proxy is a legitimate
//     deployment choice -- so the endpoint's own hostname is resolved
//     and classified instead, and every answer must be public before
//     the request leaves.
//   - The proxy's residual window, stated: once a request is handed to
//     a proxy, that proxy fetches the endpoint on the instance's
//     behalf, and this client cannot see which address it reached. A
//     proxy that resolves the endpoint differently (an intercepting
//     resolver, a proxy-side hostname mapping) can therefore reach an
//     internal service the instance's own resolution did not name.
//     That is inherent to proxying, not a bug to close here: the
//     endpoint-side resolution check is what keeps the instance from
//     being STEERED inward by its own configuration, which is the
//     danger this guard exists for. A deployment that must not reach
//     anything internal does not configure a proxy through which it
//     can be reached.
//   - Redirects are never followed: a 3xx answer is the target's
//     answer and the caller treats it as such.
//   - Every request is bounded by one timeout.
//
// The photo data path has no business here: this client carries no
// upload and no client-supplied body.
package updatecheck

import (
	"context"
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

// ErrInternalTarget is the sentinel the address guard answers: the
// address -- or, behind a proxy, the endpoint's name -- resolves to a
// range this client refuses to reach.
var ErrInternalTarget = errors.New("updatecheck: the address resolves to an internal range")

// ClientPolicy carries the client's reachability rules.
type ClientPolicy struct {
	// RefuseInternal refuses targets whose address is not a public
	// unicast range. Production pins true; tests that stand up a
	// local fixture server pin false, so the refusal itself stays a
	// pinned behavior of the same code path.
	RefuseInternal bool
	// Proxy resolves the forward proxy for a request; nil falls back
	// to http.ProxyFromEnvironment (the deployment's own https_proxy,
	// http_proxy and no_proxy). The tests that prove the proxied path
	// pin their own, so the process environment stays untouched.
	Proxy func(*http.Request) (*url.URL, error)
}

// proxiedDial marks a request whose dial goes to the proxy rather than
// to the endpoint: the dial guard must leave that address alone.
type proxiedDial struct{}

// NewClient builds the instance's outbound client under the policy.
func NewClient(policy ClientPolicy) *http.Client {
	proxy := policy.Proxy
	if proxy == nil {
		proxy = http.ProxyFromEnvironment
	}
	dialer := &net.Dialer{Timeout: RequestTimeout}
	if policy.RefuseInternal {
		dialer.ControlContext = func(ctx context.Context, _, address string, _ syscall.RawConn) error {
			if proxied, _ := ctx.Value(proxiedDial{}).(bool); proxied {
				// The dial behind this request is the proxy's, not
				// the endpoint's: the endpoint's own resolution was
				// classified before the request left (see
				// guardedTransport).
				return nil
			}
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
		Proxy:                 proxy,
		DialContext:           dialer.DialContext,
		TLSHandshakeTimeout:   RequestTimeout,
		ResponseHeaderTimeout: RequestTimeout,
		ForceAttemptHTTP2:     true,
	}
	return &http.Client{
		Timeout: RequestTimeout,
		Transport: &guardedTransport{
			proxy:  proxy,
			base:   transport,
			refuse: policy.RefuseInternal,
		},
		// A redirect is a different target than the one the
		// administrator configured; the probe never follows one.
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// guardedTransport picks the address classification that matches how
// the request will travel: a direct dial hands the resolved address to
// the dial guard, a proxied request classifies the endpoint's own host
// first (the dial guard then sees only the proxy's address, which is
// not the endpoint).
type guardedTransport struct {
	proxy  func(*http.Request) (*url.URL, error)
	base   *http.Transport
	refuse bool
}

// RoundTrip classifies the endpoint when a proxy is in effect and
// marks the dial so the dial guard leaves the proxy's address alone.
func (t *guardedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if !t.refuse {
		return t.base.RoundTrip(req)
	}
	proxyURL, err := t.proxy(req)
	if err != nil {
		return nil, fmt.Errorf("updatecheck: resolve the proxy for the endpoint: %w", err)
	}
	if proxyURL == nil {
		// A direct dial: the dial guard classifies the address the
		// name resolved to at connect time, which is the
		// rebinding-proof check.
		return t.base.RoundTrip(req)
	}
	if err := refuseInternalHost(req.Context(), req.URL); err != nil {
		return nil, err
	}
	ctx := context.WithValue(req.Context(), proxiedDial{}, true)
	return t.base.RoundTrip(req.WithContext(ctx))
}

// refuseInternalHost runs the range classification on the endpoint's
// own host: every address the name resolves to must be public. It is
// the pre-resolution guard a proxied request gets in place of the
// dial-time one, and it fails closed -- a name this instance cannot
// resolve is not a name it can vouch for.
func refuseInternalHost(ctx context.Context, endpoint *url.URL) error {
	host := endpoint.Hostname()
	addresses, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
	if err != nil {
		return fmt.Errorf("updatecheck: resolve the endpoint host %q: %w", host, err)
	}
	if len(addresses) == 0 {
		return fmt.Errorf("updatecheck: the endpoint host %q resolves to no address", host)
	}
	for _, address := range addresses {
		if !publicIP(address) {
			return fmt.Errorf("%w: the endpoint host %q resolves to %s", ErrInternalTarget, host, address)
		}
	}
	return nil
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
