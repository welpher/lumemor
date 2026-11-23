// The deployment transport policy's tests: the scheme determination
// behind the Secure cookie and HSTS, the CSRF host set, and the client
// address. The baseline case of every table is the unset deployment --
// the direct/LAN instance must answer exactly as it always has.
package api

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
)

// mustPrefixes parses a test's CIDR list.
func mustPrefixes(t *testing.T, raw ...string) []netip.Prefix {
	t.Helper()
	prefixes := make([]netip.Prefix, 0, len(raw))
	for _, entry := range raw {
		prefix, err := netip.ParsePrefix(entry)
		if err != nil {
			t.Fatalf("parse prefix %q: %v", entry, err)
		}
		prefixes = append(prefixes, prefix.Masked())
	}
	return prefixes
}

// transportRequest builds a request as a proxy would hand it over: a
// socket peer, optional forwarded headers, and optionally TLS.
func transportRequest(peer string, tlsConn bool) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = peer
	if tlsConn {
		req.TLS = &tls.ConnectionState{}
	}
	return req
}

// TestDeployTransportHTTPSDetermination pins the Secure/HSTS
// determination: the request's own TLS state, the https public origin,
// or a trusted proxy's X-Forwarded-Proto: https -- and nothing else. An
// untrusted peer's forwarded scheme is ignored, and the unset
// deployment keeps today's r.TLS answer.
func TestDeployTransportHTTPSDetermination(t *testing.T) {
	cases := []struct {
		name     string
		public   string
		trusted  []string
		peer     string
		tlsConn  bool
		proto    string
		wantHTTP bool
	}{
		{
			name: "unset plain HTTP is not https",
			peer: "192.168.1.20:41000",
		},
		{
			name:     "unset TLS is https",
			peer:     "192.168.1.20:41000",
			tlsConn:  true,
			wantHTTP: true,
		},
		{
			name:     "an https public origin makes a proxied request https",
			public:   "https://photos.example.com",
			peer:     "10.0.0.5:41000",
			wantHTTP: true,
		},
		{
			name:   "an http public origin stays plain",
			public: "http://photos.example.com",
			peer:   "10.0.0.5:41000",
		},
		{
			name:     "a trusted proxy's https report counts",
			trusted:  []string{"10.0.0.0/8"},
			peer:     "10.0.0.5:41000",
			proto:    "https",
			wantHTTP: true,
		},
		{
			name:    "a trusted proxy's http report stays plain",
			trusted: []string{"10.0.0.0/8"},
			peer:    "10.0.0.5:41000",
			proto:   "http",
		},
		{
			name:    "an untrusted peer's https report is ignored",
			trusted: []string{"10.0.0.0/8"},
			peer:    "192.168.1.20:41000",
			proto:   "https",
		},
		{
			name:     "an untrusted peer cannot spoof the report",
			peer:     "192.168.1.20:41000",
			proto:    "https",
			wantHTTP: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			policy := newDeployTransport(tc.public, mustPrefixes(t, tc.trusted...))
			req := transportRequest(tc.peer, tc.tlsConn)
			if tc.proto != "" {
				req.Header.Set("X-Forwarded-Proto", tc.proto)
			}
			if got := policy.requestIsHTTPS(req); got != tc.wantHTTP {
				t.Errorf("requestIsHTTPS() = %v, want %v", got, tc.wantHTTP)
			}
		})
	}
}

