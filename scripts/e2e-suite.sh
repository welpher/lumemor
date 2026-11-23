#!/usr/bin/env bash
# The end-to-end suite: every Playwright spec drives ONE fresh
# instance. The first-run setup creates the spec's own administrator,
# so a spec may not share a stack with another -- the runner boots the
# repository's compose stack, waits for /healthz, runs one spec, tears
# the stack down (the database volume with it), and repeats.
#
# The storage bind (`./lumemor`) survives the resets on purpose: the
# model weights downloaded by the first spec that needs them stay warm,
# so the later AI specs re-verify instead of re-downloading.
#
#   scripts/e2e-suite.sh                    # the whole suite
#   E2E_SPECS="e5-map m2-full-chain" scripts/e2e-suite.sh
#   E2E_SKIP_BUILD=1 scripts/e2e-suite.sh   # reuse an already-built image
#
# Requires docker (compose v2), pnpm, and the web dependencies (make
# web-install). The stack listens on 8244 -- the address every spec
# pins. The release check is switched off at the deployment level (no
# scene reaches the outside world for it); the tile sources are stubbed
# inside the specs themselves. A restricted network passes the build
# passthroughs its compose file documents (LUMEMOR_BUILD_GOPROXY,
# LUMEMOR_BUILD_APT_MIRROR, LUMEMOR_BUILD_ONNXRUNTIME_URL, ...).
set -uo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

export LUMEMOR_UPDATE_CHECK_DISABLED="${LUMEMOR_UPDATE_CHECK_DISABLED:-1}"

if [[ -n "${E2E_SPECS:-}" ]]; then
	specs="$E2E_SPECS"
else
	specs="$(cd web && ls e2e/*.spec.ts | sed -e 's|^e2e/||' -e 's|\.spec\.ts$||' | sort | tr '\n' ' ')"
fi

if [[ -z "${specs// /}" ]]; then
	echo "e2e-suite: no spec found under web/e2e" >&2
	exit 1
fi

if [[ "${E2E_SKIP_BUILD:-0}" != "1" ]]; then
	echo "e2e-suite: building the stack image"
	if ! docker compose build; then
		echo "e2e-suite: the image build failed" >&2
		exit 1
	fi
fi

image="$(docker compose config --images | grep -v 'pgvector' | head -n 1)"
if [[ -z "$image" ]]; then
	echo "e2e-suite: could not resolve the stack image" >&2
	exit 1
fi

# The container's fixed uid 10001 must own the storage root and the
# writable source fixture on a Linux host (the bind sources are
# created by the caller otherwise).
mkdir -p ./lumemor
prepare_dir() {
	docker run --rm -v "$1:/mnt" --entrypoint /bin/chown "$image" -R 10001:10001 /mnt
}
prepare_dir "$repo_root/lumemor"
prepare_dir "$repo_root/web/e2e/fixtures/m3-source-rw"

cleanup() {
	docker compose down -v --remove-orphans >/dev/null 2>&1 || true
}
trap cleanup EXIT

start_stack() {
	cleanup
	if ! docker compose up -d; then
		return 1
	fi
	local attempt
	for attempt in $(seq 1 60); do
		if curl -fsS -o /dev/null http://localhost:8244/healthz; then
			return 0
		fi
		sleep 2
	done
	return 1
}

failures=""
for spec in $specs; do
	echo "=== e2e: $spec (fresh stack) ==="
	if ! start_stack; then
		echo "=== e2e: $spec FAILED (the stack never became healthy) ==="
		failures="$failures $spec(stack)"
		continue
	fi
	if (cd "$repo_root/web" && pnpm exec playwright test "e2e/$spec.spec.ts"); then
		echo "=== e2e: $spec passed ==="
	else
		echo "=== e2e: $spec FAILED ==="
		failures="$failures $spec"
	fi
done

if [[ -n "$failures" ]]; then
	echo "e2e-suite: failing specs:$failures" >&2
	exit 1
fi
echo "e2e-suite: all specs passed"