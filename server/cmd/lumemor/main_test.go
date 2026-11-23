// The HTTP server's transport timeout test: the bounds are asserted on
// the constructed server, so a regression is caught without spending
// two hours proving a slow request gets cut.
package main

import (
	"net/http"
	"testing"
	"time"
)

// TestNewHTTPServerTimeouts pins the transport bounds and the one
// deliberate absence: a ReadHeaderTimeout guards header dribbling, an
// IdleTimeout reclaims unused keep-alive connections, the ReadTimeout
// is generous enough for a 2 GiB upload, and WriteTimeout stays unset
// so a large media download is never cut mid-stream.
func TestNewHTTPServerTimeouts(t *testing.T) {
	var handler http.Handler = http.NewServeMux()
	srv := newHTTPServer(":8244", handler)

	if srv.Addr != ":8244" {
		t.Errorf("Addr = %q, want %q", srv.Addr, ":8244")
	}
	if srv.Handler == nil {
		t.Error("Handler = nil, want the router")
	}
	if srv.ReadHeaderTimeout != 10*time.Second {
		t.Errorf("ReadHeaderTimeout = %v, want %v", srv.ReadHeaderTimeout, 10*time.Second)
	}
	if srv.IdleTimeout != 120*time.Second {
		t.Errorf("IdleTimeout = %v, want %v", srv.IdleTimeout, 120*time.Second)
	}
	if srv.ReadTimeout != 2*time.Hour {
		t.Errorf("ReadTimeout = %v, want %v", srv.ReadTimeout, 2*time.Hour)
	}
	if srv.WriteTimeout != 0 {
		t.Errorf("WriteTimeout = %v, want 0 (unset: a large download must never be cut)", srv.WriteTimeout)
	}
}