// TestDeployTransportHSTSHeader pins the wire shape: HSTS appears on
// the response exactly when the request is https, with one year and
// neither includeSubDomains nor preload -- and the unset deployment
// emits none.
func TestDeployTransportHSTSHeader(t *testing.T) {
	cases := []struct {
		name    string
		public  string
		trusted []string
		peer    string
		proto   string
		want    string
	}{
		{name: "unset emits no HSTS", peer: "192.168.1.20:41000", want: ""},
		{
			name:   "an https public origin emits HSTS",
			public: "https://photos.example.com",
			peer:   "10.0.0.5:41000",
			want:   "max-age=31536000",
		},
		{
			name:    "a trusted proxy's https report emits HSTS",
			trusted: []string{"10.0.0.0/8"},
			peer:    "10.0.0.5:41000",
			proto:   "https",
			want:    "max-age=31536000",
		},
		{
			name:  "an untrusted peer's report emits none",
			peer:  "192.168.1.20:41000",
			proto: "https",
			want:  "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			router := NewRouter(&Deps{
				Version:        testVersion,
				PublicURL:      tc.public,
				TrustedProxies: mustPrefixes(t, tc.trusted...),
			})
			req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
			req.RemoteAddr = tc.peer
			if tc.proto != "" {
				req.Header.Set("X-Forwarded-Proto", tc.proto)
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if got := rec.Header().Get("Strict-Transport-Security"); got != tc.want {
				t.Errorf("Strict-Transport-Security = %q, want %q", got, tc.want)
			}
			// The two standing headers ride every response either way.
			if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
				t.Errorf("X-Content-Type-Options = %q, want nosniff", got)
			}
		})
	}
}

// TestDeployTransportSameOrigin pins the CSRF host set: the request's
// own host always matches, a malformed or "null" Origin never does,
// and a configured public origin matches even when the proxy rewrote
// Host to something else.
func TestDeployTransportSameOrigin(t *testing.T) {
	cases := []struct {
		name   string
		public string
		host   string
		origin string
		want   bool
	}{
		{name: "own host", host: "photos.example.com", origin: "https://photos.example.com", want: true},
		{name: "own host with port", host: "photos.example.com:8244", origin: "http://photos.example.com:8244", want: true},
		{name: "foreign host under the unset deployment", host: "photos.example.com", origin: "https://evil.example"},
		{name: "null origin never matches", host: "photos.example.com", origin: "null"},
		{name: "malformed origin never matches", host: "photos.example.com", origin: "://nope"},
		{
			name:   "public origin survives a rewritten Host",
			public: "https://photos.example.com",
			host:   "lumemor:8244",
			origin: "https://photos.example.com",
			want:   true,
		},
		{
			name:   "a foreign origin still does not match",
			public: "https://photos.example.com",
			host:   "lumemor:8244",
			origin: "https://evil.example",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			policy := newDeployTransport(tc.public, nil)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
			req.Host = tc.host
			if got := policy.sameOrigin(req, tc.origin); got != tc.want {
				t.Errorf("sameOrigin(%q) with Host %q = %v, want %v", tc.origin, tc.host, got, tc.want)
			}
		})
	}
}

