// The deployment's transport policy: the parts of a request that a
// reverse proxy rewrites or hides, resolved from the deployment's own
// knobs rather than guessed from the socket.
//
// The LAN deployment is the baseline and stays first-class: with
// neither knob set, a request's scheme is its own TLS state, the CSRF
// check is host equality, and the client address is the socket peer --
// byte-for-byte the behaviour before the policy existed. A public
// deployment sets one or both knobs:
//
//   - the public origin (scheme + host) makes "the browser used https"
//     answerable even though the proxied request is plain HTTP, and
//     lets the CSRF check accept the address the browser actually
//     typed when the proxy rewrites Host;
//   - the trusted-proxy list makes the forwarded client address and
//     scheme trustworthy for the peers on it -- and only for them, so
//     a direct client cannot forge either header into a fresh
//     rate-limit bucket or a spurious Secure cookie.
package api

import (
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
)

// hstsValue is the HSTS policy the server emits once the request is
// known to be https: one year, deliberately without includeSubDomains
// or preload -- a family instance does not promise its whole domain
// tree, and preload is effectively irreversible.
const hstsValue = "max-age=31536000"

// deployTransport is the resolved transport policy: the public
// origin's host and scheme (both empty/false when unset) and the
// prefixes whose socket peers may assert forwarded facts.
type deployTransport struct {
	// publicHost is the public origin's host[:port]; empty when the
	// deployment names no public origin.
	publicHost string
	// publicHTTPS reports whether the public origin is an https one.
	publicHTTPS bool
	// trusted are the peers allowed to assert X-Forwarded-For and
	// X-Forwarded-Proto; empty trusts the socket peer only.
	trusted []netip.Prefix
}

// newDeployTransport builds the policy from the deployment's values.
// Config loading validates the origin, so a value that parses here is
// already canonical; an unparsable one (a wiring bug) leaves the
// public-origin half off rather than weakening the check silently --
// the unset behaviour is the safe fallback.
func newDeployTransport(publicURL string, trusted []netip.Prefix) deployTransport {
	policy := deployTransport{trusted: trusted}
	if publicURL == "" {
		return policy
	}
	parsed, err := url.Parse(publicURL)
	if err != nil || parsed.Host == "" {
		return policy
	}
	policy.publicHost = parsed.Host
	policy.publicHTTPS = parsed.Scheme == "https"
	return policy
}

// securityHeaders stamps the two response headers every response of
// the server carries: API, pages, static assets and the SPA shell
// alike (NewRouter mounts it at the router root, so every route --
// hand-mounted or catch-all -- passes through it), plus HSTS once the
// request is known to be https. The dynamic CSP arrives with the map
// feature (the CSP must know the tile origins first); CORS stays
// closed by design -- the SPA is same-origin inside the single binary
// and external tools authenticate as bearers, which CORS never
// constrains.
func (t deployTransport) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		if t.requestIsHTTPS(r) {
			w.Header().Set("Strict-Transport-Security", hstsValue)
		}
		next.ServeHTTP(w, r)
	})
}

