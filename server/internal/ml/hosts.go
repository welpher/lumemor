// The download hosts' gate: the one place that decides whether a URL
// host -- the first one or a redirect hop -- may carry model bytes.
//
// The lock discipline the supply chain pins: the sha256 digest is the
// real lock (nothing lands without matching its pinned digest), and
// the host list is the transport gate in front of it -- an allowed
// host list restricts WHERE the bytes may travel from, while the
// digest pins WHAT they must be. A host list entry is either an exact
// hostname or a `*.` wildcard prefix covering a whole domain suffix,
// so a manifest naming `*.hf.co` admits every CDN host a HuggingFace
// redirect can reach without admitting anything else.
//
// An EMPTY list admits every host: the instance has not configured a
// transport gate, and the digest alone guards the bytes. The shipped
// default settings always carry the manifest's own hosts, so
// production never runs with the empty gate by accident -- and the
// tests that stand up local fixture servers keep the list empty (or
// name the fixture) instead of threading a special case through the
// walk.
package ml

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// HostAllowed reports whether host is admitted by the list: an exact
// match or a `*.` wildcard entry whose suffix the host ends with at a
// label boundary (`*.hf.co` admits cdn.hf.co but not hf.co, and never
// evilhf.co). The list carries BARE hostnames -- the settings save
// strips any port -- and the callers pass the URL's hostname alone,
// so the match is a plain hostname comparison.
func HostAllowed(allowed []string, host string) bool {
	if len(allowed) == 0 {
		return true
	}
	host = strings.ToLower(strings.TrimSpace(host))
	for _, entry := range allowed {
		entry = strings.ToLower(strings.TrimSpace(entry))
		if entry == "" {
			continue
		}
		if strings.HasPrefix(entry, "*.") {
			suffix := entry[1:] // ".hf.co"
			if strings.HasSuffix(host, suffix) && len(host) > len(suffix) {
				return true
			}
			continue
		}
		if host == entry {
			return true
		}
	}
	return false
}

// URLHost answers a URL's host, lowercased, or an error for a URL
// without one -- the shape the chain walk validates against the list.
func URLHost(raw string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("ml: parse %s: %w", raw, err)
	}
	host := parsed.Hostname()
	if host == "" {
		return "", fmt.Errorf("ml: %s names no host", raw)
	}
	return strings.ToLower(host), nil
}

// clientWithHostGate wraps an outbound client with a redirect policy
// that validates every hop's host against the list: the first hop is
// the walk's own check, the redirect hops are this policy's. A hop
// outside the list aborts the transfer with the host named; the
// original client's transport rides along (a copy, so the caller's
// client is never mutated).
func clientWithHostGate(client *http.Client, allowed []string) *http.Client {
	gated := *client
	gated.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		host, err := URLHost(req.URL.String())
		if err != nil {
			return fmt.Errorf("host %w", err)
		}
		if !HostAllowed(allowed, host) {
			return fmt.Errorf("redirect host %s is not in the allowed list", host)
		}
		if len(via) >= 10 {
			return errors.New("stopped after 10 redirects")
		}
		return nil
	}
	return &gated
}
