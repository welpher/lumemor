#!/usr/bin/env python3
"""The offline model-source mirror of scripts/acceptance_m3a.sh.

The M3a acceptance's offline download mode reconstructs the Hugging
Face URL path structure under tools/models/cache/offline-root/ and
serves it over HTTPS from a container on the compose network: the
stack's LUMEMOR_MODELS_HF_MIRROR points the server's download service
at this mirror (the rewrite replaces the huggingface.co host name but
keeps the https scheme, so the local source must speak TLS), and the
self-signed certificate's CA is handed to the server container
through SSL_CERT_FILE (Go's crypto/x509 honors it on Linux).

Usage: offline_mirror_server.py ROOT PORT CERT KEY

ROOT is the served directory tree (the offline root). The handler is
the standard library's SimpleHTTPRequestHandler over ThreadingHTTPServer
with the listening socket wrapped in a TLS context; nothing here knows
anything about the models -- it serves files.
"""

import http.server
import ssl
import sys


def main() -> None:
    if len(sys.argv) != 5:
        sys.exit("usage: offline_mirror_server.py ROOT PORT CERT KEY")
    root, port, cert, key = sys.argv[1], int(sys.argv[2]), sys.argv[3], sys.argv[4]

    context = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
    context.load_cert_chain(cert, key)

    class Handler(http.server.SimpleHTTPRequestHandler):
        def __init__(self, *args, **kwargs):
            super().__init__(*args, directory=root, **kwargs)

    server = http.server.ThreadingHTTPServer(("0.0.0.0", port), Handler)
    server.socket = context.wrap_socket(server.socket, server_side=True)
    server.serve_forever()


if __name__ == "__main__":
    main()