// checkOrigin implements the CSRF defense of an all-JSON API: a
// state-changing request that declares a foreign Origin is refused
// outright. Requests without an Origin header pass -- they are
// non-browser clients (the mobile app, curl), which are not part of
// the browser CSRF threat model. Reads (GET/HEAD/OPTIONS) are exempt.
func (t deployTransport) checkOrigin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isStateChanging(r.Method) {
			if origin := r.Header.Get("Origin"); origin != "" && !t.sameOrigin(r, origin) {
				http.Error(w, "cross-origin request rejected", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// sameOrigin reports whether the declared Origin matches the request's
// own host -- or the deployment's public origin, which is what a proxy
// that rewrites Host leaves as the browser's truth. Only host and port
// are compared, not the scheme: behind a TLS-terminating reverse proxy
// the forwarded request is plain HTTP while the browser's Origin is
// HTTPS, and that deployment must keep working. A malformed Origin
// (including the literal "null") never matches.
func (t deployTransport) sameOrigin(r *http.Request, origin string) bool {
	originURL, err := url.Parse(origin)
	if err != nil || originURL.Host == "" {
		return false
	}
	if strings.EqualFold(originURL.Host, r.Host) {
		return true
	}
	return t.publicHost != "" && strings.EqualFold(originURL.Host, t.publicHost)
}

// requestIsHTTPS reports whether the browser reached the server over
// TLS: directly (the connection carries TLS), through the deployment's
// https public origin, or through a trusted proxy that says it
// terminated TLS. A forwarded-proto header from an untrusted peer is
// ignored -- anyone can send one.
func (t deployTransport) requestIsHTTPS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	if t.publicHTTPS {
		return true
	}
	return t.peerTrusted(r) && forwardedProto(r) == "https"
}

// remoteIP derives the client address the rate limiter keys on and the
// session records. It is the socket peer unless that peer is a trusted
// proxy; a trusted peer's X-Forwarded-For is then walked from the
// right, and the right-most hop that is not itself a trusted proxy is
// the client (the hops right of it are the proxies that appended their
// own view). An empty, all-trusted or malformed chain falls back to
// the socket peer, which is always honest even when it is coarse.
func (t deployTransport) remoteIP(r *http.Request) string {
	peer := socketPeerHost(r)
	if !t.peerTrusted(r) {
		return peer
	}
	chain, ok := forwardedChain(r)
	if !ok {
		return peer
	}
	for i := len(chain) - 1; i >= 0; i-- {
		if !t.addrTrusted(chain[i]) {
			return chain[i].String()
		}
	}
	return peer
}

// peerTrusted reports whether the request's socket peer is inside the
// trusted-proxy prefixes. No prefixes means no peer is trusted.
func (t deployTransport) peerTrusted(r *http.Request) bool {
	if len(t.trusted) == 0 {
		return false
	}
	addr, ok := socketPeer(r)
	if !ok {
		return false
	}
	return t.addrTrusted(addr)
}

// addrTrusted reports whether addr falls inside one of the trusted
// prefixes. IPv4-mapped addresses are unmapped first so a v4 prefix
// matches a v4-in-v6 peer.
func (t deployTransport) addrTrusted(addr netip.Addr) bool {
	for _, prefix := range t.trusted {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

// socketPeerHost answers the socket peer's host without its port: the
// rate-limit key and the recorded session address carry no port, and a
// malformed RemoteAddr (a bare address, a unix socket) passes through
// as it always has.
func socketPeerHost(r *http.Request) string {
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

// socketPeer answers the socket peer as a comparable address.
func socketPeer(r *http.Request) (netip.Addr, bool) {
	addr, err := netip.ParseAddr(socketPeerHost(r))
	if err != nil {
		return netip.Addr{}, false
	}
	return addr.Unmap(), true
}

// forwardedProto answers the scheme the nearest proxy reports: the
// right-most X-Forwarded-Proto value (each proxy appends its own view,
// so the last one is the hop that handed the request to this socket).
func forwardedProto(r *http.Request) string {
	values := r.Header.Values("X-Forwarded-Proto")
	if len(values) == 0 {
		return ""
	}
	raw := values[len(values)-1]
	if i := strings.LastIndex(raw, ","); i >= 0 {
		raw = raw[i+1:]
	}
	return strings.ToLower(strings.TrimSpace(raw))
}

// forwardedChain parses X-Forwarded-For into the hop list, left to
// right. Absent is an empty chain, and a malformed entry anywhere
// answers false: a header that cannot be read whole is not trusted in
// part.
func forwardedChain(r *http.Request) ([]netip.Addr, bool) {
	values := r.Header.Values("X-Forwarded-For")
	var chain []netip.Addr
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				return nil, false
			}
			addr, ok := parseForwardedAddr(part)
			if !ok {
				return nil, false
			}
			chain = append(chain, addr)
		}
	}
	return chain, true
}

// parseForwardedAddr parses one X-Forwarded-For entry: a bare address,
// a bracketed IPv6 literal, or an address carrying a port (some proxies
// append one).
func parseForwardedAddr(raw string) (netip.Addr, bool) {
	raw = strings.TrimSpace(raw)
	if addr, err := netip.ParseAddr(raw); err == nil {
		return addr.Unmap(), true
	}
	if host, _, err := net.SplitHostPort(raw); err == nil {
		if addr, err := netip.ParseAddr(host); err == nil {
			return addr.Unmap(), true
		}
	}
	if strings.HasPrefix(raw, "[") && strings.HasSuffix(raw, "]") {
		if addr, err := netip.ParseAddr(raw[1 : len(raw)-1]); err == nil {
			return addr.Unmap(), true
		}
	}
	return netip.Addr{}, false
}