// TestDeployTransportRemoteIP pins the derived client address: the
// socket peer is the answer unless a trusted proxy forwards the chain,
// in which case the right-most non-trusted hop is the client. An
// untrusted peer's header is ignored outright (a forged address must
// not buy a fresh rate-limit bucket), a malformed header falls back to
// the peer, and v6 prefixes work like v4 ones.
func TestDeployTransportRemoteIP(t *testing.T) {
	cases := []struct {
		name    string
		trusted []string
		peer    string
		xff     []string
		want    string
	}{
		{
			name: "unset uses the socket peer",
			peer: "192.168.1.20:41000",
			xff:  []string{"203.0.113.9"},
			want: "192.168.1.20",
		},
		{
			name:    "an untrusted peer's chain is ignored",
			trusted: []string{"10.0.0.0/8"},
			peer:    "192.168.1.20:41000",
			xff:     []string{"203.0.113.9"},
			want:    "192.168.1.20",
		},
		{
			name:    "a trusted proxy's single hop is the client",
			trusted: []string{"10.0.0.0/8"},
			peer:    "10.0.0.5:41000",
			xff:     []string{"203.0.113.9"},
			want:    "203.0.113.9",
		},
		{
			name:    "a chained header resolves the right-most non-trusted hop",
			trusted: []string{"10.0.0.0/8"},
			peer:    "10.0.0.5:41000",
			xff:     []string{"198.51.100.7, 10.0.0.6, 10.0.0.4"},
			want:    "198.51.100.7",
		},
		{
			name:    "a chain of two client hops answers the nearest one",
			trusted: []string{"10.0.0.0/8"},
			peer:    "10.0.0.5:41000",
			xff:     []string{"198.51.100.7, 203.0.113.9"},
			want:    "203.0.113.9",
		},
		{
			name:    "repeated header lines form one chain",
			trusted: []string{"10.0.0.0/8"},
			peer:    "10.0.0.5:41000",
			xff:     []string{"198.51.100.7", "10.0.0.4"},
			want:    "198.51.100.7",
		},
		{
			name:    "a hop with a port is parsed",
			trusted: []string{"10.0.0.0/8"},
			peer:    "10.0.0.5:41000",
			xff:     []string{"203.0.113.9:5555"},
			want:    "203.0.113.9",
		},
		{
			name:    "a malformed chain falls back to the peer",
			trusted: []string{"10.0.0.0/8"},
			peer:    "10.0.0.5:41000",
			xff:     []string{"203.0.113.9, not-an-address"},
			want:    "10.0.0.5",
		},
		{
			name:    "an empty chain falls back to the peer",
			trusted: []string{"10.0.0.0/8"},
			peer:    "10.0.0.5:41000",
			xff:     []string{""},
			want:    "10.0.0.5",
		},
		{
			name:    "an all-trusted chain falls back to the peer",
			trusted: []string{"10.0.0.0/8"},
			peer:    "10.0.0.5:41000",
			xff:     []string{"10.0.0.4, 10.0.0.6"},
			want:    "10.0.0.5",
		},
		{
			name:    "a v6 trusted peer resolves a v6 client",
			trusted: []string{"2001:db8::/32"},
			peer:    "[2001:db8::5]:41000",
			xff:     []string{"2001:db9:1::9, 2001:db8::4"},
			want:    "2001:db9:1::9",
		},
		{
			name:    "a v4-mapped peer matches its v4 prefix",
			trusted: []string{"10.0.0.0/8"},
			peer:    "[::ffff:10.0.0.5]:41000",
			xff:     []string{"203.0.113.9"},
			want:    "203.0.113.9",
		},
		{
			name:    "a v6 client through a v6 proxy survives",
			trusted: []string{"2001:db8::/32"},
			peer:    "[2001:db8::5]:41000",
			xff:     []string{"[2001:db9:2::7]"},
			want:    "2001:db9:2::7",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			policy := newDeployTransport("", mustPrefixes(t, tc.trusted...))
			req := transportRequest(tc.peer, false)
			for _, value := range tc.xff {
				req.Header.Add("X-Forwarded-For", value)
			}
			if got := policy.remoteIP(req); got != tc.want {
				t.Errorf("remoteIP() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestDeployTransportSameOriginThroughRouter pins the rewrite case end
// to end: a proxied POST whose Host the proxy rewrote passes the origin
// check when the Origin is the public origin, and a foreign Origin is
// still refused.
func TestDeployTransportSameOriginThroughRouter(t *testing.T) {
	router := NewRouter(&Deps{Version: testVersion, PublicURL: "https://photos.example.com"})
	post := func(origin string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
		req.Host = "lumemor:8244"
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", origin)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	// Past the origin check means the handler answered (an empty body
	// is a 400 on its own), not the check's 403.
	if rec := post("https://photos.example.com"); rec.Code == http.StatusForbidden {
		t.Errorf("public-origin POST: status = %d, want the origin check to pass", rec.Code)
	}
	if rec := post("https://evil.example"); rec.Code != http.StatusForbidden {
		t.Errorf("foreign-origin POST: status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}
