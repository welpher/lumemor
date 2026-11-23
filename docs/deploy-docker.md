# Deploying with the published image

Two deployment paths exist:

1. **Pull the published image** (below) — no source checkout, no build
   toolchain. The image lives at `ghcr.io/welpher/lumemor`.
2. **Build from source** — `git clone` + `docker compose up --build`
   (the repository's `docker-compose.yml` carries both paths: it
   references the image and the build).

## The pull path

### One-time: registry login

The repository (and therefore the package) is private, so every
machine that pulls needs a personal access token with the
**`read:packages`** scope:

```sh
docker login ghcr.io -u <github-username>
# paste the token as the password
```

Create the token at GitHub → Settings → Developer settings → Personal
access tokens; `read:packages` alone is enough for pulling.

### The compose file

Save as `docker-compose.yml` on the host, then `docker compose up -d`:

```yaml
services:
  db:
    image: pgvector/pgvector:pg17
    environment:
      # CHANGE THIS: a long random string. The database listens only
      # inside the compose network, but a default password in a file
      # you copied from the internet is still a bad habit.
      POSTGRES_USER: lumemor
      POSTGRES_PASSWORD: <change-me>
      POSTGRES_DB: lumemor
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD", "pg_isready", "-U", "lumemor"]
      interval: 5s
      timeout: 3s
      retries: 12
      start_period: 5s

  lumemor:
    image: ghcr.io/welpher/lumemor:latest   # or pin: ghcr.io/welpher/lumemor:v0.1.0-alpha.1
    ports:
      - "8244:8244"
    volumes:
      # The single storage root: photos, derivatives and models live
      # here. BACK THIS UP. On Linux the first start needs the
      # ownership preparation below.
      - ./lumemor:/var/lib/lumemor
    environment:
      LUMEMOR_DATABASE_URL: "postgres://lumemor:<change-me>@db:5432/lumemor?sslmode=disable"
    depends_on:
      db:
        condition: service_healthy

volumes:
  pgdata:
```

### Linux only: prepare the storage directory once

The server container runs as the fixed uid 10001; Docker creates a
missing bind-mount source as root, which the server could not write
into. After the first `mkdir -p ./lumemor`:

```sh
docker run --rm -u 0 -v ./lumemor:/var/lib/lumemor \
    ghcr.io/welpher/lumemor:latest chown -R 10001:10001 /var/lib/lumemor
```

(macOS and rootless-docker runtimes map the mount permissively and
need nothing.)

### First run

Open `http://<host>:8244` — the first-run wizard creates the
administrator account, and the first login lands on the model
management page: downloading the default AI model set (~350 MB) is one
click. Restricted networks set the HuggingFace domain mirror first:

```yaml
    environment:
      LUMEMOR_MODELS_HF_MIRROR: hf-mirror.com   # empty = direct upstreams
```

## Following your folders (directory albums)

A directory album points at a folder on the host, and the server keeps
the album's wall equal to that folder with no manual rescan: adds
appear, removals leave, re-saved files update. The mechanism is a
periodic directory diff (path, size, mtime against the last scan), so
it works on every platform, including the mounts that carry no
filesystem events into the container.

### The activity gate: the photo disk sleeps at night

The periodic walk is the only thing that touches the photo disk when
nobody is using the server, so it follows the household's attention:

- **Somebody is using the instance** (a session was active within the
  last five minutes) — the diff runs on its cadence: every 60 seconds
  unless `LUMEMOR_FOLDER_SYNC_INTERVAL` sets another window.
- **Nobody is** — the diff runs once a day, in the first hour after
  local midnight, and the disk stays quiet the rest of the time.
- **Someone signs in** — the first tick afterwards sweeps immediately
  (within one cadence window), so nothing waits for the night.

"Local midnight" is the server process's own time zone, which the
container reads from `TZ`. Set it to where the house is:

```yaml
    environment:
      TZ: Asia/Shanghai      # any IANA zone name
```

Unset or empty means **UTC**, so a household east of Greenwich that
never sets `TZ` gets its nightly sweep at 08:00 local — harmless, but
not the "disk sleeps while we sleep" the gate is for.

`LUMEMOR_FOLDER_SYNC_INTERVAL` still tunes the active cadence (a Go
duration such as `90s`), and `0` or `off` turns the folder sync off
whole: no diff, no watcher.

### The watcher accelerator

On a native Linux host with the folders on a local disk, the server
also watches each album's folder and asks for a diff about a second
and a half after it changes — an active house sees a dropped photo in
seconds instead of one cadence window. The watcher only says "look at
this folder again"; the same stamp diff decides, so a temp file, a
touch without a real change, or a re-save with an unchanged identity
still lands nothing.

It is on by default. `LUMEMOR_FOLDER_WATCH=0` (or `off`) disables it
and leaves the gated diff alone:

```yaml
    environment:
      LUMEMOR_FOLDER_WATCH: "0"
```

**Honest caveats — none of them a correctness problem:**

- **Docker Desktop (macOS/Windows) and network shares** do not carry
  the host's filesystem events into the container. The watcher then
  never fires; the periodic diff is the coverage. That is a latency
  difference (up to one cadence window), not a missed change.
