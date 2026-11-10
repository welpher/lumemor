#!/usr/bin/env bash
# M2d acceptance: the chained M2 milestone acceptance -- the M1, M2a,
# M2b and M2c scripts run sequentially first (each manages its own
# compose stack and tears it down fully, including the database
# volume, before returning), then this script runs the M2 full-chain
# story of the web app against its own stack: the bind mount is
# prepared for the container's fixed uid 10001 (the m2c pattern: mkdir
# as the host user, then chown through the image itself running as
# root -- the sudo-less fresh-install contract documented on the bind
# mount in docker-compose.yml), the stack is built and started (the
# image builds the web app itself in its Node stage, so a clean
# checkout builds here), the server's health is awaited, and the
# Playwright suite under web/e2e walks the story: deep link redirect,
# first-run bootstrap and login, two uploads through the picker
# (including the trashed-conflict-restore flow), the wall's order,
# the viewer's keyboard walk, the Live Photo press on an API-seeded
# pair, batch hide/unhide, the viewer's permanent delete, the trash
# view's retention countdown and restore, and the sign-out redirect.
#
# The stack is always torn down including the database volume, so
# consecutive runs start from a clean state. The first failed step
# aborts with a non-zero exit status and a diagnostic message (the
# server's logs on a suite failure), and the teardown still runs. A
# passing run prints "M2D ACCEPTANCE PASS" as its final line.
#
# Requirements: docker with the compose plugin, curl (the chained
# scripts), pnpm 9.15 for web/ -- the script installs the web
# toolchain and Playwright's chromium itself (both idempotent
# no-ops once present; a browser download may want the usual proxy
# environment).

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

base_url="http://localhost:8244"
health_timeout_seconds=120
run_start="$(date +%s)"

fail() {
    echo "M2D ACCEPTANCE FAIL: $*" >&2
    exit 1
}

elapsed() {
    echo "$(( $(date +%s) - run_start ))"
}

# Brought-up flag for the teardown trap: a failed step must still
# remove the containers and the pgdata volume, otherwise a rerun would
# hit an instance that is already initialized.
stack_up=0

teardown() {
    if [ "$stack_up" -ne 0 ]; then
        stack_up=0
        echo "==> tearing down the stack and removing volumes"
        docker compose down -v --remove-orphans
    fi
}
trap teardown EXIT

echo "==> running the prior acceptances (each owns its own stack)"
# The chained regressions run sequentially, each fully down before the
# next starts: they share this repository's compose project name and
# the published port, so only one of them may be up at a time.
for script in acceptance_m1 acceptance_m2a acceptance_m2b acceptance_m2c; do
    echo "==> scripts/${script}.sh"
    bash "scripts/${script}.sh" \
        || fail "scripts/${script}.sh failed (see its output above)"
    echo "    ${script} passed after $(elapsed)s"
done

echo "==> installing the web toolchain and the e2e browser (idempotent)"
make web-install || fail "make web-install failed"
# --with-deps is the CI runner's job (it needs apt); a plain install
# covers macOS and any host that already carries the system
# libraries. The command checks the browser's presence and is a fast
# no-op once it is installed.
(cd web && pnpm exec playwright install chromium) \
    || fail "installing Playwright's chromium failed (a download may need the proxy environment)"

echo "==> resetting to a clean stack state"
# Start from a clean stack state so the story runs against a fresh,
# uninitialized instance even if a previous run was interrupted.
docker compose down -v --remove-orphans
stack_up=1

echo "==> preparing the storage bind mount for uid 10001 (sudo-less)"
# The container runs as the image's fixed uid/gid 10001 and the bind
# source must be writable by it before the server starts. Docker only
# creates a missing source as root:root, so the directory is created
# by the host user first and then chowned through the image itself
# running as root -- the fresh-install contract without sudo. The
# chown rides the app image, so it must exist: build it here (a cached
# no-op when the image is current; `up -d --build` below re-checks it).
mkdir -p lumemor
docker compose build || fail "docker compose build failed"
docker compose run --rm --no-deps --entrypoint /bin/sh -u 0 lumemor \
    -c 'chown -R 10001:10001 /var/lib/lumemor' \
    || fail "chown of ./lumemor to 10001:10001 through the image failed"

echo "==> building and starting the stack"
docker compose up -d --build || fail "docker compose up -d --build failed"

echo "==> waiting for GET /healthz (up to ${health_timeout_seconds}s)"
deadline="$(( $(date +%s) + health_timeout_seconds ))"
until curl -fsS "$base_url/healthz" >/dev/null 2>&1; do
    if [ "$(date +%s)" -ge "$deadline" ]; then
        docker compose logs --no-color db lumemor || true
        fail "GET /healthz did not succeed within ${health_timeout_seconds}s"
    fi
    sleep 2
done
echo "    server healthy after $(elapsed)s"

echo "==> running the Playwright suite (web/e2e, the M2 full-chain story)"
# The spec file is named: the suite grew a second spec,
# m3-models.spec.ts, which runs against the M3a
# acceptance's own stack -- the m2d chain pins its story here so both
# suites stay deterministic against the stacks built for them.
if ! (cd web && pnpm e2e e2e/m2-full-chain.spec.ts); then
    echo "==> server logs at the time of the failure:"
    docker compose logs --no-color --tail 200 lumemor || true
    fail "the Playwright suite failed (see the report above; artifacts under web/test-results/)"
fi

teardown
echo "==> acceptance flow completed in $(elapsed)s"
echo "M2D ACCEPTANCE PASS"
