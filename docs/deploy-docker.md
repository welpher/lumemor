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

## Before exposing it beyond your LAN

Port 8244 speaks **plain HTTP**. The wizard window (before the first
administrator exists) is unauthenticated by design and binds all
interfaces so family devices can reach it from the LAN. For anything
wider than the LAN: put a reverse proxy with TLS in front (Caddy/nginx
or a tunnels solution), and complete the wizard before the port is
reachable.

## Data safety

Everything durable lives in `./lumemor/` (photos, derivatives, models)
and the `pgdata` volume (catalog). `docker compose down` keeps both;
`down -v` deletes the database volume. Back up both together — the
storage without the catalog is a pile of content-addressed blobs.

The development status applies to published images too: do not store
irreplaceable photos yet, and keep a 3-2-1 backup of everything you
love.