- **Linux inotify has a watch budget**, one watch per directory, and
  a big family tree can exceed the default (often 8192). When the
  table fills, the server logs one line and continues with the
  periodic diff — again a latency difference, never a lost change.
  Raise the limit and persist it if you want the accelerator at
  scale:

  ```sh
  sudo sysctl -w fs.inotify.max_user_watches=524288
  echo 'fs.inotify.max_user_watches=524288' | sudo tee /etc/sysctl.d/99-lumemor.conf
  ```

- The watcher is **strictly read-only** on your folders: it never
  writes a probe file or a marker into them. Whether a mount carries
  events is not probed either — the diff covers the folders either
  way.

## Before exposing it beyond your LAN

Port 8244 speaks **plain HTTP**. The wizard window (before the first
administrator exists) is unauthenticated by design and binds all
interfaces so family devices can reach it from the LAN. For anything
wider than the LAN: put a reverse proxy with TLS in front (Caddy/nginx
or a tunnels solution), and complete the wizard before the port is
reachable.

### The reverse proxy setup

TLS terminates at the proxy, so the server sees a plain-HTTP request
and cannot tell what the browser used. Two deployment-level variables
tell it, and neither is an instance setting (the network topology
belongs to the deployment, not to the family's settings page):

```yaml
    environment:
      LUMEMOR_PUBLIC_URL: "https://photos.example.com"
      LUMEMOR_TRUSTED_PROXIES: "172.18.0.0/16"
```

- **`LUMEMOR_PUBLIC_URL`** — the origin browsers type (scheme and host
  only, no path). When it is an `https` origin the session cookie gets
  the `Secure` attribute, every response carries
  `Strict-Transport-Security: max-age=31536000` (no `preload`, no
  `includeSubDomains`), and the CSRF check accepts that origin even
  when the proxy rewrites the `Host` header. A malformed value (a path,
  a query, a non-http scheme) stops the boot and names the variable.
- **`LUMEMOR_TRUSTED_PROXIES`** — the proxy's own addresses as a
  comma-separated CIDR list (IPv4 and IPv6; a bare address names one
  host). A request whose socket peer is on the list has its client
  address taken from `X-Forwarded-For` (that address keys the login
  rate limit and lands in the signed-in-devices list) and its
  `X-Forwarded-Proto: https` believed for the cookie and HSTS
  decisions. A peer outside the list has its forwarded headers ignored,
  so a direct client cannot forge either one. A malformed entry stops
  the boot and names the variable.

> **List only the proxy's own address.** Every peer inside the prefix
> may assert `X-Forwarded-For` and `X-Forwarded-Proto`, and the client
> address walk steps past trusted hops to the right: a prefix wide
> enough to cover real clients (the family LAN, `0.0.0.0/0`) lets a
> host inside it forge its client address and escape the login
> lockout.

With `LUMEMOR_PUBLIC_URL` set, the trusted-proxy list is optional (the
public origin already answers the scheme question); without it, the
list alone is enough when the proxy sends `X-Forwarded-Proto`. Set both
if the proxy also forwards the client address and you want real
per-device rate limiting and a truthful device-list address.

A Caddy that terminates TLS itself (the shortest correct setup):

```caddyfile
photos.example.com {
    reverse_proxy lumemor:8244
}
```

Caddy obtains and renews the certificate and sends `Host`,
`X-Forwarded-For` and `X-Forwarded-Proto` on its own; add its compose
address to `LUMEMOR_TRUSTED_PROXIES` if you also want the client
addresses.

An nginx front for a certificate you already hold:

```nginx
server {
    listen 443 ssl;
    server_name photos.example.com;

    ssl_certificate     /etc/letsencrypt/live/photos.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/photos.example.com/privkey.pem;

    # The upload ceiling: the server's per-file limit (2 GiB by
    # default) must fit through the proxy, and uploads stream
    # instead of being buffered to disk first.
    client_max_body_size 2g;
    proxy_request_buffering off;

    location / {
        proxy_pass http://127.0.0.1:8244;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # The activity stream (SSE) and media downloads stream
        # through; buffering them would stall both.
        proxy_buffering off;
        proxy_read_timeout 2h;
    }
}
```

**HSTS is sticky.** Once a browser has seen
`Strict-Transport-Security` for the host it refuses plain HTTP to it
for a year. Only set `LUMEMOR_PUBLIC_URL` to an `https` origin once TLS
at the proxy actually works. Serve a port-80 name that redirects to
https (`return 301 https://$host$request_uri`): HSTS only starts
protecting after the first https response, so the redirect covers the
first visit.

**An https public origin signs browsers in over https only.** The
session cookie carries `Secure`, so the same browser reaching the
instance by LAN IP or plain http will not be signed in. That is the
point of the setting -- keep the LAN address out of day-to-day use once
the public deployment is live.

**With neither variable set nothing changes.** The LAN deployment is
first-class: no `Secure` cookie, no HSTS, the request's own connection
decides, the CSRF check compares `Host`, and every browser behind a
proxy shares one rate-limit bucket while the device list shows the
proxy's address. That is the same behaviour as before these variables
existed.

## Data safety

Everything durable lives in `./lumemor/` (photos, derivatives, models)
and the `pgdata` volume (catalog). `docker compose down` keeps both;
`down -v` deletes the database volume. Back up both together — the
storage without the catalog is a pile of content-addressed blobs.

The development status applies to published images too: do not store
irreplaceable photos yet, and keep a 3-2-1 backup of everything you
love.
